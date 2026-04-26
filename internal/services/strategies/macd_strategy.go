package strategies

import (
	"errors"
	"math"
	"time"
)

// MACDParams defines parameters for MACD strategy
type MACDParams struct {
	FastPeriod   int     `json:"fast_period"`   // Fast EMA period (default: 12)
	SlowPeriod   int     `json:"slow_period"`   // Slow EMA period (default: 26)
	SignalPeriod int     `json:"signal_period"` // Signal line period (default: 9)
	StopLoss    float64 `json:"stop_loss"`     // Stop loss in pips (default: 20)
	TakeProfit  float64 `json:"take_profit"`   // Take profit in pips (default: 40)
}

// MACDValues holds MACD indicator values
type MACDValues struct {
	MACDLine    float64 // MACD line (fast EMA - slow EMA)
	SignalLine  float64 // Signal line (EMA of MACD)
	Histogram   float64 // MACD histogram (MACD line - signal line)
}

// MACDStrategy implements the MACD (Moving Average Convergence Divergence) strategy
type MACDStrategy struct {
	BaseStrategy
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
	StopLoss     float64
	TakeProfit   float64
}

// NewMACDStrategy creates a new MACD strategy
func NewMACDStrategy(params map[string]interface{}) (Strategy, error) {
	p := MACDParams{
		FastPeriod:   12,
		SlowPeriod:   26,
		SignalPeriod: 9,
		StopLoss:     20,
		TakeProfit:   40,
	}

	// Override defaults with provided params
	if v, ok := params["fast_period"].(float64); ok {
		p.FastPeriod = int(v)
	}
	if v, ok := params["fast_period"].(int); ok {
		p.FastPeriod = v
	}
	if v, ok := params["slow_period"].(float64); ok {
		p.SlowPeriod = int(v)
	}
	if v, ok := params["slow_period"].(int); ok {
		p.SlowPeriod = v
	}
	if v, ok := params["signal_period"].(float64); ok {
		p.SignalPeriod = int(v)
	}
	if v, ok := params["signal_period"].(int); ok {
		p.SignalPeriod = v
	}
	if v, ok := params["stop_loss"].(float64); ok {
		p.StopLoss = v
	}
	if v, ok := params["take_profit"].(float64); ok {
		p.TakeProfit = v
	}

	strategy := &MACDStrategy{
		FastPeriod:   p.FastPeriod,
		SlowPeriod:   p.SlowPeriod,
		SignalPeriod: p.SignalPeriod,
		StopLoss:     p.StopLoss,
		TakeProfit:   p.TakeProfit,
	}
	strategy.Name = "MACD"
	strategy.Description = "Buy when MACD line crosses above signal line, Sell when MACD line crosses below signal line"
	strategy.Parameters = map[string]interface{}{
		"fast_period":   p.FastPeriod,
		"slow_period":   p.SlowPeriod,
		"signal_period": p.SignalPeriod,
		"stop_loss":     p.StopLoss,
		"take_profit":   p.TakeProfit,
	}

	return strategy, nil
}

// ValidateParameters validates the strategy parameters
func (s *MACDStrategy) ValidateParameters() error {
	if s.FastPeriod <= 0 {
		return errors.New("fast_period must be greater than 0")
	}
	if s.SlowPeriod <= 0 {
		return errors.New("slow_period must be greater than 0")
	}
	if s.SignalPeriod <= 0 {
		return errors.New("signal_period must be greater than 0")
	}
	if s.FastPeriod >= s.SlowPeriod {
		return errors.New("fast_period must be less than slow_period")
	}
	if s.StopLoss <= 0 {
		return errors.New("stop_loss must be greater than 0")
	}
	if s.TakeProfit <= 0 {
		return errors.New("take_profit must be greater than 0")
	}
	return nil
}

