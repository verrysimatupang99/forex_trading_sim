package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
	"forex-trading-sim/internal/services/backtest"
	"forex-trading-sim/internal/services/strategies"
)

// BacktestHandler handles backtesting API requests
type BacktestHandler struct {
	db *gorm.DB
}

// NewBacktestHandler creates a new backtest handler
func NewBacktestHandler(db *gorm.DB) *BacktestHandler {
	return &BacktestHandler{db: db}
}

// RunBacktestRequest represents the request body for running a backtest
type RunBacktestRequest struct {
	StrategyName    string                 `json:"strategy_name" binding:"required"`
	Parameters      map[string]interface{} `json:"parameters"`
	CurrencyPairID  uint                   `json:"currency_pair_id" binding:"required"`
	Timeframe       string                 `json:"timeframe" binding:"required"`
	StartDate       string                 `json:"start_date" binding:"required"`
	EndDate         string                 `json:"end_date" binding:"required"`
	InitialCapital  float64                `json:"initial_capital" binding:"required"`
	Commission      float64                `json:"commission"`
	SlippagePips   float64                `json:"slippage_pips"`
	SpreadPips     float64                `json:"spread_pips"`
	PositionSizing string                 `json:"position_sizing"`
}

// RunBacktest executes a backtest
// @Summary Run a backtest
// @Description Run a backtest with specified strategy and parameters
// @Tags backtest
// @Accept json
// @Produce json
// @Param request body RunBacktestRequest true "Backtest request"
// @Success 200 {object} backtest.BacktestResult
// @Router /api/v1/backtest/run [post]
func (h *BacktestHandler) RunBacktest(c *gin.Context) {
	var req RunBacktestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, use YYYY-MM-DD"})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, use YYYY-MM-DD"})
		return
	}

	// Validate dates
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_date must be after start_date"})
		return
	}

	// Set defaults
	if req.Commission == 0 {
		req.Commission = 2.0 // Default 2 pips
	}
	if req.SlippagePips == 0 {
		req.SlippagePips = 1.0
	}
	if req.SpreadPips == 0 {
		req.SpreadPips = 2.0
	}
	if req.PositionSizing == "" {
		req.PositionSizing = "FIXED_LOT"
	}

	// Create backtest config
	config := backtest.BacktestConfig{
		StrategyName:    req.StrategyName,
		Parameters:      req.Parameters,
		CurrencyPairID:  req.CurrencyPairID,
		Timeframe:       req.Timeframe,
		StartDate:       startDate,
		EndDate:         endDate,
		InitialCapital:  req.InitialCapital,
		Commission:      req.Commission,
		SlippagePips:    req.SlippagePips,
		SpreadPips:      req.SpreadPips,
		PositionSizing: backtest.PositionSizingConfig{
			Mode: req.PositionSizing,
		},
	}

	// Create engine and run
	engine, err := backtest.NewBacktestEngine(h.db, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create backtest engine: " + err.Error()})
		return
	}

	// Set strategy
	err = engine.SetStrategy(req.StrategyName, req.Parameters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid strategy: " + err.Error()})
		return
	}

	// Run backtest
	result, err := engine.Run()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "backtest failed: " + err.Error()})
		return
	}

	// Save backtest result to database
	backtestRecord := models.Backtest{
		Name:            req.StrategyName + " Backtest",
		StrategyName:    req.StrategyName,
		Parameters:      serializeParams(req.Parameters),
		CurrencyPairID:  req.CurrencyPairID,
		Timeframe:       req.Timeframe,
		StartDate:       startDate,
		EndDate:         endDate,
		InitialCapital:  req.InitialCapital,
		TotalReturn:     result.Metrics.TotalReturn,
		SharpeRatio:    result.Metrics.SharpeRatio,
		SortinoRatio:   result.Metrics.SortinoRatio,
		MaxDrawdown:    result.Metrics.MaxDrawdown,
		MaxDrawdownDuration: result.Metrics.MaxDrawdownDuration,
		WinRate:        result.Metrics.WinRate,
		ProfitFactor:   result.Metrics.ProfitFactor,
		NumTrades:      len(result.Trades),
		Trades:         serializeTrades(result.Trades),
		EquityCurve:    serializeEquityCurve(extractEquityValues(result.EquityCurve)),
		Status:         "completed",
	}

	if err := h.db.Create(&backtestRecord).Error; err != nil {
		// Log but don't fail
		c.JSON(http.StatusOK, gin.H{
			"backtest": result,
			"warning":  "backtest completed but failed to save to database",
		})
		return
	}

	result.ID = backtestRecord.ID

	c.JSON(http.StatusOK, result)
}

