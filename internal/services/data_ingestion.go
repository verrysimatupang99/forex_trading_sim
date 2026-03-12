package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
	"forex-trading-sim/internal/utils"
)

// AlphaVantageClient handles data retrieval from Alpha Vantage API
type AlphaVantageClient struct {
	apiKey     string
	httpClient *utils.HTTPClient
	rateLimiter *utils.RateLimiter
	circuitBreaker *CircuitBreaker
	db         *gorm.DB
}

// NewAlphaVantageClient creates a new Alpha Vantage client
func NewAlphaVantageClient(apiKey string, db *gorm.DB) *AlphaVantageClient {
	return &AlphaVantageClient{
		apiKey:     apiKey,
		httpClient: utils.NewHTTPClient(nil),
		rateLimiter: utils.NewRateLimiter(5, 5), // 5 requests per minute
		circuitBreaker: NewCircuitBreaker(5, 5 * time.Minute),
		db:         db,
	}
}

// FetchHistoricalData fetches historical forex data
func (c *AlphaVantageClient) FetchHistoricalData(ctx context.Context, symbol string, outputSize string) ([]models.HistoricalPrice, error) {
	// Wait for rate limiter
	for !c.rateLimiter.Acquire(1) {
		time.Sleep(100 * time.Millisecond)
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%s&outputsize=%s&apikey=%s", 
		symbol, outputSize, c.apiKey)

	var result struct {
		TimeSeries map[string]struct {
			Open   string `json:"1. open"`
			High   string `json:"2. high"`
			Low    string `json:"3. low"`
			Close  string `json:"4. close"`
			Volume string `json:"5. volume"`
		} `json:"Time Series (Daily)"`
		ErrorMessage string `json:"Error Message"`
	}

	err := c.circuitBreaker.Execute(func() error {
		resp, err := c.httpClient.RetryRequest(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return json.NewDecoder(resp.Body).Decode(&result)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical data: %w", err)
	}

	if result.ErrorMessage != "" {
		return nil, fmt.Errorf("API error: %s", result.ErrorMessage)
	}

	// Parse the time series data
	var prices []models.HistoricalPrice
	for dateStr, data := range result.TimeSeries {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("Failed to parse date: %s", dateStr)
			continue
		}

		open := parseFloat(data.Open)
		high := parseFloat(data.High)
		low := parseFloat(data.Low)
		close := parseFloat(data.Close)
		volume := parseFloat(data.Volume)

		// Validate data
		validator := NewMarketDataValidator()
		priceData := &models.HistoricalPrice{
			Open:     open,
			High:     high,
			Low:      low,
			Close:    close,
			Volume:   volume,
			Timeframe: "1d",
		}

		if err := validator.ValidateOHLCV(priceData); err != nil {
			log.Printf("Invalid price data for %s: %v", dateStr, err)
			continue
		}

		prices = append(prices, models.HistoricalPrice{
			CurrencyPairID: 1, // Will be set based on symbol
			Timestamp:      date,
			Open:           open,
			High:           high,
			Low:            low,
			Close:          close,
			Volume:         volume,
			Timeframe:      "1d",
		})
	}

	return prices, nil
}

// FetchTechnicalIndicator fetches technical indicators from Alpha Vantage
func (c *AlphaVantageClient) FetchTechnicalIndicator(ctx context.Context, symbol, indicator, period string) (map[string]float64, error) {
	// Wait for rate limiter
	for !c.rateLimiter.Acquire(1) {
		time.Sleep(100 * time.Millisecond)
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=%s&symbol=%s&interval=daily&time_period=%s&series_type=close&apikey=%s",
		indicator, symbol, period, c.apiKey)

	var result struct {
		Technical map[string]struct {
			Value string `json:"value"`
		} `json:"Technical Analysis: %s"`
		ErrorMessage string `json:"Error Message"`
	}

	err := c.circuitBreaker.Execute(func() error {
		resp, err := c.httpClient.RetryRequest(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return json.NewDecoder(resp.Body).Decode(&result)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch technical indicator: %w", err)
	}

	values := make(map[string]float64)
	for date, data := range result.Technical {
		value := parseFloat(data.Value)
		values[date] = value
	}

	return values, nil
}

// OANDAClient handles data retrieval from OANDA API
type OANDAClient struct {
	apiKey     string
	accountID  string
	httpClient *utils.HTTPClient
	rateLimiter *utils.RateLimiter
	circuitBreaker *CircuitBreaker
	db         *gorm.DB
}

// NewOANDAClient creates a new OANDA client
func NewOANDAClient(apiKey, accountID string, db *gorm.DB) *OANDAClient {
	return &OANDAClient{
		apiKey:     apiKey,
		accountID:  accountID,
		httpClient: utils.NewHTTPClient(nil),
		rateLimiter: utils.NewRateLimiter(20, 20), // 20 requests per second
		circuitBreaker: NewCircuitBreaker(10, 1 * time.Minute),
		db:         db,
	}
}

// FetchRealTimePrice fetches real-time price from OANDA
func (c *OANDAClient) FetchRealTimePrice(ctx context.Context, instrument string) (float64, error) {
	// Wait for rate limiter
	for !c.rateLimiter.Acquire(1) {
		time.Sleep(50 * time.Millisecond)
	}

	url := fmt.Sprintf("https://api-fxpractice.oanda.com/v3/accounts/%s/pricing?instruments=%s",
		c.accountID, instrument)

	var result struct {
		Prices []struct {
			Bids []struct {
				Price string `json:"price"`
			} `json:"bids"`
			Asks []struct {
				Price string `json:"price"`
			} `json:"asks"`
		} `json:"prices"`
	}

	err := c.circuitBreaker.Execute(func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		return json.NewDecoder(resp.Body).Decode(&result)
	})

	if err != nil {
		return 0, fmt.Errorf("failed to fetch real-time price: %w", err)
	}

	if len(result.Prices) == 0 {
		return 0, fmt.Errorf("no price data returned")
	}

	// Use bid price
	bid := result.Prices[0].Bids[0].Price
	return parseFloat(bid), nil
}

// DataIngestionService coordinates data from multiple sources
type DataIngestionService struct {
	db              *gorm.DB
	alphaVantage    *AlphaVantageClient
	oanda           *OANDAClient
	validator       *MarketDataValidator
}

// NewDataIngestionService creates a new data ingestion service
func NewDataIngestionService(db *gorm.DB, alphaVantageKey, oandaKey, oandaAccountID string) *DataIngestionService {
	return &DataIngestionService{
		db:            db,
		alphaVantage:  NewAlphaVantageClient(alphaVantageKey, db),
		oanda:         NewOANDAClient(oandaKey, oandaAccountID, db),
		validator:     NewMarketDataValidator(),
	}
}

// IngestHistoricalData fetches and stores historical data
func (s *DataIngestionService) IngestHistoricalData(ctx context.Context, pairID uint, symbol string) error {
	prices, err := s.alphaVantage.FetchHistoricalData(ctx, symbol, "full")
	if err != nil {
		return err
	}

	// Store prices in database
	for i := range prices {
		prices[i].CurrencyPairID = pairID
	}

	if err := s.db.Create(&prices).Error; err != nil {
		// Check for duplicate errors
		if !strings.Contains(err.Error(), "duplicate") {
			return err
		}
		log.Printf("Some prices already exist, skipping duplicates")
	}

	return nil
}

// GetLatestPrice gets the latest price for a currency pair
func (s *DataIngestionService) GetLatestPrice(pairID uint) (*models.HistoricalPrice, error) {
	var price models.HistoricalPrice
	err := s.db.Where("currency_pair_id = ?", pairID).
		Order("timestamp DESC").
		First(&price).Error

	if err != nil {
		return nil, err
	}
	return &price, nil
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
