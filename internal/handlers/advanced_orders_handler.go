package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"forex-trading-sim/internal/services"
)

// AdvancedOrdersHandler handles advanced order types (OCO, OTO, Pending Orders)
type AdvancedOrdersHandler struct {
	advancedOrderService *services.AdvancedOrderService
}

// NewAdvancedOrdersHandler creates a new advanced orders handler
func NewAdvancedOrdersHandler(advancedOrderService *services.AdvancedOrderService) *AdvancedOrdersHandler {
	return &AdvancedOrdersHandler{advancedOrderService: advancedOrderService}
}

// CreatePendingOrder handles POST /api/v1/orders/pending
func (h *AdvancedOrdersHandler) CreatePendingOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input services.CreatePendingOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.advancedOrderService.CreatePendingOrder(userID.(uint), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetPendingOrders handles GET /api/v1/orders/pending
func (h *AdvancedOrdersHandler) GetPendingOrders(c *gin.Context) {
	accountID, err := strconv.ParseUint(c.Query("account_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	orders, err := h.advancedOrderService.GetPendingOrders(uint(accountID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// CancelPendingOrder handles DELETE /api/v1/orders/pending/:id
func (h *AdvancedOrdersHandler) CancelPendingOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	err = h.advancedOrderService.CancelPendingOrder(uint(orderID), userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order cancelled successfully"})
}

// CreateOCOOrder handles POST /api/v1/orders/oco
func (h *AdvancedOrdersHandler) CreateOCOOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input services.CreateOCOOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.advancedOrderService.CreateOCOOrder(userID.(uint), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetOCOOrders handles GET /api/v1/orders/oco
func (h *AdvancedOrdersHandler) GetOCOOrders(c *gin.Context) {
	accountID, err := strconv.ParseUint(c.Query("account_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	orders, err := h.advancedOrderService.GetOCOOrders(uint(accountID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// CancelOCOOrder handles DELETE /api/v1/orders/oco/:id
func (h *AdvancedOrdersHandler) CancelOCOOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	err = h.advancedOrderService.CancelOCOOrder(uint(orderID), userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OCO order cancelled successfully"})
}

// CreateOTOOrder handles POST /api/v1/orders/oto
func (h *AdvancedOrdersHandler) CreateOTOOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input services.CreateOTOOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.advancedOrderService.CreateOTOOrder(userID.(uint), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetOTOOrders handles GET /api/v1/orders/oto
func (h *AdvancedOrdersHandler) GetOTOOrders(c *gin.Context) {
	accountID, err := strconv.ParseUint(c.Query("account_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	orders, err := h.advancedOrderService.GetOTOOrders(uint(accountID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// CancelOTOOrder handles DELETE /api/v1/orders/oto/:id
func (h *AdvancedOrdersHandler) CancelOTOOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	err = h.advancedOrderService.CancelOTOOrder(uint(orderID), userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTO order cancelled successfully"})
}
