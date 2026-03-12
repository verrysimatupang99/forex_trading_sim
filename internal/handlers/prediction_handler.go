package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"forex-trading-sim/internal/services"
)

type PredictionHandler struct {
	predictionService *services.PredictionService
}

func NewPredictionHandler(predictionService *services.PredictionService) *PredictionHandler {
	return &PredictionHandler{predictionService: predictionService}
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

// Public handlers for data endpoints
func GetHistoricalData(c *gin.Context) {
	// Placeholder - would fetch from data service
	c.JSON(http.StatusOK, gin.H{"message": "Historical data endpoint"})
}

func GetTechnicalIndicators(c *gin.Context) {
	// Placeholder - would fetch from data service
	c.JSON(http.StatusOK, gin.H{"message": "Technical indicators endpoint"})
}

func GetCurrencyPairs(c *gin.Context) {
	// Placeholder - would fetch from database
	c.JSON(http.StatusOK, gin.H{"message": "Currency pairs endpoint"})
}
