package services

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
)

// CurrencyConverter handles currency conversion and exchange rate management
type CurrencyConverter struct {
	db             *gorm.DB
	ratesCache     map[string]*models.CurrencyRate
	cacheMutex     sync.RWMutex
	lastUpdateTime time.Time
	cacheExpiry    time.Duration
}

// NewCurrencyConverter creates a new currency converter
func NewCurrencyConverter(db *gorm.DB) *CurrencyConverter {
	return &CurrencyConverter{
		db:          db,
		ratesCache:  make(map[string]*models.CurrencyRate),
		cacheExpiry: 5 * time.Minute, // Cache expires after 5 minutes
	}
}

// GetExchangeRate gets exchange rate between two currencies
func (c *CurrencyConverter) GetExchangeRate(baseCurrency, quoteCurrency string) (float64, error) {
	if baseCurrency == quoteCurrency {
		return 1.0, nil
	}

	// Check cache first
	cachedRate := c.getCachedRate(baseCurrency, quoteCurrency)
	if cachedRate != nil {
		return cachedRate.Rate, nil
	}

	// Get from database
	rate, err := c.getRateFromDB(baseCurrency, quoteCurrency)
	if err != nil {
		return 0, err
	}

	// Cache the rate
	c.cacheRate(rate)

	return rate.Rate, nil
}

// ConvertAmount converts an amount from one currency to another
func (c *CurrencyConverter) ConvertAmount(amount float64, fromCurrency, toCurrency string) (float64, error) {
	if fromCurrency == toCurrency {
		return amount, nil
	}

	rate, err := c.GetExchangeRate(fromCurrency, toCurrency)
	if err != nil {
		return 0, err
	}

	return amount * rate, nil
}

// ConvertToUSD converts any currency amount to USD
func (c *CurrencyConverter) ConvertToUSD(amount float64, fromCurrency string) (float64, error) {
	return c.ConvertAmount(amount, fromCurrency, "USD")
}

// ConvertFromUSD converts USD amount to target currency
func (c *CurrencyConverter) ConvertFromUSD(amount float64, toCurrency string) (float64, error) {
	return c.ConvertAmount(amount, "USD", toCurrency)
}

// GetCurrencyRate gets full rate details
func (c *CurrencyConverter) GetCurrencyRate(baseCurrency, quoteCurrency string) (*models.CurrencyRate, error) {
	// Check cache
	cachedRate := c.getCachedRate(baseCurrency, quoteCurrency)
	if cachedRate != nil {
		return cachedRate, nil
	}

	// Get from database
	rate, err := c.getRateFromDB(baseCurrency, quoteCurrency)
	if err != nil {
		return nil, err
	}

	// Cache the rate
	c.cacheRate(rate)

	return rate, nil
}

// UpdateRate updates or creates a currency rate
func (c *CurrencyConverter) UpdateRate(baseCurrency, quoteCurrency string, rate, bid, ask, spread float64, source string, isRealTime bool) error {
	uniquePair := fmt.Sprintf("%s_%s", baseCurrency, quoteCurrency)

	var existingRate models.CurrencyRate
	result := c.db.Where("unique_pair = ?", uniquePair).First(&existingRate)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new rate
		newRate := models.CurrencyRate{
			BaseCurrency:  baseCurrency,
			QuoteCurrency: quoteCurrency,
			Rate:          rate,
			Bid:           bid,
			Ask:           ask,
			Spread:        spread,
			Timestamp:     time.Now(),
			Source:        source,
			IsRealTime:    isRealTime,
			UniquePair:    uniquePair,
		}
		return c.db.Create(&newRate).Error
	} else if result.Error != nil {
		return result.Error
	}

	// Update existing rate
	updates := map[string]interface{}{
		"rate":       rate,
		"bid":        bid,
		"ask":        ask,
		"spread":     spread,
		"timestamp":  time.Now(),
		"source":     source,
		"is_real_time": isRealTime,
	}

	return c.db.Model(&existingRate).Updates(updates).Error
}

// GetRatesByBase gets all rates for a base currency
func (c *CurrencyConverter) GetRatesByBase(baseCurrency string) ([]models.CurrencyRate, error) {
	var rates []models.CurrencyRate
	err := c.db.Where("base_currency = ?", baseCurrency).Order("quote_currency ASC").Find(&rates).Error
	return rates, err
}

// GetRatesByQuote gets all rates for a quote currency
func (c *CurrencyConverter) GetRatesByQuote(quoteCurrency string) ([]models.CurrencyRate, error) {
	var rates []models.CurrencyRate
	err := c.db.Where("quote_currency = ?", quoteCurrency).Order("base_currency ASC").Find(&rates).Error
	return rates, err
}

// RefreshCache refreshes the currency rate cache
func (c *CurrencyConverter) RefreshCache() error {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Get recent rates from database
	var rates []models.CurrencyRate
	err := c.db.Where("timestamp > ?", time.Now().Add(-1*time.Hour)).Find(&rates).Error
	if err != nil {
		return err
	}

	// Clear old cache
	c.ratesCache = make(map[string]*models.CurrencyRate)

	// Populate cache with fresh rates
	for i := range rates {
		c.ratesCache[rates[i].UniquePair] = &rates[i]
	}

	c.lastUpdateTime = time.Now()
	return nil
}