// OnBar generates a trading signal based on MACD crossover
func (s *MACDStrategy) OnBar(bar BarData, history []BarData, portfolio Portfolio) Signal {
	// Need enough history for MACD calculation
	requiredBars := s.SlowPeriod + s.SignalPeriod + 1
	if len(history) < requiredBars {
		return Signal{
			Type:      SignalHold,
			Strength:  0,
			Timestamp: bar.Timestamp,
			Reason:    "Insufficient historical data for MACD",
		}
	}

	// Calculate current MACD values
	currentMACD := calculateMACD(history, s.FastPeriod, s.SlowPeriod, s.SignalPeriod)

	// Calculate previous MACD values
	prevMACD := calculateMACD(history[:len(history)-1], s.FastPeriod, s.SlowPeriod, s.SignalPeriod)

	// Detect crossover
	currentCross := currentMACD.MACDLine - currentMACD.SignalLine
	prevCross := prevMACD.MACDLine - prevMACD.SignalLine

	// Buy signal: MACD line crosses above signal line
	if prevCross <= 0 && currentCross > 0 {
		stopLoss := bar.Close - (s.StopLoss * 0.0001)
		takeProfit := bar.Close + (s.TakeProfit * 0.0001)

		// Calculate strength based on histogram
		strength := calculateMACDStrength(currentMACD.Histogram)

		return Signal{
			Type:       SignalBuy,
			Strength:   strength,
			Price:      bar.Close,
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Reason:     "MACD crossed above signal line",
			Timestamp:  bar.Timestamp,
		}
	}

	// Sell signal: MACD line crosses below signal line
	if prevCross >= 0 && currentCross < 0 {
		stopLoss := bar.Close + (s.StopLoss * 0.0001)
		takeProfit := bar.Close - (s.TakeProfit * 0.0001)

		// Calculate strength based on histogram
		strength := calculateMACDStrength(-currentMACD.Histogram)

		return Signal{
			Type:       SignalSell,
			Strength:   strength,
			Price:      bar.Close,
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Reason:     "MACD crossed below signal line",
			Timestamp:  bar.Timestamp,
		}
	}

	// Zero line crossover (stronger signal)
	prevZeroCross := prevMACD.MACDLine
	currentZeroCross := currentMACD.MACDLine

	// Buy: MACD crosses above zero from negative
	if prevZeroCross <= 0 && currentZeroCross > 0 {
		stopLoss := bar.Close - (s.StopLoss * 0.0001)
		takeProfit := bar.Close + (s.TakeProfit * 0.0001)

		return Signal{
			Type:       SignalBuy,
			Strength:   90,
			Price:      bar.Close,
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Reason:     "MACD crossed above zero line",
			Timestamp:  bar.Timestamp,
		}
	}

	// Sell: MACD crosses below zero from positive
	if prevZeroCross >= 0 && currentZeroCross < 0 {
		stopLoss := bar.Close + (s.StopLoss * 0.0001)
		takeProfit := bar.Close - (s.TakeProfit * 0.0001)

		return Signal{
			Type:       SignalSell,
			Strength:   90,
			Price:      bar.Close,
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Reason:     "MACD crossed below zero line",
			Timestamp:  bar.Timestamp,
		}
	}

	// Hold: No clear signal
	return Signal{
		Type:      SignalHold,
		Strength:  50,
		Price:     bar.Close,
		Timestamp: bar.Timestamp,
		Reason:    "No MACD crossover detected",
	}
}

// OnSignal converts a signal to an order
func (s *MACDStrategy) OnSignal(signal Signal, portfolio Portfolio) *Order {
	if signal.Type == SignalHold {
		return nil
	}

	order := &Order{
		Type:          OrderMarket,
		Side:          signal.Type,
		Quantity:      1.0,
		EntryPrice:    signal.Price,
		StopLoss:      signal.StopLoss,
		TakeProfit:    signal.TakeProfit,
		Timestamp:     signal.Timestamp,
		CurrencyPair:  "EUR/USD",
	}

	return order
}

// OnTrade is called when a trade is executed
func (s *MACDStrategy) OnTrade(trade BacktestTrade, portfolio *Portfolio) {
	if trade.PnL != 0 {
		portfolio.Equity += trade.PnL
		portfolio.Cash += trade.PnL
	}
}

// calculateMACD calculates MACD indicator values
func calculateMACD(bars []BarData, fastPeriod, slowPeriod, signalPeriod int) MACDValues {
	if len(bars) < slowPeriod+signalPeriod {
		return MACDValues{
			MACDLine:   0,
			SignalLine: 0,
			Histogram:  0,
		}
	}

	// Calculate EMAs
	fastEMA := calculateEMA(bars, fastPeriod)
	slowEMA := calculateEMA(bars, slowPeriod)

	// MACD line = Fast EMA - Slow EMA
	macdLine := fastEMA - slowEMA

	// For signal line, we need to calculate EMA of MACD values
	// This is a simplified version - in production, you'd track MACD over time
	signalLine := macdLine * 0.9 // Simplified signal approximation

	// For more accurate signal, calculate historical MACD
	signalLine = calculateMACDSignal(bars, fastPeriod, slowPeriod, signalPeriod)

	histogram := macdLine - signalLine

	return MACDValues{
		MACDLine:   macdLine,
		SignalLine: signalLine,
		Histogram:  histogram,
	}
}

