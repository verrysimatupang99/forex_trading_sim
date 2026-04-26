package services

import (
	"errors"
	"math"
	"time"

	"forex-trading-sim/internal/models"
)

// DataValidator validates incoming market data
type DataValidator interface {
	Validate(rawData []byte) error
	DetectOutliers(price float64, historical []float64) bool
	ValidatePrice(price float64) error
}

// MarketDataValidator validates forex market data
type MarketDataValidator struct {
	maxPriceChangePercent float64
	zScoreThreshold       float64
}

// NewMarketDataValidator creates a new market data validator
func NewMarketDataValidator() *MarketDataValidator {
	return &MarketDataValidator{
		maxPriceChangePercent: 5.0, // Max 5% change in one candle
		zScoreThreshold:       3.0, // Z-score > 3 is outlier
	}
}

// ValidatePrice validates a single price point
func (v *MarketDataValidator) ValidatePrice(price float64) error {
	if price <= 0 {
		return errors.New("price must be positive")
	}
	if price > 1000000 { // Reasonable upper bound for forex
		return errors.New("price exceeds maximum allowed value")
	}
	return nil
}

// ValidateOHLCV validates OHLCV data
func (v *MarketDataValidator) ValidateOHLCV(data *models.HistoricalPrice) error {
	if err := v.ValidatePrice(data.Open); err != nil {
		return errors.New("invalid open price: " + err.Error())
	}
	if err := v.ValidatePrice(data.High); err != nil {
		return errors.New("invalid high price: " + err.Error())
	}
	if err := v.ValidatePrice(data.Low); err != nil {
		return errors.New("invalid low price: " + err.Error())
	}
	if err := v.ValidatePrice(data.Close); err != nil {
		return errors.New("invalid close price: " + err.Error())
	}

	// High should be >= Open, Close, Low
	if data.High < data.Open || data.High < data.Close || data.High < data.Low {
		return errors.New("high price is inconsistent")
	}

	// Low should be <= Open, Close, High
	if data.Low > data.Open || data.Low > data.Close || data.Low > data.High {
		return errors.New("low price is inconsistent")
	}

	// Volume should be non-negative
	if data.Volume < 0 {
		return errors.New("volume cannot be negative")
	}

	// Validate price changes
	if err := v.validatePriceChanges(data); err != nil {
		return err
	}

	return nil
}

// validatePriceChanges validates that price changes are within acceptable limits
func (v *MarketDataValidator) validatePriceChanges(data *models.HistoricalPrice) error {
	// Calculate percentage change from open to close
	changePercent := math.Abs((data.Close - data.Open) / data.Open * 100)
	if changePercent > v.maxPriceChangePercent {
		return errors.New("price change exceeds maximum allowed percentage")
	}

	// High-Low range should not be excessive
	rangePercent := (data.High - data.Low) / data.Low * 100
	if rangePercent > v.maxPriceChangePercent*2 {
		return errors.New("price range exceeds maximum allowed")
	}

	return nil
}

// DetectOutliers detects if a price is an outlier based on historical data
func (v *MarketDataValidator) DetectOutliers(price float64, historical []float64) bool {
	if len(historical) < 10 {
		return false // Not enough data
	}

	mean, stdDev := calculateMeanStdDev(historical)
	if stdDev == 0 {
		return false
	}

	zScore := math.Abs((price - mean) / stdDev)
	return zScore > v.zScoreThreshold
}

// calculateMeanStdDev calculates mean and standard deviation
func calculateMeanStdDev(data []float64) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(len(data))

	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(data))
	stdDev := math.Sqrt(variance)

	return mean, stdDev
}

// CircuitBreaker for external APIs
type CircuitBreaker struct {
	failures     int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
	state        CircuitState
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failures:     0,
		threshold:    threshold,
		resetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = StateHalfOpen
		} else {
			return errors.New("circuit breaker is open")
		}
	}

	err := fn()

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = StateOpen
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.failures = 0
	cb.state = StateClosed
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.state
}
