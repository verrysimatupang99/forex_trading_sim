package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"forex-trading-sim/internal/services"
)

type TradingHandler struct {
	tradingService *services.TradingService
}

func NewTradingHandler(tradingService *services.TradingService) *TradingHandler {
	return &TradingHandler{tradingService: tradingService}
}

func (h *TradingHandler) GetAccounts(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accounts, err := h.tradingService.GetAccounts(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, accounts)
}

func (h *TradingHandler) CreateAccount(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input services.CreateAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := h.tradingService.CreateAccount(userID.(uint), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, account)
}

func (h *TradingHandler) GetBalance(c *gin.Context) {
	accountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	balance, err := h.tradingService.GetBalance(uint(accountID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (h *TradingHandler) ExecuteTrade(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input services.ExecuteTradeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trade, err := h.tradingService.ExecuteTrade(userID.(uint), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, trade)
}

func (h *TradingHandler) GetPositions(c *gin.Context) {
	accountID, err := strconv.ParseUint(c.Query("account_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	positions, err := h.tradingService.GetPositions(uint(accountID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, positions)
}

func (h *TradingHandler) GetTradeHistory(c *gin.Context) {
	accountID, err := strconv.ParseUint(c.Query("account_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	trades, err := h.tradingService.GetTradeHistory(uint(accountID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trades)
}

func (h *TradingHandler) ClosePosition(c *gin.Context) {
	positionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position id"})
		return
	}

	var input struct {
		ExitPrice float64 `json:"exit_price"`
	}
	c.ShouldBindJSON(&input)

	trade, err := h.tradingService.ClosePosition(uint(positionID), input.ExitPrice)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trade)
}

func (h *TradingHandler) RunBacktest(c *gin.Context) {
	var input struct {
		Strategy     string  `json:"strategy" binding:"required"`
		Symbol       string  `json:"symbol" binding:"required"`
		StartDate    string  `json:"start_date" binding:"required"`
		EndDate      string  `json:"end_date" binding:"required"`
		InitialCapital float64 `json:"initial_capital" binding:"required"`
		Timeframe    string  `json:"timeframe"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "backtest endpoint deprecated, use /api/v1/backtest/run instead"})
}

// Deprecated: Use BacktestHandler instead
func (h *TradingHandler) RunBacktestLegacy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "use /api/v1/backtest/run instead"})
}

func (h *TradingHandler) GetBacktestResultsLegacy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "use /api/v1/backtest/results instead"})
}