// calculateMACDSignal calculates the signal line (EMA of MACD)
func calculateMACDSignal(bars []BarData, fastPeriod, slowPeriod, signalPeriod int) float64 {
	if len(bars) < slowPeriod+signalPeriod {
		return 0
	}

	// Calculate MACD values for each bar in the signal period
	macdValues := make([]float64, 0, signalPeriod)

	for i := len(bars) - signalPeriod; i < len(bars); i++ {
		fastEMA := calculateEMA(bars[:i+1], fastPeriod)
		slowEMA := calculateEMA(bars[:i+1], slowPeriod)
		macdValues = append(macdValues, fastEMA-slowEMA)
	}

	if len(macdValues) == 0 {
		return 0
	}

	// Calculate EMA of MACD values
	multiplier := 2.0 / float64(signalPeriod+1)
	signal := macdValues[0]

	for i := 1; i < len(macdValues); i++ {
		signal = (macdValues[i] - signal) * multiplier + signal
	}

	return signal
}

// calculateMACDStrength calculates signal strength based on histogram
func calculateMACDStrength(histogram float64) float64 {
	// Normalize histogram to 0-100 strength
	// Typical histogram values range from -0.01 to +0.01
	absHistogram := math.Abs(histogram)

	// Scale factor - adjust based on typical currency pair volatility
	scaleFactor := 5000.0

	strength := math.Min(absHistogram*scaleFactor, 100)

	return strength
}

// CrossoverEvent represents a MACD crossover event
type CrossoverEvent struct {
	Type          string    // "BULLISH" or "BEARISH"
	MACDLine      float64   // Current MACD line
	SignalLine   float64   // Current signal line
	Histogram    float64   // Current histogram
	ZeroLineCross bool     // Whether this crosses zero line
	Timestamp    time.Time
}

// DetectCrossover detects MACD crossover events
func DetectCrossover(bars []BarData, fastPeriod, slowPeriod, signalPeriod int) *CrossoverEvent {
	requiredBars := slowPeriod + signalPeriod + 1
	if len(bars) < requiredBars+1 {
		return nil
	}

	currentMACD := calculateMACD(bars, fastPeriod, slowPeriod, signalPeriod)
	prevMACD := calculateMACD(bars[:len(bars)-1], fastPeriod, slowPeriod, signalPeriod)

	currentCross := currentMACD.MACDLine - currentMACD.SignalLine
	prevCross := prevMACD.MACDLine - prevMACD.SignalLine

	// Bullish crossover
	if prevCross <= 0 && currentCross > 0 {
		return &CrossoverEvent{
			Type:          "BULLISH",
			MACDLine:      currentMACD.MACDLine,
			SignalLine:    currentMACD.SignalLine,
			Histogram:     currentMACD.Histogram,
			ZeroLineCross: (prevMACD.MACDLine <= 0 && currentMACD.MACDLine > 0),
			Timestamp:     bars[len(bars)-1].Timestamp,
		}
	}

	// Bearish crossover
	if prevCross >= 0 && currentCross < 0 {
		return &CrossoverEvent{
			Type:          "BEARISH",
			MACDLine:      currentMACD.MACDLine,
			SignalLine:    currentMACD.SignalLine,
			Histogram:     currentMACD.Histogram,
			ZeroLineCross: (prevMACD.MACDLine >= 0 && currentMACD.MACDLine < 0),
			Timestamp:     bars[len(bars)-1].Timestamp,
		}
	}

	return nil
}

// GetMACDValues returns current MACD values for a given bar history
func GetMACDValues(bars []BarData, fastPeriod, slowPeriod, signalPeriod int) MACDValues {
	return calculateMACD(bars, fastPeriod, slowPeriod, signalPeriod)
}

// init registers the strategy
func init() {
	RegisterStrategy("macd", NewMACDStrategy)
	RegisterStrategy("MACD", NewMACDStrategy)
}
