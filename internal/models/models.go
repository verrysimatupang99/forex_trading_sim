package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
	FirstName    string         `json:"first_name"`
	LastName     string         `json:"last_name"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	Role         string         `gorm:"default:'user'" json:"role"` // user, admin
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Accounts []Account `gorm:"foreignKey:UserID" json:"accounts,omitempty"`
}

// Account represents a trading account
type Account struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	UserID        uint           `gorm:"not null;index" json:"user_id"`
	AccountNumber string         `gorm:"uniqueIndex;not null" json:"account_number"`
	Balance       float64        `gorm:"default:10000" json:"balance"` // Starting balance for simulator
	Equity        float64        `gorm:"default:10000" json:"equity"`
	Margin        float64        `gorm:"default:0" json:"margin"`
	MarginUsed    float64        `gorm:"default:0" json:"margin_used"`
	MarginFree    float64        `gorm:"default:10000" json:"margin_free"`
	Leverage      float64        `gorm:"default:1" json:"leverage"` // 1:1 to 100:1
	Currency      string         `gorm:"default:'USD'" json:"currency"`
	IsDemo        bool           `gorm:"default:true" json:"is_demo"`
	Status        string         `gorm:"default:'active'" json:"status"` // active, closed, suspended
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	User      User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Trades    []Trade    `gorm:"foreignKey:AccountID" json:"trades,omitempty"`
	Positions []Position `gorm:"foreignKey:AccountID" json:"positions,omitempty"`
}

// CurrencyPair represents a forex currency pair
type CurrencyPair struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Symbol        string    `gorm:"uniqueIndex;not null" json:"symbol"` // e.g., EUR/USD
	BaseCurrency  string    `gorm:"not null" json:"base_currency"`        // EUR
	QuoteCurrency string    `gorm:"not null" json:"quote_currency"`       // USD
	PipValue      float64   `gorm:"default:0.0001" json:"pip_value"`      // Standard pip = 0.0001
	Digits        int       `gorm:"default:4" json:"digits"`              // Price digits
	Description   string    `json:"description"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`

	// Spread configuration (in pips)
	MinSpread     float64   `gorm:"default:1.0" json:"min_spread"`       // Minimum spread in pips
	MaxSpread     float64   `gorm:"default:5.0" json:"max_spread"`       // Maximum spread in pips
	TypicalSpread float64   `gorm:"default:2.0" json:"typical_spread"`    // Typical spread in pips
	SpreadUnit    string    `gorm:"default:'pip'" json:"spread_unit"`      // pip or point

	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Relations
	HistoricalPrices []HistoricalPrice `gorm:"foreignKey:CurrencyPairID" json:"historical_prices,omitempty"`
	Trades           []Trade           `gorm:"foreignKey:CurrencyPairID" json:"trades,omitempty"`
	Positions        []Position        `gorm:"foreignKey:CurrencyPairID" json:"positions,omitempty"`
}

// HistoricalPrice stores OHLCV data for currency pairs
type HistoricalPrice struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CurrencyPairID uint      `gorm:"not null;index" json:"currency_pair_id"`
	Timestamp      time.Time `gorm:"not null;index" json:"timestamp"`
	Open           float64   `gorm:"not null" json:"open"`
	High           float64   `gorm:"not null" json:"high"`
	Low            float64   `gorm:"not null" json:"low"`
	Close          float64   `gorm:"not null" json:"close"`
	Volume         float64   `gorm:"default:0" json:"volume"`
	Timeframe      string    `gorm:"not null" json:"timeframe"` // 1m, 5m, 15m, 1h, 4h, 1d
	CreatedAt      time.Time `json:"created_at"`

	// Relations
	CurrencyPair CurrencyPair `gorm:"foreignKey:CurrencyPairID" json:"currency_pair,omitempty"`
}

// Trade represents an executed trade
type Trade struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	AccountID      uint           `gorm:"not null;index" json:"account_id"`
	CurrencyPairID uint           `gorm:"not null;index" json:"currency_pair_id"`
	Type           string         `gorm:"not null" json:"type"` // BUY, SELL
	EntryPrice     float64        `gorm:"not null" json:"entry_price"`
	ExitPrice      float64        `json:"exit_price"`
	Quantity       float64        `gorm:"not null" json:"quantity"` // Lot size
	PnL            float64        `gorm:"default:0" json:"pnl"`     // Profit/Loss
	PnLPercent     float64        `gorm:"default:0" json:"pnl_percent"`
	Commission     float64        `gorm:"default:0" json:"commission"`
	Swap           float64        `gorm:"default:0" json:"swap"` // Overnight swap
	Status         string         `gorm:"default:'open'" json:"status"` // open, closed
	EntryTime      time.Time      `gorm:"not null" json:"entry_time"`
	ExitTime       *time.Time     `json:"exit_time"`
	StopLoss       float64        `json:"stop_loss"`
	TakeProfit     float64        `json:"take_profit"`
	Strategy       string         `json:"strategy"` // Used strategy
	Notes          string         `json:"notes"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Account      Account      `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	CurrencyPair CurrencyPair `gorm:"foreignKey:CurrencyPairID" json:"currency_pair,omitempty"`
}

// Position represents an open position
type Position struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	AccountID      uint           `gorm:"not null;index" json:"account_id"`
	CurrencyPairID uint           `gorm:"not null;index" json:"currency_pair_id"`
	Type           string         `gorm:"not null" json:"type"` // BUY, SELL
	EntryPrice     float64        `gorm:"not null" json:"entry_price"`
	CurrentPrice   float64        `gorm:"not null" json:"current_price"`
	Quantity       float64        `gorm:"not null" json:"quantity"`
	UnrealizedPnL  float64        `gorm:"default:0" json:"unrealized_pnl"`
	StopLoss       float64        `json:"stop_loss"`
	TakeProfit     float64        `json:"take_profit"`
	IsHedged       bool           `gorm:"default:false" json:"is_hedged"`
	OpenedAt       time.Time      `gorm:"not null" json:"opened_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Account      Account      `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	CurrencyPair CurrencyPair `gorm:"foreignKey:CurrencyPairID" json:"currency_pair,omitempty"`
}

