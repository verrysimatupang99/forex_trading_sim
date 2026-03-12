package backtest

import (
	"errors"
	"math"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/services/strategies"
)

// BacktestConfig holds configuration for a backtest run
type BacktestConfig struct {
	Name            string                 `json:"name"`
	StrategyName    string                 `json:"strategy_name"`
	Parameters      map[string]interface{}  `json:"parameters"`
	CurrencyPairID  uint                   `json:"currency_pair_id"`
	Timeframe       string                 `json:"timeframe"` // 1m, 5m, 15m, 1h, 4h, 1d
	StartDate       time.Time              `json:"start_date"`
	EndDate         time.Time              `json:"end_date"`
	InitialCapital  float64                 `json:"initial_capital"`
	Commission      float64                 `json:"commission"`     // Commission per trade (e.g., 0.0002 = 2 pips)
	SlippagePips    float64                 `json:"slippage_pips"`  // Slippage in pips
	SpreadPips      float64                 `json:"spread_pips"`    // Spread in pips
	PositionSizing  PositionSizingConfig    `json:"position_sizing"`
	StopLossPips    float64                 `json:"stop_loss_pips"`   // Default stop loss in pips (0 = disabled)
	TakeProfitPips  float64                 `json:"take_profit_pips"` // Default take profit in pips (0 = disabled)
}

// PositionSizingConfig defines how position size is calculated
type PositionSizingConfig struct {
	Mode          string  `json:"mode"`           // "FIXED_LOT" or "PERCENT_EQUITY"
	FixedLot      float64 `json:"fixed_lot"`     // Fixed lot size
	EquityPercent float64 `json:"equity_percent"` // Percentage of equity to risk per trade (e.g., 0.02 = 2%)
	StopLossPips  float64 `json:"stop_loss_pips"` // Used for risk-based sizing
}

// BacktestResult holds the results of a backtest run
type BacktestResult struct {
	ID             uint                   `json:"id"`
	Config         BacktestConfig         `json:"config"`
	Metrics        PerformanceMetrics    `json:"metrics"`
	Trades         []strategies.BacktestTrade `json:"trades"`
	EquityCurve    []EquityPoint         `json:"equity_curve"`
	DailyReturns   []DailyReturn         `json:"daily_returns"`
	StartTime      time.Time             `json:"start_time"`
	EndTime        time.Time             `json:"end_time"`
	Status         string                 `json:"status"`
}

// EquityPoint represents a point in the equity curve
type EquityPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Equity    float64   `json:"equity"`
	Drawdown  float64   `json:"drawdown"`
}

// DailyReturn represents daily return data
type DailyReturn struct {
	Date        time.Time `json:"date"`
	Return      float64   `json:"return"`
	Cumulative  float64   `json:"cumulative"`
}

// PerformanceMetrics holds all performance metrics
type PerformanceMetrics struct {
	TotalReturn       float64   `json:"total_return"`
	AnnualizedReturn  float64   `json:"annualized_return"`
	SharpeRatio       float64   `json:"sharpe_ratio"`
	SortinoRatio      float64   `json:"sortino_ratio"`
	MaxDrawdown       float64   `json:"max_drawdown"`
	MaxDrawdownDuration int     `json:"max_drawdown_duration_days"`
	WinRate           float64   `json:"win_rate"`
	ProfitFactor      float64   `json:"profit_factor"`
	AverageWin        float64   `json:"average_win"`
	AverageLoss       float64   `json:"average_loss"`
	Expectancy        float64   `json:"expectancy"`
	CalmarRatio       float64   `json:"calmar_ratio"`
	TotalTrades      int       `json:"total_trades"`
	WinningTrades    int       `json:"winning_trades"`
	LosingTrades     int       `json:"losing_trades"`
	GrossProfit      float64   `json:"gross_profit"`
	GrossLoss        float64   `json:"gross_loss"`
}

// BacktestEngine orchestrates the backtesting simulation
type BacktestEngine struct {
	db       *gorm.DB
	config   BacktestConfig
	strategy strategies.Strategy
	portfolio Portfolio
	replayer *DataReplayer
	executor *TradeExecutor
}

// Portfolio holds the current portfolio state
type Portfolio struct {
	Equity        float64                 // Current equity
	Cash          float64                 // Available cash
	OpenPositions []OpenPosition          // Currently open positions
	Trades        []strategies.BacktestTrade // Closed trades
	EquityCurve   []EquityPoint           // Equity curve over time
}