// GetCrossRate calculates cross rate between two currencies via USD
func (c *CurrencyConverter) GetCrossRate(currency1, currency2 string) (float64, error) {
	if currency1 == currency2 {
		return 1.0, nil
	}

	// Try direct rate first
	rate, err := c.GetExchangeRate(currency1, currency2)
	if err == nil {
		return rate, nil
	}

	// Calculate cross rate via USD
	rate1ToUSD, err := c.GetExchangeRate(currency1, "USD")
	if err != nil {
		return 0, errors.New("cannot get rate for " + currency1 + " to USD")
	}

	rate2ToUSD, err := c.GetExchangeRate(currency2, "USD")
	if err != nil {
		return 0, errors.New("cannot get rate for " + currency2 + " to USD")
	}

	// Cross rate = rate1ToUSD / rate2ToUSD
	crossRate := rate1ToUSD / rate2ToUSD
	return crossRate, nil
}

// CalculateMultiCurrencyPnL calculates total P&L across multiple currency positions
func (c *CurrencyConverter) CalculateMultiCurrencyPnL(positions []map[string]interface{}, accountBaseCurrency string) (float64, error) {
	totalPnLUSD := 0.0

	for _, pos := range positions {
		pnl, ok := pos["pnl"].(float64)
		if !ok {
			continue
		}

		currency, ok := pos["currency"].(string)
		if !ok {
			currency = accountBaseCurrency
		}

		// Convert P&L to USD
		pnlUSD, err := c.ConvertToUSD(pnl, currency)
		if err != nil {
			continue
		}

		totalPnLUSD += pnlUSD
	}

	return totalPnLUSD, nil
}

// GetCurrencyExposure calculates total exposure per currency
func (c *CurrencyConverter) GetCurrencyExposure(positions []map[string]interface{}, accountBaseCurrency string) (map[string]float64, error) {
	exposure := make(map[string]float64)

	for _, pos := range positions {
		currency, ok := pos["currency"].(string)
		if !ok {
			currency = accountBaseCurrency
		}

		quantity, ok := pos["quantity"].(float64)
		if !ok {
			continue
		}

		entryPrice, ok := pos["entry_price"].(float64)
		if !ok {
			continue
		}

		// Calculate position value in currency
		positionValue := quantity * entryPrice

		// Add to exposure
		if _, exists := exposure[currency]; !exists {
			exposure[currency] = 0
		}
		exposure[currency] += positionValue
	}

	// Convert all exposures to USD
	exposureUSD := make(map[string]float64)
	for currency, value := range exposure {
		valueUSD, err := c.ConvertToUSD(value, currency)
		if err != nil {
			continue
		}
		exposureUSD[currency] = valueUSD
	}

	return exposureUSD, nil
}

// Helper functions

func (c *CurrencyConverter) getCachedRate(baseCurrency, quoteCurrency string) *models.CurrencyRate {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	uniquePair := fmt.Sprintf("%s_%s", baseCurrency, quoteCurrency)
	rate, exists := c.ratesCache[uniquePair]

	if !exists {
		return nil
	}

	// Check if cache is expired
	if time.Since(c.lastUpdateTime) > c.cacheExpiry {
		return nil
	}

	return rate
}

func (c *CurrencyConverter) cacheRate(rate *models.CurrencyRate) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.ratesCache[rate.UniquePair] = rate
	c.lastUpdateTime = time.Now()
}

func (c *CurrencyConverter) getRateFromDB(baseCurrency, quoteCurrency string) (*models.CurrencyRate, error) {
	uniquePair := fmt.Sprintf("%s_%s", baseCurrency, quoteCurrency)

	var rate models.CurrencyRate
	err := c.db.Where("unique_pair = ?", uniquePair).Order("timestamp DESC").First(&rate).Error

	if err == nil {
		return &rate, nil
	}

	if err == gorm.ErrRecordNotFound {
		// Try reverse pair
		reversePair := fmt.Sprintf("%s_%s", quoteCurrency, baseCurrency)
		var reverseRate models.CurrencyRate
		err := c.db.Where("unique_pair = ?", reversePair).Order("timestamp DESC").First(&reverseRate).Error

		if err == nil {
			// Return inverse rate
			inverseRate := &models.CurrencyRate{
				BaseCurrency:  baseCurrency,
				QuoteCurrency: quoteCurrency,
				Rate:          1.0 / reverseRate.Rate,
				Bid:           1.0 / reverseRate.Ask,
				Ask:           1.0 / reverseRate.Bid,
				Spread:        reverseRate.Spread,
				Timestamp:     reverseRate.Timestamp,
				UniquePair:    uniquePair,
			}
			return inverseRate, nil
		}
	}

	return nil, errors.New("exchange rate not found for " + baseCurrency + "/" + quoteCurrency)
}