// OrderType represents the type of order
type OrderType string

const (
	// Market order - executes immediately at current price
	OrderTypeMarket OrderType = "MARKET"
	// Limit order - executes when price reaches specified level
	OrderTypeLimit OrderType = "LIMIT"
	// Stop order - triggers when price reaches specified level
	OrderTypeStop OrderType = "STOP"
)

// OrderSide represents buy or sell side
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// PendingOrder represents a pending (limit/stop) order
type PendingOrder struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	AccountID      uint           `gorm:"not null;index" json:"account_id"`
	CurrencyPairID uint           `gorm:"not null;index" json:"currency_pair_id"`
	OrderType      string         `gorm:"not null" json:"order_type"` // LIMIT, STOP
	Side           string         `gorm:"not null" json:"side"`       // BUY, SELL
	Quantity       float64        `gorm:"not null" json:"quantity"`
	Price          float64        `gorm:"not null" json:"price"`     // Limit price or stop price
	StopLoss       float64        `json:"stop_loss"`
	TakeProfit     float64        `json:"take_profit"`
	Status         string         `gorm:"default:'pending'" json:"status"` // pending, triggered, cancelled, executed
	OCOGroupID     *uint          `gorm:"index" json:"oco_group_id"` // Reference to OCO group
	OTOGroupID     *uint          `gorm:"index" json:"oto_group_id"` // Reference to OTO group
	IsPrimary      bool           `gorm:"default:false" json:"is_primary"` // For OTO: true if primary order
	ExpiresAt      *time.Time     `json:"expires_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Account      Account      `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	CurrencyPair CurrencyPair `gorm:"foreignKey:CurrencyPairID" json:"currency_pair,omitempty"`
}

// OCOOrder represents One Cancels Other order group
type OCOOrder struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	AccountID      uint           `gorm:"not null;index" json:"account_id"`
	CurrencyPairID uint           `gorm:"not null;index" json:"currency_pair_id"`
	Name           string         `json:"name"` // Optional name for the OCO group
	Status         string         `gorm:"default:'active'" json:"status"` // active, cancelled, executed
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Account       Account        `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	CurrencyPair  CurrencyPair   `gorm:"foreignKey:CurrencyPairID" json:"currency_pair,omitempty"`
}

// OCOOrderWithDetails holds OCO order with its two pending orders for creation
type OCOOrderWithDetails struct {
	OCOGroupID   uint
	BuyOrder    PendingOrder
	SellOrder   PendingOrder
}

// OTOOrder represents One Triggers Other order group
type OTOOrder struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	AccountID      uint           `gorm:"not null;index" json:"account_id"`
	CurrencyPairID uint           `gorm:"not null;index" json:"currency_pair_id"`
	Name           string         `json:"name"` // Optional name for the OTO group
	Status         string         `gorm:"default:'active'" json:"status"` // active, cancelled, executed
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Account          Account        `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	CurrencyPair     CurrencyPair   `gorm:"foreignKey:CurrencyPairID" json:"currency_pair,omitempty"`
}