// OpenPosition represents an open position
type OpenPosition struct {
	ID            string                  `json:"id"`
	Type          strategies.SignalType   `json:"type"` // BUY or SELL
	EntryPrice    float64                 `json:"entry_price"`
	Quantity      float64                 `json:"quantity"`
	StopLoss      float64                 `json:"stop_loss"`
	TakeProfit    float64                 `json:"take_profit"`
	OpenedAt      time.Time               `json:"opened_at"`
	CurrencyPair  string                  `json:"currency_pair"`
}

// NewBacktestEngine creates a new backtest engine
func NewBacktestEngine(db *gorm.DB, config BacktestConfig) (*BacktestEngine, error) {
	// Create strategy instance
	strategy, err := strategies.CreateStrategy(config.StrategyName, config.Parameters)
	if err != nil {
		return nil, errors.New("failed to create strategy: " + err.Error())
	}

	// Validate strategy parameters
	if err := strategy.ValidateParameters(); err != nil {
		return nil, errors.New("invalid strategy parameters: " + err.Error())
	}

	engine := &BacktestEngine{
		db:       db,
		config:   config,
		strategy: strategy,
		portfolio: Portfolio{
			Equity:      config.InitialCapital,
			Cash:        config.InitialCapital,
			EquityCurve: make([]EquityPoint, 0),
		},
	}

	// Initialize replayer
	engine.replayer = NewDataReplayer(db, config.CurrencyPairID, config.Timeframe, config.StartDate, config.EndDate)

	// Initialize executor
	engine.executor = NewTradeExecutor(config.Commission, config.SlippagePips, config.SpreadPips)

	return engine, nil
}

// SetStrategy sets a new strategy for the backtest engine
func (e *BacktestEngine) SetStrategy(strategyName string, parameters map[string]interface{}) error {
	strategy, err := strategies.CreateStrategy(strategyName, parameters)
	if err != nil {
		return errors.New("failed to create strategy: " + err.Error())
	}

	// Validate strategy parameters
	if err := strategy.ValidateParameters(); err != nil {
		return errors.New("invalid strategy parameters: " + err.Error())
	}

	e.strategy = strategy
	return nil
}

// Run executes the backtest
func (e *BacktestEngine) Run() (*BacktestResult, error) {
	startTime := time.Now()

	// Get historical data
	bars, err := e.replayer.GetBars()
	if err != nil {
		return nil, errors.New("failed to load historical data: " + err.Error())
	}

	if len(bars) == 0 {
		return nil, errors.New("no historical data available for the specified period")
	}

	// Record initial equity
	e.portfolio.EquityCurve = append(e.portfolio.EquityCurve, EquityPoint{
		Timestamp: bars[0].Timestamp,
		Equity:    e.config.InitialCapital,
		Drawdown:  0,
	})

	// Run simulation
	for i, bar := range bars {
		// Convert to BarData for strategy
		barData := strategies.BarData{
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			Volume:    bar.Volume,
			Timestamp: bar.Timestamp,
		}

		// Get historical context (last N bars)
		history := bars[:i]
		if len(history) > 200 {
			history = history[len(history)-200:]
		}

		// Convert history to BarData
		historyData := make([]strategies.BarData, len(history))
		for j, h := range history {
			historyData[j] = strategies.BarData{
				Open:      h.Open,
				High:      h.High,
				Low:       h.Low,
				Close:     h.Close,
				Volume:    h.Volume,
				Timestamp: h.Timestamp,
			}
		}

		// Generate signal
		signal := e.strategy.OnBar(barData, historyData, e.convertPortfolio())

		// Process signal
		if signal.Type != strategies.SignalHold {
			order := e.strategy.OnSignal(signal, e.convertPortfolio())
			if order != nil {
				// Calculate position size
				positionSize := e.calculatePositionSize(order)
				order.Quantity = positionSize

				// Execute trade
				trade := e.executor.Execute(order, &e.portfolio)
				if trade != nil {
					e.portfolio.Trades = append(e.portfolio.Trades, *trade)
				}
			}
		}

		// Update open positions (check stop loss / take profit)
		e.updatePositions(bar.Close, bar.Timestamp)

		// Update equity curve
		currentEquity := e.calculateEquity(bar.Close)
		peakEquity := e.portfolio.EquityCurve[len(e.portfolio.EquityCurve)-1].Equity
		drawdown := 0.0
		if peakEquity > 0 {
			drawdown = (peakEquity - currentEquity) / peakEquity * 100
		}

		e.portfolio.EquityCurve = append(e.portfolio.EquityCurve, EquityPoint{
			Timestamp: bar.Timestamp,
			Equity:    currentEquity,
			Drawdown:  drawdown,
		})
	}

	// Close any remaining positions at final price
	finalBar := bars[len(bars)-1]
	e.closeAllPositions(finalBar.Close, finalBar.Timestamp)

	// Calculate performance metrics
	metrics := e.calculateMetrics()

	endTime := time.Now()

	result := &BacktestResult{
		Config:      e.config,
		Metrics:     metrics,
		Trades:      e.portfolio.Trades,
		EquityCurve: e.portfolio.EquityCurve,
		DailyReturns: e.calculateDailyReturns(),
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      "completed",
	}

	return result, nil
}

