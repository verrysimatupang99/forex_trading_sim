package strategies

import (
	"errors"
	"math"
	"time"
)

// RSIParams defines parameters for RSI strategy
type RSIParams struct {
	Period     int     `json:"period"`      // RSI period (default: 14)
	Overbought float64 `json:"overbought"`  // Overbought threshold (default: 70)
	Oversold   float64 `json:"oversold"`    // Oversold threshold (default: 30)
	StopLoss   float64 `json:"stop_loss"`   // Stop loss in pips (default: 20)
	TakeProfit float64 `json:"take_profit"` // Take profit in pips (default: 40)
}

// RSIStrategy implements the RSI (Relative Strength Index) strategy
type RSIStrategy struct {
	BaseStrategy
	Period     int
	Overbought float64
	Oversold   float64
	StopLoss   float64
	TakeProfit float64
}

// NewRSIStrategy creates a new RSI strategy
func NewRSIStrategy(params map[string]interface{}) (Strategy, error) {
	p := RSIParams{
		Period:     14,
		Overbought: 70,
		Oversold:   30,
		StopLoss:   20,
		TakeProfit: 40,
	}

	// Override defaults with provided params
	if v, ok := params["period"].(float64); ok {
		p.Period = int(v)
	}
	if v, ok := params["period"].(int); ok {
		p.Period = v
	}
	if v, ok := params["overbought"].(float64); ok {
		p.Overbought = v
	}
	if v, ok := params["oversold"].(float64); ok {
		p.Oversold = v
	}
	if v, ok := params["stop_loss"].(float64); ok {
		p.StopLoss = v
	}
	if v, ok := params["take_profit"].(float64); ok {
		p.TakeProfit = v
	}

	strategy := &RSIStrategy{
		Period:     p.Period,
		Overbought: p.Overbought,
		Oversold:   p.Oversold,
		StopLoss:   p.StopLoss,
		TakeProfit: p.TakeProfit,
	}
	strategy.Name = "RSI"
	strategy.Description = "Buy when RSI enters oversold territory, Sell when RSI enters overbought territory"
	strategy.Parameters = map[string]interface{}{
		"period":      p.Period,
		"overbought":  p.Overbought,
		"oversold":    p.Oversold,
		"stop_loss":   p.StopLoss,
		"take_profit": p.TakeProfit,
	}

	return strategy, nil
}

// ValidateParameters validates the strategy parameters
func (s *RSIStrategy) ValidateParameters() error {
	if s.Period <= 0 {
		return errors.New("period must be greater than 0")
	}
	if s.Overbought <= s.Oversold {
		return errors.New("overbought must be greater than oversold")
	}
	if s.Overbought > 100 || s.Overbought < 0 {
		return errors.New("overbought must be between 0 and 100")
	}
	if s.Oversold > 100 || s.Oversold < 0 {
		return errors.New("oversold must be between 0 and 100")
	}
	if s.StopLoss <= 0 {
		return errors.New("stop_loss must be greater than 0")
	}
	if s.TakeProfit <= 0 {
		return errors.New("take_profit must be greater than 0")
	}
	return nil
}