// MLModel stores information about trained ML models
type MLModel struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null" json:"name"`
	Version     string         `gorm:"not null" json:"version"`
	Type        string         `gorm:"not null" json:"type"` // LSTM, XGBoost, etc.
	FilePath    string         `gorm:"not null" json:"file_path"`
	Metrics     string         `gorm:"type:jsonb" json:"metrics"` // JSON metrics (accuracy, loss, etc.)
	Hyperparams string         `gorm:"type:jsonb" json:"hyperparams"`
	TrainingData string        `json:"training_data"` // Date range used for training
	Accuracy    float64        `json:"accuracy"`
	Loss        float64        `json:"loss"`
	IsActive    bool           `gorm:"default:false" json:"is_active"`
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	TrainedAt   time.Time      `json:"trained_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Predictions []Prediction `gorm:"foreignKey:ModelID" json:"predictions,omitempty"`
}

// Prediction stores ML model predictions
type Prediction struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	ModelID        uint           `gorm:"not null;index" json:"model_id"`
	CurrencyPairID uint           `gorm:"not null;index" json:"currency_pair_id"`
	Signal         string         `gorm:"not null" json:"signal"`     // BUY, SELL, HOLD
	Confidence     float64        `gorm:"not null" json:"confidence"` // 0-100
	EntryPrice     float64        `json:"entry_price"`
	TargetPrice    float64        `json:"target_price"`
	StopLoss       float64        `json:"stop_loss"`
	TakeProfit     float64        `json:"take_profit"`
	Timeframe      string         `gorm:"not null" json:"timeframe"`
	PredictionTime time.Time      `gorm:"not null" json:"prediction_time"`
	ExpiryTime     *time.Time     `json:"expiry_time"`
	IsActualized   bool           `gorm:"default:false" json:"is_actualized"`
	ActualResult   string         `json:"actual_result"` // WIN, LOSS
	ActualPnL      float64        `json:"actual_pnl"`
	CreatedAt      time.Time      `json:"created_at"`

	// Relations
	Model        MLModel        `gorm:"foreignKey:ModelID" json:"model,omitempty"`
	CurrencyPair CurrencyPair   `gorm:"foreignKey:CurrencyPairID" json:"currency_pair,omitempty"`
}

// TechnicalIndicator stores computed technical indicators
type TechnicalIndicator struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CurrencyPairID uint      `gorm:"not null;index" json:"currency_pair_id"`
	Timestamp      time.Time `gorm:"not null;index" json:"timestamp"`
	Timeframe      string    `gorm:"not null" json:"timeframe"`
	IndicatorName  string    `gorm:"not null" json:"indicator_name"`
	Value          float64   `gorm:"not null" json:"value"`
	CreatedAt      time.Time `json:"created_at"`
}

// CurrencyRate stores exchange rates between currencies
type CurrencyRate struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	BaseCurrency    string    `gorm:"not null;index" json:"base_currency"`     // e.g., USD
	QuoteCurrency   string    `gorm:"not null;index" json:"quote_currency"`    // e.g., EUR
	Rate            float64   `gorm:"not null" json:"rate"`                    // Exchange rate
	Bid             float64   `gorm:"not null" json:"bid"`                     // Bid price
	Ask             float64   `gorm:"not null" json:"ask"`                     // Ask price
	Spread          float64   `gorm:"not null" json:"spread"`                  // Spread in pips
	Timestamp       time.Time `gorm:"not null;index" json:"timestamp"`         // Rate timestamp
	Source          string    `gorm:"default:'manual'" json:"source"`          // Rate source (OANDA, AlphaVantage, manual)
	IsRealTime      bool      `gorm:"default:false" json:"is_real_time"`       // Real-time or delayed
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Unique constraint for currency pair
	UniquePair string `gorm:"uniqueIndex:idx_currency_pair_unique;not null"` // base_quote
}

// MultiCurrencyAccount supports multiple base currencies
type MultiCurrencyAccount struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"not null;index" json:"user_id"`
	BaseCurrency    string    `gorm:"not null" json:"base_currency"`           // Account base currency
	TotalBalanceUSD float64   `gorm:"not null" json:"total_balance_usd"`       // Total balance in USD
	TotalEquityUSD  float64   `gorm:"not null" json:"total_equity_usd"`        // Total equity in USD
	MarginUsedUSD   float64   `gorm:"not null" json:"margin_used_usd"`         // Margin used in USD
	FreeMarginUSD   float64   `gorm:"not null" json:"free_margin_usd"`         // Free margin in USD
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relations
	User            User                    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CurrencyBalances []CurrencyBalance      `gorm:"foreignKey:AccountID" json:"currency_balances,omitempty"`
}

