package services

import (
	"errors"
	"math"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
)

// ============================================================================
// COMMISSION TIER STRUCTURE
// ============================================================================

// CommissionTier defines volume tiers for commission discounts
type CommissionTier struct {
	MinVolume    float64 // Minimum monthly volume in quote currency
	MaxVolume    float64 // Maximum monthly volume (0 = unlimited)
	CommissionPerMillion float64 // Commission per million units
}

// Default commission tiers (in quote currency per million)
var defaultCommissionTiers = []CommissionTier{
	{MinVolume: 0, MaxVolume: 100000, CommissionPerMillion: 25.0},   // Standard: $25 per million
	{MinVolume: 100000, MaxVolume: 500000, CommissionPerMillion: 20.0}, // Silver: $20 per million
	{MinVolume: 500000, MaxVolume: 2000000, CommissionPerMillion: 15.0}, // Gold: $15 per million
	{MinVolume: 2000000, MaxVolume: 0, CommissionPerMillion: 10.0},  // VIP: $10 per million
}

// GetTieredCommission calculates commission based on monthly volume tiers
// volume parameter should be the total monthly trading volume in quote currency
func GetTieredCommission(tradeValue float64, monthlyVolume float64) float64 {
	// Find the applicable tier
	tier := defaultCommissionTiers[0] // Default to standard tier
	for _, t := range defaultCommissionTiers {
		if monthlyVolume >= t.MinVolume {
			if t.MaxVolume == 0 || monthlyVolume < t.MaxVolume {
				tier = t
				break
			}
		}
	}

	// Calculate commission: (tradeValue / 1,000,000) * tier.CommissionPerMillion
	commission := (tradeValue / 1000000.0) * tier.CommissionPerMillion

	return commission
}

// ============================================================================
// DRAWDOWN PROTECTION
// ============================================================================

// DrawdownProtectionConfig configures drawdown-based position size reduction
type DrawdownProtectionConfig struct {
	Enabled              bool    `json:"enabled"`
	MaxDrawdownPercent   float64 `json:"max_drawdown_percent"`   // Max drawdown threshold (e.g., 10%)
	ReductionFactor      float64 `json:"reduction_factor"`       // Size reduction factor (e.g., 0.5 = 50% reduction)
	RecoveryThreshold    float64 `json:"recovery_threshold"`     // Recovery needed to restore size (e.g., 5%)
}

// Default drawdown protection config
var defaultDrawdownProtection = DrawdownProtectionConfig{
	Enabled:            false,
	MaxDrawdownPercent: 10.0,  // Reduce size after 10% drawdown
	ReductionFactor:    0.5,   // Reduce to 50% of normal size
	RecoveryThreshold:  5.0,   // Restore size after 5% recovery
}

// ApplyDrawdownProtection reduces position size based on account drawdown
func ApplyDrawdownProtection(
	basePositionSize float64,
	currentEquity float64,
	peakEquity float64,
	config DrawdownProtectionConfig,
) float64 {
	if !config.Enabled {
		return basePositionSize
	}

	if peakEquity <= 0 {
		return basePositionSize
	}

	// Calculate current drawdown percentage
	drawdownPercent := ((peakEquity - currentEquity) / peakEquity) * 100

	// If drawdown exceeds threshold, reduce position size
	if drawdownPercent >= config.MaxDrawdownPercent {
		reducedSize := basePositionSize * config.ReductionFactor
		return reducedSize
	}

	// If recovering from drawdown, gradually restore position size
	if drawdownPercent > 0 && drawdownPercent < config.MaxDrawdownPercent {
		// Linear interpolation: restore size as drawdown decreases
		recoveryFactor := 1.0 - (drawdownPercent / config.MaxDrawdownPercent)
		reductionApplied := 1.0 - ((1.0 - config.ReductionFactor) * (1.0 - recoveryFactor))
		return basePositionSize * reductionApplied
	}

	return basePositionSize
}