// calculatePositionSize calculates position size based on configuration
func (e *BacktestEngine) calculatePositionSize(order *strategies.Order) float64 {
	// Apply stop-loss and take-profit from config if not set in order
	if order.StopLoss == 0 && e.config.StopLossPips > 0 {
		order.StopLoss = e.calculateStopLossPrice(order.EntryPrice, order.Side)
	}
	if order.TakeProfit == 0 && e.config.TakeProfitPips > 0 {
		order.TakeProfit = e.calculateTakeProfitPrice(order.EntryPrice, order.Side)
	}

	switch e.config.PositionSizing.Mode {
	case "FIXED_LOT":
		return e.config.PositionSizing.FixedLot
	case "PERCENT_EQUITY":
		riskAmount := e.portfolio.Equity * e.config.PositionSizing.EquityPercent
		stopLossPips := e.config.PositionSizing.StopLossPips
		if stopLossPips > 0 {
			// Calculate lot size based on risk
			pipValue := 0.0001 * e.config.InitialCapital // Simplified
			return riskAmount / (stopLossPips * pipValue)
		}
		return riskAmount / order.EntryPrice
	default:
		return 1.0 // Default lot size
	}
}

// calculateStopLossPrice calculates stop-loss price based on pips
func (e *BacktestEngine) calculateStopLossPrice(entryPrice float64, side strategies.SignalType) float64 {
	pips := e.config.StopLossPips * 0.0001
	if side == strategies.SignalBuy {
		return entryPrice - pips
	}
	return entryPrice + pips
}

// calculateTakeProfitPrice calculates take-profit price based on pips
func (e *BacktestEngine) calculateTakeProfitPrice(entryPrice float64, side strategies.SignalType) float64 {
	pips := e.config.TakeProfitPips * 0.0001
	if side == strategies.SignalBuy {
		return entryPrice + pips
	}
	return entryPrice - pips
}

// updatePositions checks and updates open positions
func (e *BacktestEngine) updatePositions(currentPrice float64, timestamp time.Time) {
	for i := len(e.portfolio.OpenPositions) - 1; i >= 0; i-- {
		pos := &e.portfolio.OpenPositions[i]

		// Check stop loss / take profit
		var closed bool
		var exitPrice float64

		if pos.Type == strategies.SignalBuy {
			if pos.StopLoss > 0 && currentPrice <= pos.StopLoss {
				closed = true
				exitPrice = pos.StopLoss
			} else if pos.TakeProfit > 0 && currentPrice >= pos.TakeProfit {
				closed = true
				exitPrice = pos.TakeProfit
			}
		} else if pos.Type == strategies.SignalSell {
			if pos.StopLoss > 0 && currentPrice >= pos.StopLoss {
				closed = true
				exitPrice = pos.StopLoss
			} else if pos.TakeProfit > 0 && currentPrice <= pos.TakeProfit {
				closed = true
				exitPrice = pos.TakeProfit
			}
		}

		if closed {
			// Calculate P&L
			var pnl float64
			if pos.Type == strategies.SignalBuy {
				pnl = (exitPrice - pos.EntryPrice) * pos.Quantity
			} else {
				pnl = (pos.EntryPrice - exitPrice) * pos.Quantity
			}

			trade := strategies.BacktestTrade{
				ID:           pos.ID,
				Type:         pos.Type,
				EntryPrice:   pos.EntryPrice,
				ExitPrice:    exitPrice,
				Quantity:     pos.Quantity,
				PnL:          pnl,
				PnLPercent:   (pnl / (pos.EntryPrice * pos.Quantity)) * 100,
				EntryTime:    pos.OpenedAt,
				ExitTime:     timestamp,
				StopLoss:     pos.StopLoss,
				TakeProfit:   pos.TakeProfit,
				Strategy:     e.strategy.GetName(),
				CurrencyPair: pos.CurrencyPair,
			}

			e.portfolio.Trades = append(e.portfolio.Trades, trade)
			e.portfolio.Equity += pnl
			e.portfolio.Cash += pnl

			// Remove position
			e.portfolio.OpenPositions = append(e.portfolio.OpenPositions[:i], e.portfolio.OpenPositions[i+1:]...)
		}
	}
}