// GetBacktestResults returns all backtest results
// @Summary Get all backtest results
// @Description Retrieve all backtest results
// @Tags backtest
// @Produce json
// @Success 200 {array} models.Backtest
// @Router /api/v1/backtest/results [get]
func (h *BacktestHandler) GetBacktestResults(c *gin.Context) {
	var results []models.Backtest
	
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	err := h.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&results).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetBacktestResult returns a specific backtest result
// @Summary Get a specific backtest
// @Description Retrieve a specific backtest by ID
// @Tags backtest
// @Produce json
// @Param id path int true "Backtest ID"
// @Success 200 {object} models.Backtest
// @Router /api/v1/backtest/results/{id} [get]
func (h *BacktestHandler) GetBacktestResult(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var result models.Backtest
	if err := h.db.First(&result, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "backtest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetEquityCurve returns equity curve data for a backtest
// @Summary Get equity curve
// @Description Retrieve equity curve data for charting
// @Tags backtest
// @Produce json
// @Param id path int true "Backtest ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/backtest/equity-curve/{id} [get]
func (h *BacktestHandler) GetEquityCurve(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var result models.Backtest
	if err := h.db.First(&result, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "backtest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	equityCurve, err := deserializeEquityCurve(result.EquityCurve)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse equity curve"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"equity_curve": equityCurve,
		"initial_capital": result.InitialCapital,
		"final_capital": result.InitialCapital * (1 + result.TotalReturn/100),
	})
}

// GetBacktestTrades returns trades for a specific backtest
// @Summary Get backtest trades
// @Description Retrieve all trades from a backtest
// @Tags backtest
// @Produce json
// @Param id path int true "Backtest ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/backtest/trades/{id} [get]
func (h *BacktestHandler) GetBacktestTrades(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var result models.Backtest
	if err := h.db.First(&result, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "backtest not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	trades, err := deserializeTrades(result.Trades)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse trades"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trades":      trades,
		"total_trades": len(trades),
		"win_rate":    result.WinRate,
		"profit_factor": result.ProfitFactor,
	})
}

// Helper functions for serialization
func serializeParams(params map[string]interface{}) string {
	data, _ := json.Marshal(params)
	return string(data)
}

func serializeTrades(trades []strategies.BacktestTrade) string {
	data, _ := json.Marshal(trades)
	return string(data)
}

func serializeEquityCurve(curve []float64) string {
	data, _ := json.Marshal(curve)
	return string(data)
}

func deserializeEquityCurve(data string) ([]float64, error) {
	var curve []float64
	err := json.Unmarshal([]byte(data), &curve)
	return curve, err
}

func deserializeTrades(data string) ([]map[string]interface{}, error) {
	var trades []map[string]interface{}
	err := json.Unmarshal([]byte(data), &trades)
	return trades, err
}

// extractEquityValues extracts equity values from EquityPoint slice
func extractEquityValues(equityPoints []backtest.EquityPoint) []float64 {
	values := make([]float64, len(equityPoints))
	for i, ep := range equityPoints {
		values[i] = ep.Equity
	}
	return values
}

