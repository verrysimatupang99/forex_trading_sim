package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
	"forex-trading-sim/internal/services"
)

type PredictionHandler struct {
	predictionService *services.PredictionService
	db               *gorm.DB
}

func NewPredictionHandler(predictionService *services.PredictionService, db *gorm.DB) *PredictionHandler {
	return &PredictionHandler{
		predictionService: predictionService,
		db:               db,
	}
}

func (h *PredictionHandler) Predict(c *gin.Context) {
	var input services.PredictInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prediction, err := h.predictionService.Predict(input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prediction)
}

func (h *PredictionHandler) GetPredictionHistory(c *gin.Context) {
	currencyPairID, err := strconv.ParseUint(c.Query("currency_pair_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid currency pair id"})
		return
	}

	limit, _ := strconv.Atoi(c.Query("limit"))

	predictions, err := h.predictionService.GetPredictionHistory(uint(currencyPairID), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, predictions)
}

// GetHistoricalData handles GET /api/v1/historical-data
func GetHistoricalData(c *gin.Context) {
	pairID := c.Query("pair_id")
	timeframe := c.DefaultQuery("timeframe", "1d")
	limit := c.DefaultQuery("limit", "100")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Get DB from context (set by middleware)
	db, exists := c.Get("db")
	if !exists || db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	gormDB := db.(*gorm.DB)

	// Parse pair ID
	pairIDUint, err := strconv.ParseUint(pairID, 10, 32)
	if err != nil && pairID != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pair_id"})
		return
	}

	// Build query
	query := gormDB.Model(&models.HistoricalPrice{})

	if pairID != "" {
		query = query.Where("currency_pair_id = ?", pairIDUint)
	}

	if timeframe != "" {
		query = query.Where("timeframe = ?", timeframe)
	}

	// Parse dates if provided
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("timestamp >= ?", t)
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("timestamp <= ?", t)
		}
	}

	// Parse limit
	limitInt, _ := strconv.Atoi(limit)
	if limitInt <= 0 || limitInt > 1000 {
		limitInt = 100
	}

	var prices []models.HistoricalPrice
	if err := query.Order("timestamp DESC").Limit(limitInt).Find(&prices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch historical data"})
		return
	}

	// Reverse to get chronological order
	for i, j := 0, len(prices)-1; i < j; i, j = i+1, j-1 {
		prices[i], prices[j] = prices[j], prices[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      prices,
		"count":     len(prices),
		"timeframe": timeframe,
	})
}

// GetTechnicalIndicators handles GET /api/v1/technical-indicators
func GetTechnicalIndicators(c *gin.Context) {
	pairID := c.Query("pair_id")
	timeframe := c.DefaultQuery("timeframe", "1d")
	indicator := c.Query("indicator")
	limit := c.DefaultQuery("limit", "100")

	// Get DB from context
	db, exists := c.Get("db")
	if !exists || db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	gormDB := db.(*gorm.DB)

	// Parse pair ID
	pairIDUint, err := strconv.ParseUint(pairID, 10, 32)
	if err != nil && pairID != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pair_id"})
		return
	}

	// Build query
	query := gormDB.Model(&models.TechnicalIndicator{})

	if pairID != "" {
		query = query.Where("currency_pair_id = ?", pairIDUint)
	}

	if timeframe != "" {
		query = query.Where("timeframe = ?", timeframe)
	}

	if indicator != "" {
		query = query.Where("indicator_name LIKE ?", indicator+"%")
	}

	// Parse limit
	limitInt, _ := strconv.Atoi(limit)
	if limitInt <= 0 || limitInt > 1000 {
		limitInt = 100
	}

	var indicators []models.TechnicalIndicator
	if err := query.Order("timestamp DESC").Limit(limitInt).Find(&indicators).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch technical indicators"})
		return
	}

	// Reverse to get chronological order
	for i, j := 0, len(indicators)-1; i < j; i, j = i+1, j-1 {
		indicators[i], indicators[j] = indicators[j], indicators[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  indicators,
		"count": len(indicators),
	})
}

// GetCurrencyPairs handles GET /api/v1/currency-pairs
func GetCurrencyPairs(c *gin.Context) {
	// Get DB from context
	db, exists := c.Get("db")
	if !exists || db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	gormDB := db.(*gorm.DB)

	// Get only active pairs by default
	activeOnly := c.DefaultQuery("active", "true") == "true"

	query := gormDB.Model(&models.CurrencyPair{})

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	var pairs []models.CurrencyPair
	if err := query.Order("symbol ASC").Find(&pairs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch currency pairs"})
		return
	}

	// If no pairs in DB, return default pairs
	if len(pairs) == 0 {
		pairs = []models.CurrencyPair{
			{ID: 1, Symbol: "EUR/USD", BaseCurrency: "EUR", QuoteCurrency: "USD", PipValue: 0.0001, Digits: 4, IsActive: true},
			{ID: 2, Symbol: "GBP/USD", BaseCurrency: "GBP", QuoteCurrency: "USD", PipValue: 0.0001, Digits: 4, IsActive: true},
			{ID: 3, Symbol: "USD/JPY", BaseCurrency: "USD", QuoteCurrency: "JPY", PipValue: 0.01, Digits: 2, IsActive: true},
			{ID: 4, Symbol: "USD/CHF", BaseCurrency: "USD", QuoteCurrency: "CHF", PipValue: 0.0001, Digits: 4, IsActive: true},
			{ID: 5, Symbol: "AUD/USD", BaseCurrency: "AUD", QuoteCurrency: "USD", PipValue: 0.0001, Digits: 4, IsActive: true},
			{ID: 6, Symbol: "USD/CAD", BaseCurrency: "USD", QuoteCurrency: "CAD", PipValue: 0.0001, Digits: 4, IsActive: true},
			{ID: 7, Symbol: "NZD/USD", BaseCurrency: "NZD", QuoteCurrency: "USD", PipValue: 0.0001, Digits: 4, IsActive: true},
			{ID: 8, Symbol: "EUR/GBP", BaseCurrency: "EUR", QuoteCurrency: "GBP", PipValue: 0.0001, Digits: 4, IsActive: true},
			{ID: 9, Symbol: "EUR/JPY", BaseCurrency: "EUR", QuoteCurrency: "JPY", PipValue: 0.01, Digits: 2, IsActive: true},
			{ID: 10, Symbol: "GBP/JPY", BaseCurrency: "GBP", QuoteCurrency: "JPY", PipValue: 0.01, Digits: 2, IsActive: true},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  pairs,
		"count": len(pairs),
	})
}
