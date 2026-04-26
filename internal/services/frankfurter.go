package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
)

// FrankfurterClient handles data retrieval from Frankfurter API (Free, Open Source)
type FrankfurterClient struct {
	httpClient *http.Client
	rateLimiter *RateLimiter
	db         *gorm.DB
}

// NewFrankfurterClient creates a new Frankfurter client
func NewFrankfurterClient(db *gorm.DB) *FrankfurterClient {
	return &FrankfurterClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		rateLimiter: NewRateLimiter(100, time.Minute), // 100 requests per minute
		db:         db,
	}
}

// FetchLatestRate fetches the latest exchange rate
func (c *FrankfurterClient) FetchLatestRate(baseCurrency, quoteCurrency string) (float64, error) {
	// Wait for rate limiter
	for !c.rateLimiter.Acquire(1) {
		time.Sleep(100 * time.Millisecond)
	}

	url := fmt.Sprintf("https://api.frankfurter.app/latest?from=%s&to=%s", baseCurrency, quoteCurrency)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var result struct {
		Amount   string             `json:"amount"`
		Base     string             `json:"base"`
		Date     string             `json:"date"`
		Rates    map[string]float64 `json:"rates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	rate, exists := result.Rates[quoteCurrency]
	if !exists {
		return 0, fmt.Errorf("rate not found for %s/%s", baseCurrency, quoteCurrency)
	}

	return rate, nil
}

// FetchHistoricalRate fetches historical exchange rate for a specific date
func (c *FrankfurterClient) FetchHistoricalRate(baseCurrency, quoteCurrency string, date time.Time) (float64, error) {
	for !c.rateLimiter.Acquire(1) {
		time.Sleep(100 * time.Millisecond)
	}

	dateStr := date.Format("2006-01-02")
	url := fmt.Sprintf("https://api.frankfurter.app/%s?from=%s&to=%s", dateStr, baseCurrency, quoteCurrency)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch historical rate: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Amount   string             `json:"amount"`
		Base     string             `json:"base"`
		Date     string             `json:"date"`
		Rates    map[string]float64 `json:"rates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	rate, exists := result.Rates[quoteCurrency]
	if !exists {
		return 0, fmt.Errorf("rate not found for %s/%s", baseCurrency, quoteCurrency)
	}

	return rate, nil
}

// FetchTimeSeries fetches historical data for a date range
func (c *FrankfurterClient) FetchTimeSeries(baseCurrency, quoteCurrency string, startDate, endDate time.Time) ([]models.HistoricalPrice, error) {
	for !c.rateLimiter.Acquire(1) {
		time.Sleep(100 * time.Millisecond)
	}

	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")
	
	url := fmt.Sprintf("https://api.frankfurter.app/%s..%s?from=%s&to=%s", 
		startStr, endStr, baseCurrency, quoteCurrency)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch time series: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var result struct {
		Amount   string             `json:"amount"`
		Base     string             `json:"base"`
		StartDate string            `json:"start_date"`
		EndDate   string            `json:"end_date"`
		Rates    map[string]map[string]float64 `json:"rates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var prices []models.HistoricalPrice
	for dateStr, rates := range result.Rates {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		rate, exists := rates[quoteCurrency]
		if !exists {
			continue
		}

		prices = append(prices, models.HistoricalPrice{
			CurrencyPairID: 0, // Will be set by caller
			Timestamp:      date,
			Open:           rate,
			High:           rate,
			Low:            rate,
			Close:          rate,
			Volume:         0,
			Timeframe:      "1d",
		})
	}

	return prices, nil
}

// GetAvailableCurrencies gets list of available currencies
func (c *FrankfurterClient) GetAvailableCurrencies() (map[string]string, error) {
	url := "https://api.frankfurter.app/currencies"

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch currencies: %w", err)
	}
	defer resp.Body.Close()

	var currencies map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&currencies); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return currencies, nil
}

// SaveHistoricalPrices saves historical prices to database
func (c *FrankfurterClient) SaveHistoricalPrices(prices []models.HistoricalPrice, currencyPairID uint) error {
	for i := range prices {
		prices[i].CurrencyPairID = currencyPairID
	}

	return c.db.Create(&prices).Error
}

// FetchAndSaveHistorical fetches and saves historical data for a currency pair
func (c *FrankfurterClient) FetchAndSaveHistorical(baseCurrency, quoteCurrency string, currencyPairID uint, days int) error {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	prices, err := c.FetchTimeSeries(baseCurrency, quoteCurrency, startDate, endDate)
	if err != nil {
		return err
	}

	return c.SaveHistoricalPrices(prices, currencyPairID)
}

// RateLimiter simple rate limiter
type RateLimiter struct {
	tokens    int
	maxTokens int
	lastFill  time.Time
	interval  time.Duration
}

func NewRateLimiter(maxTokens int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:    maxTokens,
		maxTokens: maxTokens,
		lastFill:  time.Now(),
		interval:  interval,
	}
}

func (rl *RateLimiter) Acquire(tokens int) bool {
	now := time.Now()
	elapsed := now.Sub(rl.lastFill)
	
	// Refill tokens
	refill := int(elapsed / rl.interval) * rl.maxTokens
	if refill > 0 {
		rl.tokens = rl.maxTokens
		rl.lastFill = now
	}

	if rl.tokens >= tokens {
		rl.tokens -= tokens
		return true
	}
	return false
}

// FrankfurterDataService wraps Frankfurter API for use in the trading simulator
type FrankfurterDataService struct {
	client *FrankfurterClient
	db     *gorm.DB
}

// NewFrankfurterDataService creates a new Frankfurter data service
func NewFrankfurterDataService(db *gorm.DB) *FrankfurterDataService {
	return &FrankfurterDataService{
		client: NewFrankfurterClient(db),
		db:     db,
	}
}

// GetLatestPrice gets the latest price for a currency pair
func (s *FrankfurterDataService) GetLatestPrice(currencyPairID uint) (float64, error) {
	// Get currency pair from database
	var pair models.CurrencyPair
	if err := s.db.First(&pair, currencyPairID).Error; err != nil {
		return 0, err
	}

	return s.client.FetchLatestRate(pair.BaseCurrency, pair.QuoteCurrency)
}

// GetHistoricalPrices gets historical prices for a currency pair
func (s *FrankfurterDataService) GetHistoricalPrices(currencyPairID uint, days int) ([]models.HistoricalPrice, error) {
	var pair models.CurrencyPair
	if err := s.db.First(&pair, currencyPairID).Error; err != nil {
		return nil, err
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	prices, err := s.client.FetchTimeSeries(pair.BaseCurrency, pair.QuoteCurrency, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Update currency pair ID
	for i := range prices {
		prices[i].CurrencyPairID = currencyPairID
	}

	return prices, nil
}

// SyncHistoricalData syncs historical data for a currency pair
func (s *FrankfurterDataService) SyncHistoricalData(currencyPairID uint, days int) error {
	prices, err := s.GetHistoricalPrices(currencyPairID, days)
	if err != nil {
		return err
	}

	// Save to database (GORM will handle duplicates based on timestamp)
	for _, price := range prices {
		s.db.Where("currency_pair_id = ? AND timestamp = ?", price.CurrencyPairID, price.Timestamp).
			Assign(price).
			FirstOrCreate(&price)
	}

	return nil
}

// GetCurrencies gets available currencies from Frankfurter
func (s *FrankfurterDataService) GetCurrencies() (map[string]string, error) {
	return s.client.GetAvailableCurrencies()
}

// FetchExchangeRate fetches rate between two currencies
func (s *FrankfurterDataService) FetchExchangeRate(from, to string) (float64, error) {
	return s.client.FetchLatestRate(from, to)
}

// FetchHistoricalExchangeRate fetches historical rate for a specific date
func (s *FrankfurterDataService) FetchHistoricalExchangeRate(from, to string, date time.Time) (float64, error) {
	return s.client.FetchHistoricalRate(from, to, date)
}