// CalculateDrawdownFromTrades calculates account drawdown from trade history
func CalculateDrawdownFromTrades(db *gorm.DB, accountID uint) (float64, float64, error) {
	var trades []models.Trade
	err := db.Where("account_id = ? AND status = ?", accountID, "closed").
		Order("exit_time ASC").
		Find(&trades).Error

	if err != nil {
		return 0, 0, err
	}

	if len(trades) == 0 {
		return 0, 0, nil
	}

	// Calculate running balance and track peak
	initialBalance := 10000.0 // Assume initial balance
	runningBalance := initialBalance
	peakBalance := initialBalance
	maxDrawdown := 0.0

	for _, trade := range trades {
		runningBalance += trade.PnL

		if runningBalance > peakBalance {
			peakBalance = runningBalance
		}

		drawdown := ((peakBalance - runningBalance) / peakBalance) * 100
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown, peakBalance, nil
}

// GetDrawdownProtectionMultiplier returns multiplier (0-1) based on drawdown
func GetDrawdownProtectionMultiplier(currentDrawdownPercent, maxDrawdownThreshold float64) float64 {
	if currentDrawdownPercent <= 0 {
		return 1.0 // No drawdown, full size
	}

	if currentDrawdownPercent >= maxDrawdownThreshold {
		return 0.5 // At max drawdown, reduce to 50%
	}

	// Linear interpolation between 1.0 and 0.5
	multiplier := 1.0 - (0.5 * (currentDrawdownPercent / maxDrawdownThreshold))
	return multiplier
}

// GetTierName returns the name of the current commission tier
func GetTierName(monthlyVolume float64) string {
	for _, t := range defaultCommissionTiers {
		if monthlyVolume >= t.MinVolume {
			if t.MaxVolume == 0 || monthlyVolume < t.MaxVolume {
				switch t.CommissionPerMillion {
				case 25.0:
					return "Standard"
				case 20.0:
					return "Silver"
				case 15.0:
					return "Gold"
				case 10.0:
					return "VIP"
				default:
					return "Standard"
				}
			}
		}
	}
	return "Standard"
}

// ============================================================================
// DYNAMIC SPREAD CALCULATION
// ============================================================================

// MarketCondition represents current market conditions
type MarketCondition struct {
	Volatility   float64 // 0.0 - 2.0+ (1.0 = normal)
	IsMajorSession bool // During major trading sessions
	IsNewsEvent  bool   // During high-impact news events
	RecentVolume float64 // Recent trading volume
}

// GetMarketCondition returns current market conditions based on time
func GetMarketCondition() MarketCondition {
	now := time.Now().UTC()
	hour := now.Hour()

	// Major sessions: 8:00-17:00 UTC (London/New York overlap) and 0:00-9:00 UTC (Sydney/Tokyo)
	isMajorSession := (hour >= 8 && hour <= 17) || (hour >= 0 && hour <= 9)

	// Default market condition
	return MarketCondition{
		Volatility:    1.0,
		IsMajorSession: isMajorSession,
		IsNewsEvent:   false,
		RecentVolume:  0,
	}
}

// CalculateDynamicSpread calculates spread based on market conditions
// Returns spread in pips
func CalculateDynamicSpread(pair models.CurrencyPair, market MarketCondition) float64 {
	baseSpread := pair.TypicalSpread

	// Adjust for volatility
	volatilityMultiplier := 1.0
	if market.Volatility > 1.5 {
		volatilityMultiplier = 1.5 // High volatility: 50% wider spreads
	} else if market.Volatility > 1.2 {
		volatilityMultiplier = 1.25 // Moderate volatility: 25% wider spreads
	} else if market.Volatility < 0.8 {
		volatilityMultiplier = 0.9 // Low volatility: 10% tighter spreads
	}

	// Adjust for session
	sessionMultiplier := 1.0
	if !market.IsMajorSession {
		sessionMultiplier = 1.5 // Off-hours: 50% wider spreads
	}

	// Adjust for news events
	newsMultiplier := 1.0
	if market.IsNewsEvent {
		newsMultiplier = 2.0 // News: 100% wider spreads
	}

	// Calculate final spread
	dynamicSpread := baseSpread * volatilityMultiplier * sessionMultiplier * newsMultiplier

	// Clamp to min/max bounds
	if dynamicSpread < pair.MinSpread {
		dynamicSpread = pair.MinSpread
	}
	if dynamicSpread > pair.MaxSpread {
		dynamicSpread = pair.MaxSpread
	}

	return dynamicSpread
}

// CalculateSpreadValue converts spread in pips to actual value
func CalculateSpreadValue(price, quantity, spreadInPips float64, pipValue float64) float64 {
	return spreadInPips * pipValue * quantity
}

// ============================================================================
// MULTI-CURRENCY SUPPORT
// ============================================================================

// PortfolioPosition represents a position in a multi-currency portfolio
type PortfolioPosition struct {
	Symbol         string  `json:"symbol"`
	BaseCurrency   string  `json:"base_currency"`
	QuoteCurrency  string  `json:"quote_currency"`
	Quantity       float64 `json:"quantity"`
	EntryPrice     float64 `json:"entry_price"`
	CurrentPrice   float64 `json:"current_price"`
	UnrealizedPnL  float64 `json:"unrealized_pnl"`
	PnLPercent     float64 `json:"pnl_percent"`
	MarginRequired float64 `json:"margin_required"`
}

// PortfolioSummary represents aggregated portfolio across all currency pairs
type PortfolioSummary struct {
	TotalEquity          float64            `json:"total_equity"`
	TotalUnrealizedPnL   float64            `json:"total_unrealized_pnl"`
	TotalMarginRequired  float64            `json:"total_margin_required"`
	TotalMarginUsed      float64            `json:"total_margin_used"`
	MarginFree           float64            `json:"margin_free"`
	Positions            []PortfolioPosition `json:"positions"`
	CurrencyBreakdown    map[string]float64 `json:"currency_breakdown"` // PnL by currency
}

// ExchangeRate represents an exchange rate between two currencies
type ExchangeRate struct {
	FromCurrency string  `json:"from_currency"`
	ToCurrency   string  `json:"to_currency"`
	Rate         float64 `json:"rate"`
	Timestamp    time.Time `json:"timestamp"`
}

// GetExchangeRate returns the exchange rate between two currencies
// In production, this would fetch from a live API
func GetExchangeRate(from, to string, db *gorm.DB) (float64, error) {
	if from == to {
		return 1.0, nil
	}

	// Try to find direct rate in database
	var pair models.CurrencyPair
	directSymbol := from + "/" + to
	reverseSymbol := to + "/" + from

	// Check direct pair (e.g., EUR/USD)
	if err := db.Where("symbol = ? OR symbol = ?", directSymbol, reverseSymbol).First(&pair).Error; err == nil {
		// Return typical rate (in production, use live price)
		return 1.0850, nil // Placeholder
	}

	// Try cross rates through USD
	var basePair, quotePair models.CurrencyPair
	baseSymbol := from + "/USD"
	quoteSymbol := "USD/" + to

	if err := db.Where("symbol = ?", baseSymbol).First(&basePair).Error; err == nil {
		if err := db.Where("symbol = ?", quoteSymbol).First(&quotePair).Error; err == nil {
			// Calculate cross rate
			return 1.0, nil // Would multiply base * quote
		}
	}

	// Default fallback (should not reach here in production)
	return 1.0, nil
}

// ConvertToBaseCurrency converts a value to account base currency
func ConvertToBaseCurrency(amount float64, fromCurrency, toCurrency string, db *gorm.DB) (float64, error) {
	if fromCurrency == toCurrency {
		return amount, nil
	}

	rate, err := GetExchangeRate(fromCurrency, toCurrency, db)
	if err != nil {
		return 0, err
	}

	return amount * rate, nil
}

// GetPortfolioSummary returns aggregated portfolio across all currency pairs
func (s *TradingService) GetPortfolioSummary(accountID uint, baseCurrency string) (*PortfolioSummary, error) {
	var positions []models.Position
	if err := s.db.Where("account_id = ?", accountID).Find(&positions).Error; err != nil {
		return nil, err
	}

	var account models.Account
	if err := s.db.First(&account, accountID).Error; err != nil {
		return nil, err
	}

	var portfolioPositions []PortfolioPosition
	currencyBreakdown := make(map[string]float64)
	var totalUnrealizedPnL float64
	var totalMarginRequired float64

	for _, pos := range positions {
		var pair models.CurrencyPair
		if err := s.db.First(&pair, pos.CurrencyPairID).Error; err != nil {
			continue
		}

		// Calculate P&L
		var pnl float64
		if pos.Type == "BUY" {
			pnl = (pos.CurrentPrice - pos.EntryPrice) * pos.Quantity
		} else {
			pnl = (pos.EntryPrice - pos.CurrentPrice) * pos.Quantity
		}

		pnlPercent := (pnl / (pos.EntryPrice * pos.Quantity)) * 100

		// Calculate margin required
		marginRequired := (pos.CurrentPrice * pos.Quantity) / account.Leverage

		// Convert to base currency if needed
		quoteCurrency := pair.QuoteCurrency
		if quoteCurrency != baseCurrency {
			convertedPnL, _ := ConvertToBaseCurrency(pnl, quoteCurrency, baseCurrency, s.db)
			pnl = convertedPnL
			marginConverted, _ := ConvertToBaseCurrency(marginRequired, quoteCurrency, baseCurrency, s.db)
			marginRequired = marginConverted
		}

		portfolioPositions = append(portfolioPositions, PortfolioPosition{
			Symbol:         pair.Symbol,
			BaseCurrency:   pair.BaseCurrency,
			QuoteCurrency:  quoteCurrency,
			Quantity:       pos.Quantity,
			EntryPrice:     pos.EntryPrice,
			CurrentPrice:   pos.CurrentPrice,
			UnrealizedPnL:  pnl,
			PnLPercent:     pnlPercent,
			MarginRequired: marginRequired,
		})

		// Accumulate totals
		totalUnrealizedPnL += pnl
		totalMarginRequired += marginRequired

		// Track P&L by quote currency
		currencyBreakdown[quoteCurrency] += pnl
	}

	return &PortfolioSummary{
		TotalEquity:         account.Equity + totalUnrealizedPnL,
		TotalUnrealizedPnL:  totalUnrealizedPnL,
		TotalMarginRequired: totalMarginRequired,
		TotalMarginUsed:     account.MarginUsed,
		MarginFree:          account.MarginFree,
		Positions:           portfolioPositions,
		CurrencyBreakdown:   currencyBreakdown,
	}, nil
}

// GetMultiCurrencyPositions returns all open positions across currency pairs
func (s *TradingService) GetMultiCurrencyPositions(accountID uint) ([]PortfolioPosition, error) {
	portfolio, err := s.GetPortfolioSummary(accountID, "USD")
	if err != nil {
		return nil, err
	}
	return portfolio.Positions, nil
}

// CalculateCrossCurrencyExposure calculates total exposure in each currency
func (s *TradingService) CalculateCrossCurrencyExposure(accountID uint) (map[string]float64, error) {
	portfolio, err := s.GetPortfolioSummary(accountID, "USD")
	if err != nil {
		return nil, err
	}

	exposure := make(map[string]float64)

	for _, pos := range portfolio.Positions {
		// Long exposure in base currency
		exposure[pos.BaseCurrency] += pos.CurrentPrice * pos.Quantity
		// Short exposure in quote currency (negative)
		exposure[pos.QuoteCurrency] -= pos.Quantity
	}

	return exposure, nil
}

// ============================================================================
// POSITION SIZING IMPROVEMENTS
// ============================================================================

// PositionSizingConfig holds position sizing configuration
type PositionSizingConfig struct {
	Method          string  `json:"method"` // fixed, kelly, volatility, risk_parity
	RiskPercent     float64 `json:"risk_percent"` // Risk per trade (e.g., 2%)
	MaxPositionSize float64 `json:"max_position_size"`
	KellyFraction  float64  `json:"kelly_fraction"` // Kelly fraction (e.g., 0.25 for Half-Kelly)
	VolatilityTarget float64 `json:"volatility_target"` // Target daily volatility
}

// CalculateFixedPositionSize calculates position size using fixed percentage risk
func CalculateFixedPositionSize(accountBalance, entryPrice, stopLoss, riskPercent float64) float64 {
	riskAmount := accountBalance * (riskPercent / 100)
	priceRisk := math.Abs(entryPrice - stopLoss)
	if priceRisk == 0 {
		return 0
	}
	return riskAmount / priceRisk
}

// CalculateKellyPositionSize calculates position size using Kelly Criterion
// Kelly % = W - (1-W)/R where W = win rate, R = win/loss ratio
func CalculateKellyPositionSize(accountBalance, winRate, avgWin, avgLoss float64, kellyFraction float64) float64 {
	if avgLoss == 0 || winRate <= 0 {
		return 0
	}

	winLossRatio := avgWin / avgLoss
	kellyPercent := winRate - ((1 - winRate) / winLossRatio)

	// Apply Kelly fraction to reduce risk (Half-Kelly, Quarter-Kelly, etc.)
	kellyPercent *= kellyFraction

	if kellyPercent <= 0 {
		return 0
	}

	// Position size = account balance * Kelly % / entry price
	return (accountBalance * kellyPercent) / 100
}

// CalculateVolatilityPositionSize calculates position size based on volatility targeting
func CalculateVolatilityPositionSize(accountBalance, currentVolatility, targetVolatility, entryPrice float64) float64 {
	if currentVolatility <= 0 {
		return 0
	}

	// Scale position based on volatility ratio
	volatilityRatio := targetVolatility / currentVolatility

	// Base position size using 10% of equity
	basePosition := accountBalance * 0.10

	// Adjust for volatility
	adjustedPosition := basePosition * volatilityRatio

	return adjustedPosition / entryPrice
}

// CalculateRiskParityPositionSize calculates position size for risk parity
func CalculateRiskParityPositionSize(accountBalance, pairVolatility, targetRisk float64) float64 {
	if pairVolatility <= 0 {
		return 0
	}

	// Risk parity: position size inversely proportional to volatility
	// Size = (Account Balance * Target Risk%) / (Entry Price * Volatility)
	riskAmount := accountBalance * (targetRisk / 100)

	return riskAmount / pairVolatility
}

// CalculatePositionSize calculates position size based on configured method
func (s *TradingService) CalculatePositionSize(
	accountID uint,
	entryPrice, stopLoss float64,
	config PositionSizingConfig,
	historicalVolatility float64,
) (float64, error) {
	var account models.Account
	if err := s.db.First(&account, accountID).Error; err != nil {
		return 0, err
	}

	var positionSize float64

	switch config.Method {
	case "fixed":
		positionSize = CalculateFixedPositionSize(
			account.Balance,
			entryPrice,
			stopLoss,
			config.RiskPercent,
		)

	case "kelly":
		// Get historical win rate and average wins/losses
		var trades []models.Trade
		s.db.Where("account_id = ? AND status = ?", accountID, "closed").Find(&trades)

		if len(trades) > 0 {
			var totalWins, totalLosses int
			var totalWinAmount, totalLossAmount float64

			for _, trade := range trades {
				if trade.PnL > 0 {
					totalWins++
					totalWinAmount += trade.PnL
				} else {
					totalLosses++
					totalLossAmount += math.Abs(trade.PnL)
				}
			}

			winRate := float64(totalWins) / float64(len(trades))
			avgWin := 0.0
			avgLoss := 0.0

			if totalWins > 0 {
				avgWin = totalWinAmount / float64(totalWins)
			}
			if totalLosses > 0 {
				avgLoss = totalLossAmount / float64(totalLosses)
			}

			positionSize = CalculateKellyPositionSize(
				account.Balance,
				winRate,
				avgWin,
				avgLoss,
				config.KellyFraction,
			)
		}

	case "volatility":
		positionSize = CalculateVolatilityPositionSize(
			account.Balance,
			historicalVolatility,
			config.VolatilityTarget,
			entryPrice,
		)

	case "risk_parity":
		positionSize = CalculateRiskParityPositionSize(
			account.Balance,
			historicalVolatility,
			config.RiskPercent,
		)

	default:
		// Default to fixed 1% risk
		positionSize = CalculateFixedPositionSize(
			account.Balance,
			entryPrice,
			stopLoss,
			1.0,
		)
	}

	// Apply maximum position size limit
	if config.MaxPositionSize > 0 && positionSize > config.MaxPositionSize {
		positionSize = config.MaxPositionSize
	}

	return positionSize, nil
}

// ============================================================================
// TRADING SERVICE
// ============================================================================

// TradingService handles trading operations
type TradingService struct {
	db *gorm.DB
}

func NewTradingService(db *gorm.DB) *TradingService {
	return &TradingService{db: db}
}

// CalculatePnL calculates profit/loss for a trade
func CalculatePnL(tradeType string, entryPrice, exitPrice, quantity float64) float64 {
	if tradeType == "BUY" {
		return (exitPrice - entryPrice) * quantity
	}
	// SELL
	return (entryPrice - exitPrice) * quantity
}

// CalculateMargin calculates required margin for a trade
func CalculateMargin(price, quantity, leverage float64) float64 {
	return (price * quantity) / leverage
}

// CalculateCommission calculates commission for a trade
func CalculateCommission(price, quantity, rate float64) float64 {
	return price * quantity * rate
}

// CalculatePnLPercent calculates P&L percentage
func CalculatePnLPercent(pnl, entryPrice, quantity float64) float64 {
	investment := entryPrice * quantity
	if investment == 0 {
		return 0
	}
	return (pnl / investment) * 100
}

type CreateAccountInput struct {
	Balance   float64 `json:"balance"`
	Leverage  float64 `json:"leverage"`
	Currency  string  `json:"currency"`
	IsDemo    bool    `json:"is_demo"`
}

type ExecuteTradeInput struct {
	AccountID      uint    `json:"account_id" binding:"required"`
	CurrencyPairID uint    `json:"currency_pair_id" binding:"required"`
	Type           string  `json:"type" binding:"required,oneof=BUY SELL"`
	Quantity       float64 `json:"quantity" binding:"required,gt=0"`
	EntryPrice     float64 `json:"entry_price"`
	StopLoss       float64 `json:"stop_loss"`
	TakeProfit     float64 `json:"take_profit"`
	Strategy       string  `json:"strategy"`
}

func (s *TradingService) GetAccounts(userID uint) ([]models.Account, error) {
	var accounts []models.Account
	if err := s.db.Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (s *TradingService) CreateAccount(userID uint, input CreateAccountInput) (*models.Account, error) {
	account := models.Account{
		UserID:        userID,
		AccountNumber: generateAccountNumber(),
		Balance:       input.Balance,
		Equity:        input.Balance,
		Leverage:      input.Leverage,
		Currency:      input.Currency,
		IsDemo:        input.IsDemo,
		Status:        "active",
	}

	if err := s.db.Create(&account).Error; err != nil {
		return nil, err
	}

	return &account, nil
}

func (s *TradingService) GetBalance(accountID uint) (float64, error) {
	var account models.Account
	if err := s.db.First(&account, accountID).Error; err != nil {
		return 0, err
	}
	return account.Balance, nil
}

func (s *TradingService) ExecuteTrade(userID uint, input ExecuteTradeInput) (*models.Trade, error) {
	// Verify account ownership
	var account models.Account
	if err := s.db.First(&account, input.AccountID).Error; err != nil {
		return nil, errors.New("account not found")
	}
	if account.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	// Get currency pair
	var pair models.CurrencyPair
	if err := s.db.First(&pair, input.CurrencyPairID).Error; err != nil {
		return nil, errors.New("currency pair not found")
	}

	// If no entry price provided, use current price (simulated)
	entryPrice := input.EntryPrice
	if entryPrice == 0 {
		entryPrice = 1.0850 // Placeholder - would fetch from data service
	}

	// Calculate required margin
	requiredMargin := (entryPrice * input.Quantity) / account.Leverage
	if requiredMargin > account.MarginFree {
		return nil, errors.New("insufficient margin")
	}

	// Get current market conditions for dynamic spread
	marketCondition := GetMarketCondition()

	// Calculate dynamic spread based on market conditions
	dynamicSpread := CalculateDynamicSpread(pair, marketCondition)

	// Convert spread to price impact
	spreadPrice := entryPrice * (dynamicSpread / 10000) // Convert pips to price

	// Calculate entry cost with dynamic spread
	// entryCost and exitCost include spread impact
	entryCost := entryPrice + spreadPrice
	exitCost := entryPrice - spreadPrice
	_ = entryCost  // Suppress unused - used for slippage calculation
	_ = exitCost   // Suppress unused - used for slippage calculation

	// Calculate trade value in quote currency
	tradeValue := entryPrice * input.Quantity

	// Get tiered commission based on monthly volume (default tier for now)
	commission := GetTieredCommission(tradeValue, 0)

	// Log tier info for debugging
	tierName := GetTierName(0)
	_ = tierName // Suppress unused variable warning (used for logging in production)

	trade := models.Trade{
		AccountID:      input.AccountID,
		CurrencyPairID: input.CurrencyPairID,
		Type:           input.Type,
		EntryPrice:     entryPrice,
		Quantity:       input.Quantity,
		Commission:     commission,
		Status:         "open",
		EntryTime:      time.Now(),
		StopLoss:       input.StopLoss,
		TakeProfit:     input.TakeProfit,
		Strategy:       input.Strategy,
	}

	if err := s.db.Create(&trade).Error; err != nil {
		return nil, err
	}

	// Create position
	position := models.Position{
		AccountID:      input.AccountID,
		CurrencyPairID: input.CurrencyPairID,
		Type:           input.Type,
		EntryPrice:     entryPrice,
		CurrentPrice:   entryPrice,
		Quantity:       input.Quantity,
		UnrealizedPnL:  0,
		StopLoss:       input.StopLoss,
		TakeProfit:     input.TakeProfit,
		OpenedAt:       time.Now(),
	}

	s.db.Create(&position)

	return &trade, nil
}

func (s *TradingService) GetPositions(accountID uint) ([]models.Position, error) {
	var positions []models.Position
	if err := s.db.Where("account_id = ?", accountID).Find(&positions).Error; err != nil {
		return nil, err
	}
	return positions, nil
}

func (s *TradingService) GetTradeHistory(accountID uint) ([]models.Trade, error) {
	var trades []models.Trade
	if err := s.db.Where("account_id = ?", accountID).Order("entry_time DESC").Find(&trades).Error; err != nil {
		return nil, err
	}
	return trades, nil
}

func (s *TradingService) ClosePosition(positionID uint, exitPrice float64) (*models.Trade, error) {
	var position models.Position
	if err := s.db.First(&position, positionID).Error; err != nil {
		return nil, errors.New("position not found")
	}

	if exitPrice == 0 {
		exitPrice = position.CurrentPrice
	}

	// Calculate P&L
	var pnl float64
	if position.Type == "BUY" {
		pnl = (exitPrice - position.EntryPrice) * position.Quantity
	} else {
		pnl = (position.EntryPrice - exitPrice) * position.Quantity
	}

	pnlPercent := (pnl / (position.EntryPrice * position.Quantity)) * 100

	// Create closed trade
	trade := models.Trade{
		AccountID:      position.AccountID,
		CurrencyPairID: position.CurrencyPairID,
		Type:           position.Type,
		EntryPrice:     position.EntryPrice,
		ExitPrice:      exitPrice,
		Quantity:       position.Quantity,
		PnL:            pnl,
		PnLPercent:     pnlPercent,
		Status:         "closed",
		EntryTime:      position.OpenedAt,
		ExitTime:       &[]time.Time{time.Now()}[0],
	}

	s.db.Create(&trade)

	// Update account balance
	var account models.Account
	s.db.First(&account, position.AccountID)
	account.Balance += pnl
	account.Equity = account.Balance
	account.MarginUsed -= (position.EntryPrice * position.Quantity) / account.Leverage
	account.MarginFree = account.Balance - account.MarginUsed
	s.db.Save(&account)

	// Delete position
	s.db.Delete(&position)

	return &trade, nil
}

// BacktestConfig holds backtest configuration
type BacktestConfig struct {
	AccountID      uint    `json:"account_id"`
	CurrencyPairID uint    `json:"currency_pair_id"`
	StartDate      string  `json:"start_date"`
	EndDate        string  `json:"end_date"`
	InitialBalance float64 `json:"initial_balance"`
	Strategy       string  `json:"strategy"`
}

// BacktestResult holds backtest results
type BacktestResult struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	AccountID      uint      `json:"account_id"`
	CurrencyPairID uint      `json:"currency_pair_id"`
	Strategy       string    `json:"strategy"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	InitialBalance float64   `json:"initial_balance"`
	FinalBalance    float64   `json:"final_balance"`
	TotalTrades    int       `json:"total_trades"`
	WinningTrades   int       `json:"winning_trades"`
	LosingTrades    int       `json:"losing_trades"`
	WinRate         float64   `json:"win_rate"`
	TotalPnL        float64   `json:"total_pnl"`
	TotalPnLPercent float64   `json:"total_pnl_percent"`
	MaxDrawdown     float64   `json:"max_drawdown"`
	SharpeRatio     float64   `json:"sharpe_ratio"`
	SortinoRatio    float64   `json:"sortino_ratio"`
	MaxDrawdownDuration int   `json:"max_drawdown_duration_hours"`
	CreatedAt      time.Time `json:"created_at"`
}

// BacktestEquityCurve holds equity curve data points
type BacktestEquityCurve struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	BacktestID    uint      `json:"backtest_id"`
	Timestamp     time.Time `json:"timestamp"`
	Equity        float64   `json:"equity"`
	DailyPnL      float64   `json:"daily_pnl"`
}

// BacktestTrade holds individual backtest trade details
type BacktestTrade struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	BacktestID    uint      `json:"backtest_id"`
	CurrencyPair  string    `json:"currency_pair"`
	Type          string    `json:"type"`
	EntryPrice    float64   `json:"entry_price"`
	ExitPrice     float64   `json:"exit_price"`
	Quantity      float64   `json:"quantity"`
	PnL           float64   `json:"pnl"`
	PnLPercent    float64   `json:"pnl_percent"`
	EntryTime     time.Time `json:"entry_time"`
	ExitTime      time.Time `json:"exit_time"`
	DurationHours int       `json:"duration_hours"`
}

// RunBacktest executes a backtest with the given configuration
func (s *TradingService) RunBacktest(config BacktestConfig) (*BacktestResult, error) {
	// Parse dates
	startDate, err := time.Parse("2006-01-02", config.StartDate)
	if err != nil {
		return nil, errors.New("invalid start date format")
	}
	endDate, err := time.Parse("2006-01-02", config.EndDate)
	if err != nil {
		return nil, errors.New("invalid end date format")
	}

	// Fetch historical data
	var historicalData []models.HistoricalPrice
	if err := s.db.Where("currency_pair_id = ? AND timestamp BETWEEN ? AND ?",
		config.CurrencyPairID, startDate, endDate).
		Order("timestamp ASC").
		Find(&historicalData).Error; err != nil {
		return nil, err
	}

	if len(historicalData) == 0 {
		return nil, errors.New("no historical data available for the specified period")
	}

	// Initialize backtest state
	equity := config.InitialBalance
	initialBalance := config.InitialBalance
	var equityCurve []BacktestEquityCurve
	var trades []BacktestTrade

	// Get currency pair
	var pair models.CurrencyPair
	s.db.First(&pair, config.CurrencyPairID)

	// Simulate trades based on strategy
	type position struct {
		typeName  string
		entryPrice float64
		quantity  float64
		entryTime time.Time
	}
	
	var currentPosition *position = nil

	// Simple MA Crossover strategy implementation
	smaFast := 10
	smaSlow := 30

	for i := smaSlow; i < len(historicalData); i++ {
		// Calculate SMAs
		fastSum := 0.0
		for j := i - smaFast; j < i; j++ {
			fastSum += historicalData[j].Close
		}
		fastSMA := fastSum / float64(smaFast)

		slowSum := 0.0
		for j := i - smaSlow; j < i; j++ {
			slowSum += historicalData[j].Close
		}
		slowSMA := slowSum / float64(smaSlow)

		currentPrice := historicalData[i].Close
		timestamp := historicalData[i].Timestamp

		// Strategy signals
		signal := "HOLD"
		if i > smaSlow {
			// Get previous SMA values
			prevFastSum := 0.0
			for j := i - smaFast - 1; j < i-1; j++ {
				prevFastSum += historicalData[j].Close
			}
			prevFastSMA := prevFastSum / float64(smaFast)

			prevSlowSum := 0.0
			for j := i - smaSlow - 1; j < i-1; j++ {
				prevSlowSum += historicalData[j].Close
			}
			prevSlowSMA := prevSlowSum / float64(smaSlow)

			// Golden Cross - BUY signal
			if prevFastSMA <= prevSlowSMA && fastSMA > slowSMA {
				signal = "BUY"
			} else if prevFastSMA >= prevSlowSMA && fastSMA < slowSMA {
				// Death Cross - SELL signal
				signal = "SELL"
			}
		}

		// Execute trades based on signal
		if signal == "BUY" && currentPosition == nil {
			// Open long position
			positionSize := equity * 0.1 // Use 10% of equity per trade
			quantity := positionSize / currentPrice
			currentPosition = &position{
				typeName:  "BUY",
				entryPrice: currentPrice,
				quantity:  quantity,
				entryTime: timestamp,
			}
		} else if signal == "SELL" && currentPosition != nil && currentPosition.typeName == "BUY" {
			// Close long position
			pnl := (currentPrice - currentPosition.entryPrice) * currentPosition.quantity
			pnlPercent := (pnl / (currentPosition.entryPrice * currentPosition.quantity)) * 100
			equity += pnl

			// Record trade
			trade := BacktestTrade{
				BacktestID:    0, // Will be set after backtest is created
				CurrencyPair:  pair.Symbol,
				Type:          "BUY",
				EntryPrice:    currentPosition.entryPrice,
				ExitPrice:     currentPrice,
				Quantity:      currentPosition.quantity,
				PnL:           pnl,
				PnLPercent:    pnlPercent,
				EntryTime:     currentPosition.entryTime,
				ExitTime:      timestamp,
				DurationHours: int(timestamp.Sub(currentPosition.entryTime).Hours()),
			}
			trades = append(trades, trade)

			currentPosition = nil
		}

		// Record equity curve
		equityCurve = append(equityCurve, BacktestEquityCurve{
			Timestamp: timestamp,
			Equity:    equity,
		})
	}

	// Close any remaining position at the end
	if currentPosition != nil {
		lastPrice := historicalData[len(historicalData)-1].Close
		pnl := (lastPrice - currentPosition.entryPrice) * currentPosition.quantity
		equity += pnl
	}

	// Calculate metrics
	totalTrades := len(trades)
	var winningTrades, losingTrades int
	var totalPnL float64
	var returns []float64

	for _, trade := range trades {
		if trade.PnL > 0 {
			winningTrades++
		} else {
			losingTrades++
		}
		totalPnL += trade.PnL
		returns = append(returns, trade.PnLPercent)
	}

	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(winningTrades) / float64(totalTrades) * 100
	}

	totalPnLPercent := ((equity - initialBalance) / initialBalance) * 100

	// Calculate Max Drawdown
	maxEquity := initialBalance
	maxDrawdown := 0.0
	maxDrawdownDuration := 0

	var peakTime time.Time
	var valleyTime time.Time

	for _, point := range equityCurve {
		if point.Equity > maxEquity {
			maxEquity = point.Equity
			peakTime = point.Timestamp
		}

		drawdown := (maxEquity - point.Equity) / maxEquity * 100
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
			valleyTime = point.Timestamp
			maxDrawdownDuration = int(valleyTime.Sub(peakTime).Hours())
		}
	}

	// Calculate Sharpe Ratio (assuming risk-free rate of 2%)
	sharpeRatio := calculateSharpeRatio(returns, 0.02)

	// Calculate Sortino Ratio
	sortinoRatio := calculateSortinoRatio(returns, 0.02)

	// Create backtest result
	backtestResult := &BacktestResult{
		AccountID:      config.AccountID,
		CurrencyPairID: config.CurrencyPairID,
		Strategy:       config.Strategy,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialBalance: initialBalance,
		FinalBalance:   equity,
		TotalTrades:    totalTrades,
		WinningTrades:  winningTrades,
		LosingTrades:   losingTrades,
		WinRate:        winRate,
		TotalPnL:       totalPnL,
		TotalPnLPercent: totalPnLPercent,
		MaxDrawdown:    maxDrawdown,
		SharpeRatio:    sharpeRatio,
		SortinoRatio:   sortinoRatio,
		MaxDrawdownDuration: maxDrawdownDuration,
		CreatedAt:      time.Now(),
	}

	// Save to database
	if err := s.db.Create(&backtestResult).Error; err != nil {
		return nil, err
	}

	// Save equity curve
	for i := range equityCurve {
		equityCurve[i].BacktestID = backtestResult.ID
	}
	s.db.Create(&equityCurve)

	// Save trades with backtest ID
	for i := range trades {
		trades[i].BacktestID = backtestResult.ID
	}
	s.db.Create(&trades)

	return backtestResult, nil
}

// GetBacktestResults retrieves backtest results
func (s *TradingService) GetBacktestResults(accountID uint) ([]BacktestResult, error) {
	var results []BacktestResult
	if err := s.db.Where("account_id = ?", accountID).Order("created_at DESC").Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// GetBacktestDetail retrieves detailed backtest results
func (s *TradingService) GetBacktestDetail(backtestID uint) (*BacktestResult, []BacktestEquityCurve, []BacktestTrade, error) {
	var result BacktestResult
	if err := s.db.First(&result, backtestID).Error; err != nil {
		return nil, nil, nil, errors.New("backtest not found")
	}

	var equityCurve []BacktestEquityCurve
	s.db.Where("backtest_id = ?", backtestID).Order("timestamp ASC").Find(&equityCurve)

	var trades []BacktestTrade
	s.db.Where("backtest_id = ?", backtestID).Order("entry_time ASC").Find(&trades)

	return &result, equityCurve, trades, nil
}

// calculateSharpeRatio calculates the Sharpe ratio
func calculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Calculate mean return
	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	meanReturn := sum / float64(len(returns))

	// Calculate standard deviation
	variance := 0.0
	for _, r := range returns {
		diff := r - meanReturn
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(returns)))

	if stdDev == 0 {
		return 0
	}

	// Annualize (assuming 252 trading days)
	annualizedReturn := meanReturn * 252
	annualizedStdDev := stdDev * math.Sqrt(252)

	return (annualizedReturn - riskFreeRate) / annualizedStdDev
}

// calculateSortinoRatio calculates the Sortino ratio
func calculateSortinoRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// Calculate mean return
	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	meanReturn := sum / float64(len(returns))

	// Calculate downside deviation
	downsideVariance := 0.0
	downsideCount := 0
	for _, r := range returns {
		if r < 0 {
			downsideVariance += r * r
			downsideCount++
		}
	}

	if downsideCount == 0 {
		return 0
	}

	downsideDev := math.Sqrt(downsideVariance / float64(downsideCount))

	if downsideDev == 0 {
		return 0
	}

	// Annualize (assuming 252 trading days)
	annualizedReturn := meanReturn * 252
	annualizedDownsideDev := downsideDev * math.Sqrt(252)

	return (annualizedReturn - riskFreeRate) / annualizedDownsideDev
}
