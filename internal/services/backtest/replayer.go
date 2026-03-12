package backtest

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
)

// DataReplayer handles historical data replay for backtesting
type DataReplayer struct {
	db             *gorm.DB
	currencyPairID uint
	timeframe      string
	startDate      time.Time
	endDate        time.Time
}

// NewDataReplayer creates a new data replayer
func NewDataReplayer(db *gorm.DB, currencyPairID uint, timeframe string, startDate, endDate time.Time) *DataReplayer {
	return &DataReplayer{
		db:             db,
		currencyPairID: currencyPairID,
		timeframe:      timeframe,
		startDate:      startDate,
		endDate:        endDate,
	}
}

// Bar represents a single bar of OHLCV data
type Bar struct {
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timestamp time.Time
}

// GetBars retrieves historical bars for the backtest period
func (r *DataReplayer) GetBars() ([]Bar, error) {
	var prices []models.HistoricalPrice

	err := r.db.Where("currency_pair_id = ? AND timeframe = ? AND timestamp >= ? AND timestamp <= ?",
		r.currencyPairID, r.timeframe, r.startDate, r.endDate).
		Order("timestamp ASC").
		Find(&prices).Error

	if err != nil {
		return nil, errors.New("failed to fetch historical data: " + err.Error())
	}

	// Convert to Bar slice
	bars := make([]Bar, len(prices))
	for i, price := range prices {
		bars[i] = Bar{
			Open:      price.Open,
			High:      price.High,
			Low:       price.Low,
			Close:     price.Close,
			Volume:    price.Volume,
			Timestamp: price.Timestamp,
		}
	}

	return bars, nil
}

// GetBarCount returns the number of bars available
func (r *DataReplayer) GetBarCount() (int, error) {
	var count int64

	err := r.db.Model(&models.HistoricalPrice{}).
		Where("currency_pair_id = ? AND timeframe = ? AND timestamp >= ? AND timestamp <= ?",
			r.currencyPairID, r.timeframe, r.startDate, r.endDate).
		Count(&count).Error

	if err != nil {
		return 0, errors.New("failed to count bars: " + err.Error())
	}

	return int(count), nil
}

// ValidateData checks if sufficient data is available
func (r *DataReplayer) ValidateData() error {
	count, err := r.GetBarCount()
	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("no historical data available for the specified period")
	}

	// Minimum bars required for most strategies
	minBars := 50
	if count < minBars {
		return errors.New("insufficient data: need at least " + string(rune(minBars)) + " bars")
	}

	return nil
}

// GetLatestPrice returns the most recent price
func (r *DataReplayer) GetLatestPrice() (*Bar, error) {
	var price models.HistoricalPrice

	err := r.db.Where("currency_pair_id = ? AND timeframe = ?", r.currencyPairID, r.timeframe).
		Order("timestamp DESC").
		First(&price).Error

	if err != nil {
		return nil, errors.New("no price data available")
	}

	return &Bar{
		Open:      price.Open,
		High:      price.High,
		Low:       price.Low,
		Close:     price.Close,
		Volume:    price.Volume,
		Timestamp: price.Timestamp,
	}, nil
}

// GetPriceRange returns the price range for the period
func (r *DataReplayer) GetPriceRange() (min, max float64, err error) {
	var result struct {
		Min float64
		Max float64
	}

	err = r.db.Model(&models.HistoricalPrice{}).
		Select("MIN(low) as min, MAX(high) as max").
		Where("currency_pair_id = ? AND timeframe = ? AND timestamp >= ? AND timestamp <= ?",
			r.currencyPairID, r.timeframe, r.startDate, r.endDate).
		Scan(&result).Error

	if err != nil {
		return 0, 0, errors.New("failed to get price range: " + err.Error())
	}

	return result.Min, result.Max, nil
}
