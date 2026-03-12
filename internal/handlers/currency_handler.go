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

// CurrencyHandler handles currency conversion API requests
type CurrencyHandler struct {
	converter *services.CurrencyConverter
	db        *gorm.DB
}

// NewCurrencyHandler creates a new currency handler
func NewCurrencyHandler(converter *services.CurrencyConverter, db *gorm.DB) *CurrencyHandler {
	return &CurrencyHandler{
		converter: converter,
		db:        db,
	}
}

// GetExchangeRateInput represents the input for getting exchange rate
type GetExchangeRateInput struct {
	BaseCurrency  string `json:"base_currency" binding:"required"`
	QuoteCurrency string `json:"quote_currency" binding:"required"`
}

// GetExchangeRate handles GET /api/v1/currency/rate
// @Summary Get exchange rate
// @Description Get exchange rate between two currencies
// @Tags currency
// @Accept json
// @Produce json
// @Param base_currency query string true "Base currency (e.g., USD)"
// @Param quote_currency query string true "Quote currency (e.g., EUR)"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/currency/rate [get]
func (h *CurrencyHandler) GetExchangeRate(c *gin.Context) {
	baseCurrency := c.Query("base_currency")
	quoteCurrency := c.Query("quote_currency")

	if baseCurrency == "" || quoteCurrency == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "base_currency and quote_currency are required"})
		return
	}

	rate, err := h.converter.GetExchangeRate(baseCurrency, quoteCurrency)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"base_currency":  baseCurrency,
		"quote_currency": quoteCurrency,
		"rate":           rate,
		"timestamp":      time.Now(),
	})
}

// ConvertCurrencyInput represents the input for currency conversion
type ConvertCurrencyInput struct {
	Amount        float64 `json:"amount" binding:"required"`
	FromCurrency  string  `json:"from_currency" binding:"required"`
	ToCurrency    string  `json:"to_currency" binding:"required"`
}

// ConvertCurrency handles POST /api/v1/currency/convert
// @Summary Convert currency
// @Description Convert amount from one currency to another
// @Tags currency
// @Accept json
// @Produce json
// @Param request body ConvertCurrencyInput true "Conversion request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/currency/convert [post]
func (h *CurrencyHandler) ConvertCurrency(c *gin.Context) {
	var input ConvertCurrencyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	convertedAmount, err := h.converter.ConvertAmount(input.Amount, input.FromCurrency, input.ToCurrency)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"original_amount":  input.Amount,
		"from_currency":    input.FromCurrency,
		"to_currency":      input.ToCurrency,
		"converted_amount": convertedAmount,
		"timestamp":        time.Now(),
	})
}

// UpdateRateInput represents the input for updating currency rate
type UpdateRateInput struct {
	BaseCurrency  string  `json:"base_currency" binding:"required"`
	QuoteCurrency string  `json:"quote_currency" binding:"required"`
	Rate          float64 `json:"rate" binding:"required"`
	Bid           float64 `json:"bid" binding:"required"`
	Ask           float64 `json:"ask" binding:"required"`
	Spread        float64 `json:"spread"`
	Source        string  `json:"source"`
	IsRealTime    bool    `json:"is_real_time"`
}

// UpdateRate handles POST /api/v1/currency/rate
// @Summary Update currency rate
// @Description Update or create a currency exchange rate
// @Tags currency
// @Accept json
// @Produce json
// @Param request body UpdateRateInput true "Rate update request"
// @Success 200 {object} models.CurrencyRate
// @Router /api/v1/currency/rate [post]
func (h *CurrencyHandler) UpdateRate(c *gin.Context) {
	var input UpdateRateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Spread == 0 {
		input.Spread = (input.Ask - input.Bid) * 10000 // Convert to pips
	}

	if input.Source == "" {
		input.Source = "manual"
	}

	err := h.converter.UpdateRate(
		input.BaseCurrency,
		input.QuoteCurrency,
		input.Rate,
		input.Bid,
		input.Ask,
		input.Spread,
		input.Source,
		input.IsRealTime,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Rate updated successfully",
		"rate":    input.Rate,
		"pair":    input.BaseCurrency + "/" + input.QuoteCurrency,
	})
}

// GetRates handles GET /api/v1/currency/rates
// @Summary Get all currency rates
// @Description Get all currency rates with optional filters
// @Tags currency
// @Produce json
// @Param base_currency query string false "Filter by base currency"
// @Param quote_currency query string false "Filter by quote currency"
// @Success 200 {array} models.CurrencyRate
// @Router /api/v1/currency/rates [get]
func (h *CurrencyHandler) GetRates(c *gin.Context) {
	baseCurrency := c.Query("base_currency")
	quoteCurrency := c.Query("quote_currency")

	var rates []models.CurrencyRate
	query := h.db.Model(&models.CurrencyRate{})

	if baseCurrency != "" {
		query = query.Where("base_currency = ?", baseCurrency)
	}
	if quoteCurrency != "" {
		query = query.Where("quote_currency = ?", quoteCurrency)
	}

	err := query.Order("updated_at DESC").Limit(100).Find(&rates).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rates)
}

