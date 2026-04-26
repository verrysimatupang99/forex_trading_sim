package services

import (
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
)

// PriceService handles price fetching and caching
type PriceService struct {
	db          *gorm.DB
	priceCache map[uint]float64
	cacheMutex sync.RWMutex
	lastUpdate map[uint]time.Time
}

// NewPriceService creates a new price service
func NewPriceService(db *gorm.DB) *PriceService {
	return &PriceService{
		db:          db,
		priceCache: make(map[uint]float64),
		lastUpdate: make(map[uint]time.Time),
	}
}

// GetCurrentPrice gets the current price for a currency pair
// Priority: 1. Cache (30s) -> 2. Database latest -> 3. Fallback
func (s *PriceService) GetCurrentPrice(currencyPairID uint) (float64, error) {
	// Check cache first (30 second TTL)
	s.cacheMutex.RLock()
	if cachedPrice, ok := s.priceCache[currencyPairID]; ok {
		if time.Since(s.lastUpdate[currencyPairID]) < 30*time.Second {
			s.cacheMutex.RUnlock()
			return cachedPrice, nil
		}
	}
	s.cacheMutex.RUnlock()

	// Try to get from database
	var price models.HistoricalPrice
	err := s.db.Where("currency_pair_id = ?", currencyPairID).
		Order("timestamp DESC").
		First(&price).Error

	if err == nil {
		s.cacheMutex.Lock()
		s.priceCache[currencyPairID] = price.Close
		s.lastUpdate[currencyPairID] = time.Now()
		s.cacheMutex.Unlock()
		return price.Close, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Return fallback price for simulation (should be replaced with actual market data)
		return s.getFallbackPrice(currencyPairID)
	}

	return 0, errors.New("failed to fetch price: " + err.Error())
}

// getFallbackPrice returns a fallback price for simulation purposes
// In production, this should never be used - real market data should be available
func (s *PriceService) getFallbackPrice(currencyPairID uint) (float64, error) {
	// Get currency pair to determine fallback
	var pair models.CurrencyPair
	if err := s.db.First(&pair, currencyPairID).Error; err != nil {
		return 0, errors.New("currency pair not found")
	}

	// Return realistic fallback prices for major pairs
	// These should only be used for simulation, not real trading
	fallbackPrices := map[string]float64{
		"EUR/USD": 1.0850,
		"GBP/USD": 1.2650,
		"USD/JPY": 149.50,
		"USD/CHF": 0.8820,
		"AUD/USD": 0.6520,
		"USD/CAD": 1.3650,
		"NZD/USD": 0.6120,
		"EUR/GBP": 0.8580,
		"EUR/JPY": 162.20,
		"GBP/JPY": 189.00,
	}

	if price, ok := fallbackPrices[pair.Symbol]; ok {
		return price, nil
	}

	// Default fallback
	return 1.0000, nil
}

// GetHistoricalPrices gets historical price data
func (s *PriceService) GetHistoricalPrices(currencyPairID uint, startDate, endDate time.Time) ([]models.HistoricalPrice, error) {
	var prices []models.HistoricalPrice
	err := s.db.Where("currency_pair_id = ? AND timestamp BETWEEN ? AND ?",
		currencyPairID, startDate, endDate).
		Order("timestamp ASC").
		Find(&prices).Error

	return prices, err
}

// RefreshPriceCache manually refreshes the price cache for a currency pair
func (s *PriceService) RefreshPriceCache(currencyPairID uint) error {
	price, err := s.GetCurrentPrice(currencyPairID)
	if err != nil {
		return err
	}

	s.cacheMutex.Lock()
	s.priceCache[currencyPairID] = price
	s.lastUpdate[currencyPairID] = time.Now()
	s.cacheMutex.Unlock()

	return nil
}

// InvalidateCache invalidates the cache for a currency pair
func (s *PriceService) InvalidateCache(currencyPairID uint) {
	s.cacheMutex.Lock()
	delete(s.priceCache, currencyPairID)
	delete(s.lastUpdate, currencyPairID)
	s.cacheMutex.Unlock()
}