// OnBar generates a trading signal based on RSI
func (s *RSIStrategy) OnBar(bar BarData, history []BarData, portfolio Portfolio) Signal {
	// Need enough history for RSI calculation
	if len(history) < s.Period+1 {
		return Signal{
			Type:      SignalHold,
			Strength:  0,
			Timestamp: bar.Timestamp,
			Reason:    "Insufficient historical data for RSI",
		}
	}

	// Calculate RSI
	rsi := calculateRSI(history, s.Period)

	// Buy signal: RSI crosses above oversold threshold (was below, now above)
	prevRSI := calculateRSI(history[:len(history)-1], s.Period)
	wasOversold := prevRSI < s.Oversold
	nowAboveOversold := rsi > s.Oversold

	if wasOversold && nowAboveOversold {
		stopLoss := bar.Close - (s.StopLoss * 0.0001)
		takeProfit := bar.Close + (s.TakeProfit * 0.0001)

		return Signal{
			Type:       SignalBuy,
			Strength:   calculateSignalStrength(rsi, s.Oversold, s.Overbought),
			Price:      bar.Close,
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Reason:     "RSI exited oversold territory",
			Timestamp:  bar.Timestamp,
		}
	}

	// Sell signal: RSI crosses below overbought threshold (was above, now below)
	wasOverbought := prevRSI > s.Overbought
	nowBelowOverbought := rsi < s.Overbought

	if wasOverbought && nowBelowOverbought {
		stopLoss := bar.Close + (s.StopLoss * 0.0001)
		takeProfit := bar.Close - (s.TakeProfit * 0.0001)

		return Signal{
			Type:       SignalSell,
			Strength:   calculateSignalStrength(100-rsi, s.Oversold, s.Overbought),
			Price:      bar.Close,
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Reason:     "RSI exited overbought territory",
			Timestamp:  bar.Timestamp,
		}
	}

	// Strong buy: RSI is deeply oversold
	if rsi < s.Oversold {
		return Signal{
			Type:      SignalHold,
			Strength:  30,
			Price:     bar.Close,
			Timestamp: bar.Timestamp,
			Reason:    "RSI in oversold territory - waiting for exit",
		}
	}

	// Strong sell: RSI is deeply overbought
	if rsi > s.Overbought {
		return Signal{
			Type:      SignalHold,
			Strength:  30,
			Price:     bar.Close,
			Timestamp: bar.Timestamp,
			Reason:    "RSI in overbought territory - waiting for exit",
		}
	}

	// Hold: RSI is in neutral territory
	return Signal{
		Type:      SignalHold,
		Strength:  50,
		Price:     bar.Close,
		Timestamp: bar.Timestamp,
		Reason:    "RSI in neutral territory",
	}
}

// OnSignal converts a signal to an order
func (s *RSIStrategy) OnSignal(signal Signal, portfolio Portfolio) *Order {
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
func (s *RSIStrategy) OnTrade(trade BacktestTrade, portfolio *Portfolio) {
	if trade.PnL != 0 {
		portfolio.Equity += trade.PnL
		portfolio.Cash += trade.PnL
	}
}

// calculateRSI calculates Relative Strength Index
func calculateRSI(bars []BarData, period int) float64 {
	if len(bars) < period+1 {
		return 50.0
	}

	var gains, losses float64

	for i := len(bars) - period; i < len(bars); i++ {
		change := bars[i].Close - bars[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses += math.Abs(change)
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateSignalStrength calculates signal strength based on RSI position
func calculateSignalStrength(rsi, oversold, overbought float64) float64 {
	rangeSize := overbought - oversold
	if rangeSize == 0 {
		return 50
	}

	// Calculate how far RSI is from center
	center := (oversold + overbought) / 2
	distance := math.Abs(rsi - center)

	// Normalize to 0-100 scale
	strength := (distance / (rangeSize / 2)) * 100

	// Cap at 100
	if strength > 100 {
		strength = 100
	}

	return strength
}

// Divergence represents RSI divergence signal
type Divergence struct {
	Type      string    // "BULLISH" or "BEARISH"
	PriceDiff float64   // Price change over period
	RSIDiff   float64   // RSI change over period
	Timestamp time.Time
}

// DetectDivergence detects RSI divergence
func DetectDivergence(bars []BarData, period int) *Divergence {
	if len(bars) < period*2+1 {
		return nil
	}

	// Get price and RSI for first half
	firstHalf := bars[:len(bars)/2]
	secondHalf := bars[len(bars)/2:]

	firstPriceChange := firstHalf[len(firstHalf)-1].Close - firstHalf[0].Close
	firstRSI := calculateRSI(firstHalf, period)

	secondPriceChange := secondHalf[len(secondHalf)-1].Close - secondHalf[0].Close
	secondRSI := calculateRSI(secondHalf, period)

	priceDiff := secondPriceChange - firstPriceChange
	rsiDiff := secondRSI - firstRSI

	// Bullish divergence: price makes lower low, RSI makes higher low
	if priceDiff < 0 && rsiDiff > 0 {
		return &Divergence{
			Type:      "BULLISH",
			PriceDiff: priceDiff,
			RSIDiff:   rsiDiff,
			Timestamp: bars[len(bars)-1].Timestamp,
		}
	}

	// Bearish divergence: price makes higher high, RSI makes lower high
	if priceDiff > 0 && rsiDiff < 0 {
		return &Divergence{
			Type:      "BEARISH",
			PriceDiff: priceDiff,
			RSIDiff:   rsiDiff,
			Timestamp: bars[len(bars)-1].Timestamp,
		}
	}

	return nil
}

// init registers the strategy
func init() {
	RegisterStrategy("rsi", NewRSIStrategy)
	RegisterStrategy("RSI", NewRSIStrategy)
}