// CurrencyBalance stores balance per currency in multi-currency account
type CurrencyBalance struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	AccountID       uint      `gorm:"not null;index" json:"account_id"`
	Currency        string    `gorm:"not null;index" json:"currency"`          // Currency code (USD, EUR, etc.)
	Balance         float64   `gorm:"not null" json:"balance"`                 // Available balance
	Reserved        float64   `gorm:"default:0" json:"reserved"`               // Reserved for pending orders
	Equity          float64   `gorm:"not null" json:"equity"`                  // Balance + unrealized P&L
	RateToUSD       float64   `gorm:"not null" json:"rate_to_usd"`             // Exchange rate to USD
	BalanceUSD      float64   `gorm:"not null" json:"balance_usd"`             // Balance in USD
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedAt       time.Time `json:"created_at"`

	// Relations
	Account         MultiCurrencyAccount    `gorm:"foreignKey:AccountID" json:"account,omitempty"`
}

// Backtest stores backtest results
type Backtest struct {
	ID                       uint           `gorm:"primaryKey" json:"id"`
	Name                     string         `gorm:"not null" json:"name"`
	StrategyName             string         `gorm:"not null" json:"strategy_name"`
	Parameters               string         `gorm:"type:jsonb" json:"parameters"` // JSON strategy params
	CurrencyPairID           uint           `gorm:"not null" json:"currency_pair_id"`
	Timeframe                string         `gorm:"not null" json:"timeframe"` // 1h, 4h, 1d
	StartDate                time.Time      `gorm:"not null" json:"start_date"`
	EndDate                  time.Time      `gorm:"not null" json:"end_date"`
	InitialCapital           float64        `gorm:"not null" json:"initial_capital"`

	// Results
	TotalReturn              float64        `json:"total_return"`
	AnnualizedReturn         float64        `json:"annualized_return"`
	SharpeRatio             float64        `json:"sharpe_ratio"`
	SortinoRatio            float64        `json:"sortino_ratio"`
	MaxDrawdown             float64        `json:"max_drawdown"`
	MaxDrawdownDuration     int            `json:"max_drawdown_duration"` // in bars
	WinRate                 float64        `json:"win_rate"`
	ProfitFactor            float64        `json:"profit_factor"`
	NumTrades               int            `json:"num_trades"`
	AverageWin              float64        `json:"average_win"`
	AverageLoss             float64        `json:"average_loss"`
	Expectancy              float64        `json:"expectancy"`
	CalmarRatio             float64        `json:"calmar_ratio"`

	// Detailed results stored as JSON
	Trades                   string         `gorm:"type:jsonb" json:"trades"`
	EquityCurve              string         `gorm:"type:jsonb" json:"equity_curve"`
	DailyReturns             string         `gorm:"type:jsonb" json:"daily_returns"`

	Status                   string         `gorm:"default:'pending'" json:"status"` // pending, running, completed, failed
	ErrorMessage             string         `json:"error_message"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	DeletedAt                gorm.DeletedAt `gorm:"index" json:"-"`
}

// WalkForwardAnalysis stores walk-forward analysis results
type WalkForwardAnalysis struct {
	ID                     uint           `gorm:"primaryKey" json:"id"`
	StrategyName           string         `gorm:"not null" json:"strategy_name"`
	CurrencyPairID         uint           `gorm:"not null" json:"currency_pair_id"`
	Timeframe              string         `gorm:"not null" json:"timeframe"`
	StartDate              time.Time      `gorm:"not null" json:"start_date"`
	EndDate                time.Time      `gorm:"not null" json:"end_date"`
	TrainingPeriodDays     int            `gorm:"not null" json:"training_period_days"`
	TestingPeriodDays      int            `gorm:"not null" json:"testing_period_days"`
	StepForwardDays        int            `gorm:"not null" json:"step_forward_days"`
	NumFolds               int            `gorm:"not null" json:"num_folds"`
	
	// Folds stored as JSON
	Folds                  string         `gorm:"type:jsonb" json:"folds"`
	
	// Aggregate metrics
	AvgOutSampleReturn     float64        `json:"avg_out_sample_return"`
	AvgOutSampleSharpe     float64        `json:"avg_out_sample_sharpe"`
	AvgOutSampleDD         float64        `json:"avg_out_sample_max_drawdown"`
	CoefficientOfVariation float64        `json:"coefficient_of_variation"`
	
	StartTime              time.Time      `json:"start_time"`
	EndTime                time.Time      `json:"end_time"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
}