// closeAllPositions closes all remaining positions at current price
func (e *BacktestEngine) closeAllPositions(currentPrice float64, timestamp time.Time) {
	for _, pos := range e.portfolio.OpenPositions {
		var pnl float64
		if pos.Type == strategies.SignalBuy {
			pnl = (currentPrice - pos.EntryPrice) * pos.Quantity
		} else {
			pnl = (pos.EntryPrice - currentPrice) * pos.Quantity
		}

		trade := strategies.BacktestTrade{
			ID:           pos.ID,
			Type:         pos.Type,
			EntryPrice:   pos.EntryPrice,
			ExitPrice:    currentPrice,
			Quantity:     pos.Quantity,
			PnL:          pnl,
			PnLPercent:   (pnl / (pos.EntryPrice * pos.Quantity)) * 100,
			EntryTime:    pos.OpenedAt,
			ExitTime:     timestamp,
			Strategy:     e.strategy.GetName(),
			CurrencyPair: pos.CurrencyPair,
		}

		e.portfolio.Trades = append(e.portfolio.Trades, trade)
		e.portfolio.Equity += pnl
		e.portfolio.Cash += pnl
	}

	e.portfolio.OpenPositions = nil
}

// calculateEquity calculates current total equity
func (e *BacktestEngine) calculateEquity(currentPrice float64) float64 {
	equity := e.portfolio.Cash

	for _, pos := range e.portfolio.OpenPositions {
		var unrealizedPnL float64
		if pos.Type == strategies.SignalBuy {
			unrealizedPnL = (currentPrice - pos.EntryPrice) * pos.Quantity
		} else {
			unrealizedPnL = (pos.EntryPrice - currentPrice) * pos.Quantity
		}
		equity += unrealizedPnL
	}

	return equity
}

// calculateMetrics calculates performance metrics
func (e *BacktestEngine) calculateMetrics() PerformanceMetrics {
	metrics := PerformanceMetrics{
		TotalTrades: len(e.portfolio.Trades),
	}

	if metrics.TotalTrades == 0 {
		return metrics
	}

	// Calculate basic metrics
	finalEquity := e.portfolio.EquityCurve[len(e.portfolio.EquityCurve)-1].Equity
	metrics.TotalReturn = ((finalEquity - e.config.InitialCapital) / e.config.InitialCapital) * 100

	// Calculate trade statistics
	var grossProfit, grossLoss float64
	for _, trade := range e.portfolio.Trades {
		if trade.PnL > 0 {
			metrics.WinningTrades++
			grossProfit += trade.PnL
		} else {
			metrics.LosingTrades++
			grossLoss += mathAbs(trade.PnL)
		}
	}

	metrics.GrossProfit = grossProfit
	metrics.GrossLoss = grossLoss

	// Win rate
	if metrics.TotalTrades > 0 {
		metrics.WinRate = (float64(metrics.WinningTrades) / float64(metrics.TotalTrades)) * 100
	}

	// Average win/loss
	if metrics.WinningTrades > 0 {
		metrics.AverageWin = grossProfit / float64(metrics.WinningTrades)
	}
	if metrics.LosingTrades > 0 {
		metrics.AverageLoss = grossLoss / float64(metrics.LosingTrades)
	}

	// Profit factor
	if grossLoss > 0 {
		metrics.ProfitFactor = grossProfit / grossLoss
	}

	// Expectancy
	winRate := metrics.WinRate / 100
	metrics.Expectancy = (winRate * metrics.AverageWin) - ((1 - winRate) * metrics.AverageLoss)

	// Max drawdown
	maxDrawdown := 0.0
	for _, point := range e.portfolio.EquityCurve {
		if point.Drawdown > maxDrawdown {
			maxDrawdown = point.Drawdown
		}
	}
	metrics.MaxDrawdown = maxDrawdown

	// Annualized return
	tradingDays := len(e.portfolio.EquityCurve)
	if tradingDays > 0 {
		days := float64(tradingDays)
		metrics.AnnualizedReturn = (math.Pow(1+metrics.TotalReturn/100, 252/days) - 1) * 100
	}

	// Sharpe Ratio and Sortino Ratio (using equity curve returns)
	if len(e.portfolio.EquityCurve) > 1 {
		returns := make([]float64, len(e.portfolio.EquityCurve)-1)
		for i := 1; i < len(e.portfolio.EquityCurve); i++ {
			returns[i-1] = (e.portfolio.EquityCurve[i].Equity - e.portfolio.EquityCurve[i-1].Equity) / e.portfolio.EquityCurve[i-1].Equity
		}
		avgReturn := sum(returns) / float64(len(returns))
		stdDeviation := stdDev(returns)

		// Sharpe Ratio
		if stdDeviation > 0 {
			metrics.SharpeRatio = (avgReturn - 0.02/252) / stdDeviation * math.Sqrt(252)
		}

		// Sortino Ratio - uses downside deviation only
		downsideReturns := make([]float64, 0)
		for _, r := range returns {
			if r < 0 {
				downsideReturns = append(downsideReturns, r)
			}
		}
		if len(downsideReturns) > 0 {
			downsideDev := stdDev(downsideReturns)
			if downsideDev > 0 {
				metrics.SortinoRatio = (avgReturn - 0.02/252) / downsideDev * math.Sqrt(252)
			}
		}
	}

	// Max Drawdown Duration - tracks consecutive days in drawdown
	maxDrawdownDuration := 0
	currentDrawdownDuration := 0
	for _, point := range e.portfolio.EquityCurve {
		if point.Drawdown > 0 {
			currentDrawdownDuration++
			if currentDrawdownDuration > maxDrawdownDuration {
				maxDrawdownDuration = currentDrawdownDuration
			}
		} else {
			currentDrawdownDuration = 0
		}
	}
	metrics.MaxDrawdownDuration = maxDrawdownDuration

	// Calmar Ratio
	if metrics.MaxDrawdown > 0 {
		metrics.CalmarRatio = metrics.AnnualizedReturn / metrics.MaxDrawdown
	}

	return metrics
}

