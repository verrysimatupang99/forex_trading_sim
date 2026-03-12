package strategies

import (
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

// MACrossoverParams defines parameters for Moving Average Crossover strategy
type MACrossoverParams struct {
	FastMA    int     `json:"fast_ma"`    // Fast MA period (default: 10)
	SlowMA    int     `json:"slow_ma"`    // Slow MA period (default: 50)
	StopLoss  float64 `json:"stop_loss"`  // Stop loss in pips (default: 20)
	TakeProfit float64 `json:"take_profit"` // Take profit in pips (default: 40)
}

// MACrossoverStrategy implements the Moving Average Crossover strategy
type MACrossoverStrategy struct {
	BaseStrategy
	FastMA    int
	SlowMA    int
	StopLoss  float64
	TakeProfit float64
}

// NewMACrossoverStrategy creates a new Moving Average Crossover strategy
func NewMACrossoverStrategy(params map[string]interface{}) (Strategy, error) {
	p := MACrossoverParams{
		FastMA:     10,
		SlowMA:     50,
		StopLoss:   20,
		TakeProfit: 40,
	}

	// Override defaults with provided params
	if v, ok := params["fast_ma"].(float64); ok {
		p.FastMA = int(v)
	}
	if v, ok := params["slow_ma"].(float64); ok {
		p.SlowMA = int(v)
	}
	if v, ok := params["slow_ma"].(int); ok {
		p.SlowMA = v
	}
	if v, ok := params["stop_loss"].(float64); ok {
		p.StopLoss = v
	}
	if v, ok := params["take_profit"].(float64); ok {
		p.TakeProfit = v
	}

	strategy := &MACrossoverStrategy{
		FastMA:      p.FastMA,
		SlowMA:      p.SlowMA,
		StopLoss:    p.StopLoss,
		TakeProfit:  p.TakeProfit,
	}
	strategy.Name = "MA Crossover"
	strategy.Description = "Buy when fast MA crosses above slow MA, Sell when fast MA crosses below slow MA"
	strategy.Parameters = map[string]interface{}{
		"fast_ma":     p.FastMA,
		"slow_ma":     p.SlowMA,
		"stop_loss":   p.StopLoss,
		"take_profit": p.TakeProfit,
	}

	return strategy, nil
}

// ValidateParameters validates the strategy parameters
func (s *MACrossoverStrategy) ValidateParameters() error {
	if s.FastMA <= 0 {
		return errors.New("fast_ma must be greater than 0")
	}
	if s.SlowMA <= 0 {
		return errors.New("slow_ma must be greater than 0")
	}
	if s.FastMA >= s.SlowMA {
		return errors.New("fast_ma must be less than slow_ma")
	}
	if s.StopLoss <= 0 {
		return errors.New("stop_loss must be greater than 0")
	}
	if s.TakeProfit <= 0 {
		return errors.New("take_profit must be greater than 0")
	}
	return nil
}

// OnBar generates a trading signal based on MA crossover
func (s *MACrossoverStrategy) OnBar(bar BarData, history []BarData, portfolio Portfolio) Signal {
	// Need enough history for both MAs
	requiredBars := s.SlowMA + 1
	if len(history) < requiredBars {
		return Signal{
			Type:      SignalHold,
			Strength:  0,
			Timestamp: bar.Timestamp,
			Reason:    "Insufficient historical data",
		}
	}

	// Calculate moving averages
	fastMA := calculateSMA(history, s.FastMA)
	slowMA := calculateSMA(history, s.SlowMA)

	// Get previous MA values
	prevFastMA := calculateSMA(history[:len(history)-1], s.FastMA)
	prevSlowMA := calculateSMA(history[:len(history)-1], s.SlowMA)

	// Detect crossover
	currentCross := fastMA - slowMA
	prevCross := prevFastMA - prevSlowMA

	// Buy signal: fast MA crosses above slow MA
	if prevCross <= 0 && currentCross > 0 {
		stopLoss := bar.Close - (s.StopLoss * 0.0001) // Convert pips to price
		takeProfit := bar.Close + (s.TakeProfit * 0.0001)

		return Signal{
			Type:       SignalBuy,
			Strength:   80,
			Price:      bar.Close,
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Reason:     "Fast MA crossed above Slow MA",
			Timestamp:  bar.Timestamp,
		}
	}

	// Sell signal: fast MA crosses below slow MA
	if prevCross >= 0 && currentCross < 0 {
		stopLoss := bar.Close + (s.StopLoss * 0.0001)
		takeProfit := bar.Close - (s.TakeProfit * 0.0001)

		return Signal{
			Type:       SignalSell,
			Strength:   80,
			Price:      bar.Close,
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Reason:     "Fast MA crossed below Slow MA",
			Timestamp:  bar.Timestamp,
		}
	}

	// Hold if no crossover
	return Signal{
		Type:      SignalHold,
		Strength:  50,
		Price:     bar.Close,
		Timestamp: bar.Timestamp,
		Reason:    "No crossover detected",
	}
}

// OnSignal converts a signal to an order
func (s *MACrossoverStrategy) OnSignal(signal Signal, portfolio Portfolio) *Order {
	if signal.Type == SignalHold {
		return nil
	}

	order := &Order{
		Type:          OrderMarket,
		Side:          signal.Type,
		Quantity:      1.0, // Default lot size - should be calculated based on risk
		EntryPrice:    signal.Price,
		StopLoss:      signal.StopLoss,
		TakeProfit:    signal.TakeProfit,
		Timestamp:     signal.Timestamp,
		CurrencyPair:  "EUR/USD", // Default - should be passed in params
	}

	return order
}

// OnTrade is called when a trade is executed (can be used for logging, etc.)
func (s *MACrossoverStrategy) OnTrade(trade BacktestTrade, portfolio *Portfolio) {
	// Update portfolio equity
	if trade.PnL != 0 {
		portfolio.Equity += trade.PnL
		portfolio.Cash += trade.PnL
	}
}

// calculateSMA calculates Simple Moving Average
func calculateSMA(bars []BarData, period int) float64 {
	if len(bars) < period {
		return 0
	}

	sum := 0.0
	for i := len(bars) - period; i < len(bars); i++ {
		sum += bars[i].Close
	}
	return sum / float64(period)
}

// calculateEMA calculates Exponential Moving Average
func calculateEMA(bars []BarData, period int) float64 {
	if len(bars) < period {
		return 0
	}

	multiplier := 2.0 / float64(period+1)
	ema := bars[0].Close

	for i := 1; i < len(bars); i++ {
		ema = (bars[i].Close - ema) * multiplier + ema
	}

	return ema
}

// CrossEvent represents a MA crossover event
type CrossEvent struct {
	Type      string    // "BULLISH" or "BEARISH"
	FastMA    float64   // Current fast MA value
	SlowMA    float64   // Current slow MA value
	Timestamp time.Time
}

// DetectCross detects if there's a crossover event
func DetectCross(bars []BarData, fastPeriod, slowPeriod int) *CrossEvent {
	if len(bars) < slowPeriod+1 {
		return nil
	}

	fastMA := calculateEMA(bars, fastPeriod)
	slowMA := calculateEMA(bars, slowPeriod)

	prevFastMA := calculateEMA(bars[:len(bars)-1], fastPeriod)
	prevSlowMA := calculateEMA(bars[:len(bars)-1], slowPeriod)

	currentCross := fastMA - slowMA
	prevCross := prevFastMA - prevSlowMA

	if prevCross <= 0 && currentCross > 0 {
		return &CrossEvent{
			Type:      "BULLISH",
			FastMA:    fastMA,
			SlowMA:    slowMA,
			Timestamp: bars[len(bars)-1].Timestamp,
		}
	}

	if prevCross >= 0 && currentCross < 0 {
		return &CrossEvent{
			Type:      "BEARISH",
			FastMA:    fastMA,
			SlowMA:    slowMA,
			Timestamp: bars[len(bars)-1].Timestamp,
		}
	}

	return nil
}

// init registers the strategy
func init() {
	RegisterStrategy("ma_crossover", NewMACrossoverStrategy)
	RegisterStrategy("MA Crossover", NewMACrossoverStrategy)
}

// GenerateUniqueID generates a unique ID for trades/positions
func GenerateUniqueID() string {
	return uuid.New().String()
}

// Math helpers

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func round(x float64) float64 {
	return math.Round(x*10000) / 10000
}