// WalkForwardRequest represents the request body for walk-forward analysis
type WalkForwardRequest struct {
	StrategyName       string                 `json:"strategy_name" binding:"required"`
	Parameters         map[string]interface{} `json:"parameters"`
	CurrencyPairID     uint                   `json:"currency_pair_id" binding:"required"`
	Timeframe          string                 `json:"timeframe" binding:"required"`
	StartDate          string                 `json:"start_date" binding:"required"`
	EndDate            string                 `json:"end_date" binding:"required"`
	InitialCapital     float64                `json:"initial_capital" binding:"required"`
	TrainingPeriodDays int                    `json:"training_period_days" binding:"required,min=30"`
	TestingPeriodDays  int                    `json:"testing_period_days" binding:"required,min=7"`
	StepForwardDays    int                    `json:"step_forward_days" binding:"required,min=1"`
	Commission         float64                `json:"commission"`
	SlippagePips       float64                `json:"slippage_pips"`
	SpreadPips         float64                `json:"spread_pips"`
	PositionSizingMode string                 `json:"position_sizing_mode"` // "FIXED_LOT" or "PERCENT_EQUITY"
	FixedLot           float64                `json:"fixed_lot"`
	EquityPercent      float64                `json:"equity_percent"`
}

// RunWalkForward executes walk-forward analysis
// @Summary Run walk-forward analysis
// @Description Run walk-forward analysis with rolling window optimization
// @Tags backtest
// @Accept json
// @Produce json
// @Param request body WalkForwardRequest true "Walk-forward request"
// @Success 200 {object} backtest.WalkForwardResult
// @Router /api/v1/walkforward/run [post]
func (h *BacktestHandler) RunWalkForward(c *gin.Context) {
	var req WalkForwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, use YYYY-MM-DD"})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, use YYYY-MM-DD"})
		return
	}

	// Set defaults
	if req.Commission == 0 {
		req.Commission = 2.0
	}
	if req.SlippagePips == 0 {
		req.SlippagePips = 1.0
	}
	if req.SpreadPips == 0 {
		req.SpreadPips = 2.0
	}
	if req.PositionSizingMode == "" {
		req.PositionSizingMode = "FIXED_LOT"
		req.FixedLot = 0.1
	}

	// Create walk-forward config
	config := backtest.WalkForwardConfig{
		StrategyName:       req.StrategyName,
		Parameters:         req.Parameters,
		CurrencyPairID:     req.CurrencyPairID,
		Timeframe:          req.Timeframe,
		StartDate:          startDate,
		EndDate:            endDate,
		InitialCapital:     req.InitialCapital,
		TrainingPeriodDays: req.TrainingPeriodDays,
		TestingPeriodDays:  req.TestingPeriodDays,
		StepForwardDays:    req.StepForwardDays,
		Commission:         req.Commission,
		SlippagePips:       req.SlippagePips,
		SpreadPips:         req.SpreadPips,
		PositionSizingMode: req.PositionSizingMode,
		FixedLot:           req.FixedLot,
		EquityPercent:      req.EquityPercent,
	}

	// Run walk-forward analysis
	service := backtest.NewWalkForwardService(h.db)
	result, err := service.Run(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "walk-forward analysis failed: " + err.Error()})
		return
	}

	// Save to database
	if err := result.Save(h.db); err != nil {
		// Log but don't fail
		c.JSON(http.StatusOK, gin.H{
			"walk_forward": result,
			"warning":      "analysis completed but failed to save to database",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetWalkForwardResults returns all walk-forward analysis results
// @Summary Get all walk-forward results
// @Description Retrieve all walk-forward analysis results
// @Tags backtest
// @Produce json
// @Success 200 {array} models.WalkForwardAnalysis
// @Router /api/v1/walkforward/results [get]
func (h *BacktestHandler) GetWalkForwardResults(c *gin.Context) {
	var results []models.WalkForwardAnalysis

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	err := h.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&results).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}