// calculateDailyReturns calculates daily returns
func (e *BacktestEngine) calculateDailyReturns() []DailyReturn {
	if len(e.portfolio.EquityCurve) < 2 {
		return nil
	}

	dailyMap := make(map[string]float64)

	for i := 1; i < len(e.portfolio.EquityCurve); i++ {
		date := e.portfolio.EquityCurve[i].Timestamp.Format("2006-01-02")
		prevEquity := e.portfolio.EquityCurve[i-1].Equity
		if prevEquity > 0 {
			dailyReturn := (e.portfolio.EquityCurve[i].Equity - prevEquity) / prevEquity * 100
			if existing, ok := dailyMap[date]; ok {
				dailyMap[date] = existing + dailyReturn
			} else {
				dailyMap[date] = dailyReturn
			}
		}
	}

	dailyReturns := make([]DailyReturn, 0, len(dailyMap))
	cumulative := 0.0

	for date, ret := range dailyMap {
		t, _ := time.Parse("2006-01-02", date)
		cumulative += ret
		dailyReturns = append(dailyReturns, DailyReturn{
			Date:       t,
			Return:     ret,
			Cumulative: cumulative,
		})
	}

	return dailyReturns
}

// convertPortfolio converts internal portfolio to strategy portfolio
func (e *BacktestEngine) convertPortfolio() strategies.Portfolio {
	positions := make([]strategies.Position, len(e.portfolio.OpenPositions))
	for i, pos := range e.portfolio.OpenPositions {
		positions[i] = strategies.Position{
			ID:           pos.ID,
			Type:         pos.Type,
			EntryPrice:   pos.EntryPrice,
			Quantity:     pos.Quantity,
			StopLoss:     pos.StopLoss,
			TakeProfit:   pos.TakeProfit,
			OpenedAt:     pos.OpenedAt,
			CurrencyPair: pos.CurrencyPair,
		}
	}

	return strategies.Portfolio{
		Equity:        e.portfolio.Equity,
		Cash:          e.portfolio.Cash,
		OpenPositions: positions,
		ClosedTrades:  e.portfolio.Trades,
	}
}

// Helper functions
func mathAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func sum(arr []float64) float64 {
	total := 0.0
	for _, v := range arr {
		total += v
	}
	return total
}

func stdDev(arr []float64) float64 {
	if len(arr) == 0 {
		return 0
	}
	avg := sum(arr) / float64(len(arr))
	var sumSqDiff float64
	for _, v := range arr {
		diff := v - avg
		sumSqDiff += diff * diff
	}
	return math.Sqrt(sumSqDiff / float64(len(arr)))
}
