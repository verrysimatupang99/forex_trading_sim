package services

import (
	"errors"
	"math"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
)

type PredictionService struct {
	db *gorm.DB
}

func NewPredictionService(db *gorm.DB) *PredictionService {
	return &PredictionService{db: db}
}

type PredictInput struct {
	CurrencyPairID uint   `json:"currency_pair_id" binding:"required"`
	Timeframe      string `json:"timeframe" binding:"required"`
}

type PredictOutput struct {
	Signal      string  `json:"signal"`
	Confidence  float64 `json:"confidence"`
	EntryPrice  float64 `json:"entry_price"`
	TargetPrice float64 `json:"target_price"`
	StopLoss    float64 `json:"stop_loss"`
	TakeProfit  float64 `json:"take_profit"`
	Timestamp   time.Time `json:"timestamp"`
}

func (s *PredictionService) Predict(input PredictInput) (*PredictOutput, error) {
	// Get active ML model
	var model models.MLModel
	if err := s.db.Where("is_active = ?", true).First(&model).Error; err != nil {
		return nil, errors.New("no active model available")
	}

	// Get latest price data for the currency pair
	var price models.HistoricalPrice
	if err := s.db.Where("currency_pair_id = ?", input.CurrencyPairID).
		Order("timestamp DESC").First(&price).Error; err != nil {
		return nil, errors.New("no price data available")
	}

	// In production, this would call the ML model inference
	// For now, we'll simulate a prediction based on simple indicators
	signal, confidence, targetPrice, stopLoss, takeProfit := s.generateSignal(price.Close)

	prediction := models.Prediction{
		ModelID:        model.ID,
		CurrencyPairID: input.CurrencyPairID,
		Signal:         signal,
		Confidence:     confidence,
		EntryPrice:     price.Close,
		TargetPrice:    targetPrice,
		StopLoss:       stopLoss,
		TakeProfit:     takeProfit,
		Timeframe:      input.Timeframe,
		PredictionTime: time.Now(),
	}

	if err := s.db.Create(&prediction).Error; err != nil {
		return nil, err
	}

	return &PredictOutput{
		Signal:      signal,
		Confidence: confidence,
		EntryPrice:  price.Close,
		TargetPrice: targetPrice,
		StopLoss:    stopLoss,
		TakeProfit:  takeProfit,
		Timestamp:   time.Now(),
	}, nil
}

func (s *PredictionService) generateSignal(currentPrice float64) (string, float64, float64, float64, float64) {
	// Simplified signal generation - in production, this would use actual ML inference
	// and technical indicators (RSI, MACD, Moving Averages)

	// Simulate based on random for demo (replace with actual ML prediction)
	signal := "HOLD"
	confidence := 50.0
	targetPrice := currentPrice
	stopLoss := currentPrice * 0.99
	takeProfit := currentPrice * 1.01

	// Simple logic for demonstration
	// In production: use actual model prediction

	return signal, confidence, targetPrice, stopLoss, takeProfit
}

func (s *PredictionService) GetPredictionHistory(currencyPairID uint, limit int) ([]models.Prediction, error) {
	if limit == 0 {
		limit = 50
	}

	var predictions []models.Prediction
	if err := s.db.Where("currency_pair_id = ?", currencyPairID).
		Order("prediction_time DESC").Limit(limit).Find(&predictions).Error; err != nil {
		return nil, err
	}
	return predictions, nil
}

// CalculateTechnicalIndicators computes technical indicators
func (s *PredictionService) CalculateTechnicalIndicators(prices []float64) map[string]float64 {
	indicators := make(map[string]float64)

	if len(prices) < 20 {
		return indicators
	}

	// Simple Moving Averages
	indicators["SMA_20"] = calculateSMA(prices, 20)
	indicators["SMA_50"] = calculateSMA(prices, 50)
	indicators["SMA_200"] = calculateSMA(prices, 200)

	// RSI (14-period)
	indicators["RSI_14"] = calculateRSI(prices, 14)

	// Bollinger Bands
	bbUpper, bbMiddle, bbLower := calculateBollingerBands(prices, 20, 2)
	indicators["BB_Upper"] = bbUpper
	indicators["BB_Middle"] = bbMiddle
	indicators["BB_Lower"] = bbLower

	return indicators
}

func calculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

func calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50.0 // Neutral
	}

	var gains, losses float64
	for i := len(prices) - period; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
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

func calculateBollingerBands(prices []float64, period int, stdDev float64) (float64, float64, float64) {
	if len(prices) < period {
		return 0, 0, 0
	}

	middle := calculateSMA(prices, period)

	// Calculate standard deviation
	var sumSquaredDiff float64
	for i := len(prices) - period; i < len(prices); i++ {
		diff := prices[i] - middle
		sumSquaredDiff += diff * diff
	}
	std := math.Sqrt(sumSquaredDiff / float64(period))

	upper := middle + (stdDev * std)
	lower := middle - (stdDev * std)

	return upper, middle, lower
}