// GetCrossRate handles GET /api/v1/currency/cross
// @Summary Get cross rate
// @Description Get cross rate between two currencies via USD
// @Tags currency
// @Produce json
// @Param currency1 query string true "First currency"
// @Param currency2 query string true "Second currency"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/currency/cross [get]
func (h *CurrencyHandler) GetCrossRate(c *gin.Context) {
	currency1 := c.Query("currency1")
	currency2 := c.Query("currency2")

	if currency1 == "" || currency2 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "currency1 and currency2 are required"})
		return
	}

	crossRate, err := h.converter.GetCrossRate(currency1, currency2)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"currency1":  currency1,
		"currency2":  currency2,
		"cross_rate": crossRate,
		"timestamp":  time.Now(),
	})
}

// CreateMultiCurrencyAccountInput represents the input for creating multi-currency account
type CreateMultiCurrencyAccountInput struct {
	UserID       uint    `json:"user_id" binding:"required"`
	BaseCurrency string  `json:"base_currency" binding:"required"`
	InitialBalance float64 `json:"initial_balance" binding:"required,min=0"`
}

// CreateMultiCurrencyAccount handles POST /api/v1/currency/account
// @Summary Create multi-currency account
// @Description Create a new multi-currency trading account
// @Tags currency
// @Accept json
// @Produce json
// @Param request body CreateMultiCurrencyAccountInput true "Account creation request"
// @Success 200 {object} models.MultiCurrencyAccount
// @Router /api/v1/currency/account [post]
func (h *CurrencyHandler) CreateMultiCurrencyAccount(c *gin.Context) {
	var input CreateMultiCurrencyAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user exists
	var user models.User
	if err := h.db.First(&user, input.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Convert initial balance to USD
	initialBalanceUSD, err := h.converter.ConvertToUSD(input.InitialBalance, input.BaseCurrency)
	if err != nil {
		initialBalanceUSD = input.InitialBalance // Fallback to original amount
	}

	// Create multi-currency account
	account := models.MultiCurrencyAccount{
		UserID:          input.UserID,
		BaseCurrency:    input.BaseCurrency,
		TotalBalanceUSD: initialBalanceUSD,
		TotalEquityUSD:  initialBalanceUSD,
		MarginUsedUSD:   0,
		FreeMarginUSD:   initialBalanceUSD,
	}

	if err := h.db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create initial currency balance
	balance := models.CurrencyBalance{
		AccountID:   account.ID,
		Currency:    input.BaseCurrency,
		Balance:     input.InitialBalance,
		Reserved:    0,
		Equity:      input.InitialBalance,
		RateToUSD:   initialBalanceUSD / input.InitialBalance,
		BalanceUSD:  initialBalanceUSD,
	}

	if err := h.db.Create(&balance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, account)
}

// GetMultiCurrencyAccount handles GET /api/v1/currency/account/:id
// @Summary Get multi-currency account
// @Description Get multi-currency account details with currency balances
// @Tags currency
// @Produce json
// @Param id path int true "Account ID"
// @Success 200 {object} models.MultiCurrencyAccount
// @Router /api/v1/currency/account/:id [get]
func (h *CurrencyHandler) GetMultiCurrencyAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	var account models.MultiCurrencyAccount
	if err := h.db.Preload("CurrencyBalances").Preload("User").First(&account, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, account)
}

// GetCurrencyExposure handles GET /api/v1/currency/exposure/:account_id
// @Summary Get currency exposure
// @Description Get currency exposure for a multi-currency account
// @Tags currency
// @Produce json
// @Param account_id path int true "Account ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/currency/exposure/:account_id [get]
func (h *CurrencyHandler) GetCurrencyExposure(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("account_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	// Get account balances
	var balances []models.CurrencyBalance
	if err := h.db.Where("account_id = ?", id).Find(&balances).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate exposure
	exposure := make(map[string]float64)
	totalExposureUSD := 0.0

	for _, balance := range balances {
		exposure[balance.Currency] = balance.BalanceUSD
		totalExposureUSD += balance.BalanceUSD
	}

	c.JSON(http.StatusOK, gin.H{
		"account_id":         id,
		"currency_exposure":  exposure,
		"total_exposure_usd": totalExposureUSD,
		"timestamp":          time.Now(),
	})
}

// RefreshRates handles POST /api/v1/currency/refresh
// @Summary Refresh currency rates cache
// @Description Refresh the currency rates cache from database
// @Tags currency
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/currency/refresh [post]
func (h *CurrencyHandler) RefreshRates(c *gin.Context) {
	err := h.converter.RefreshCache()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Currency rates cache refreshed successfully",
		"timestamp": time.Now(),
	})
}
