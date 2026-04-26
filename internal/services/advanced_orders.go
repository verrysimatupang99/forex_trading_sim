package services

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
)

// AdvancedOrderService handles advanced order types (OCO, OTO, Pending Orders)
type AdvancedOrderService struct {
	db *gorm.DB
}

// NewAdvancedOrderService creates a new advanced order service
func NewAdvancedOrderService(db *gorm.DB) *AdvancedOrderService {
	return &AdvancedOrderService{db: db}
}

// CreatePendingOrderInput represents input for creating a pending order
type CreatePendingOrderInput struct {
	AccountID      uint     `json:"account_id" binding:"required"`
	CurrencyPairID uint     `json:"currency_pair_id" binding:"required"`
	OrderType      string   `json:"order_type" binding:"required,oneof=LIMIT STOP"`
	Side           string   `json:"side" binding:"required,oneof=BUY SELL"`
	Quantity       float64  `json:"quantity" binding:"required,gt=0"`
	Price          float64  `json:"price" binding:"required"`
	StopLoss       float64  `json:"stop_loss"`
	TakeProfit     float64  `json:"take_profit"`
	ExpiresAt      *string  `json:"expires_at"` // RFC3339 format
}

// CreateOCOOrderInput represents input for creating an OCO order
type CreateOCOOrderInput struct {
	AccountID      uint    `json:"account_id" binding:"required"`
	CurrencyPairID uint    `json:"currency_pair_id" binding:"required"`
	Name           string  `json:"name"`
	// Buy order details
	BuyOrderType   string  `json:"buy_order_type" binding:"required,oneof=LIMIT STOP"`
	BuyQuantity    float64 `json:"buy_quantity" binding:"required,gt=0"`
	BuyPrice       float64 `json:"buy_price" binding:"required"`
	BuyStopLoss    float64 `json:"buy_stop_loss"`
	BuyTakeProfit  float64 `json:"buy_take_profit"`
	// Sell order details
	SellOrderType   string  `json:"sell_order_type" binding:"required,oneof=LIMIT STOP"`
	SellQuantity    float64 `json:"sell_quantity" binding:"required,gt=0"`
	SellPrice       float64 `json:"sell_price" binding:"required"`
	SellStopLoss    float64 `json:"sell_stop_loss"`
	SellTakeProfit  float64 `json:"sell_take_profit"`
}

// CreateOTOOrderInput represents input for creating an OTO order
type CreateOTOOrderInput struct {
	AccountID      uint                   `json:"account_id" binding:"required"`
	CurrencyPairID uint                  `json:"currency_pair_id" binding:"required"`
	Name           string                 `json:"name"`
	PrimaryOrder   CreatePendingOrderInput `json:"primary_order" binding:"required"`
	SecondaryOrders []CreatePendingOrderInput `json:"secondary_orders" binding:"required,min=1,max=4"`
}

// PendingOrderResponse represents a pending order response
type PendingOrderResponse struct {
	ID             uint      `json:"id"`
	AccountID      uint      `json:"account_id"`
	CurrencyPairID uint      `json:"currency_pair_id"`
	OrderType      string    `json:"order_type"`
	Side           string    `json:"side"`
	Quantity       float64   `json:"quantity"`
	Price          float64   `json:"price"`
	StopLoss       float64   `json:"stop_loss"`
	TakeProfit     float64   `json:"take_profit"`
	Status         string    `json:"status"`
	OCOGroupID     *uint     `json:"oco_group_id"`
	OTOGroupID     *uint     `json:"oto_group_id"`
	IsPrimary      bool      `json:"is_primary"`
	ExpiresAt      *string   `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// OCOOrderResponse represents an OCO order response
type OCOOrderResponse struct {
	ID             uint                   `json:"id"`
	AccountID      uint                   `json:"account_id"`
	CurrencyPairID uint                   `json:"currency_pair_id"`
	Name           string                 `json:"name"`
	Status         string                 `json:"status"`
	Orders         []PendingOrderResponse `json:"orders"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// OTOOrderResponse represents an OTO order response
type OTOOrderResponse struct {
	ID               uint                   `json:"id"`
	AccountID        uint                   `json:"account_id"`
	CurrencyPairID   uint                   `json:"currency_pair_id"`
	Name             string                 `json:"name"`
	Status           string                 `json:"status"`
	PrimaryOrder     *PendingOrderResponse  `json:"primary_order"`
	SecondaryOrders  []PendingOrderResponse `json:"secondary_orders"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// CreatePendingOrder creates a new pending order
func (s *AdvancedOrderService) CreatePendingOrder(userID uint, input CreatePendingOrderInput) (*PendingOrderResponse, error) {
	// Verify account ownership
	var account models.Account
	if err := s.db.First(&account, input.AccountID).Error; err != nil {
		return nil, errors.New("account not found")
	}
	if account.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	// Verify currency pair exists
	var pair models.CurrencyPair
	if err := s.db.First(&pair, input.CurrencyPairID).Error; err != nil {
		return nil, errors.New("currency pair not found")
	}

	// Validate price
	if input.Price <= 0 {
		return nil, errors.New("price must be greater than 0")
	}

	// Parse expiration if provided
	var expiresAt *time.Time
	if input.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *input.ExpiresAt)
		if err != nil {
			return nil, errors.New("invalid expires_at format, use RFC3339")
		}
		expiresAt = &t
	}

	// Calculate required margin for the order
	requiredMargin := (input.Price * input.Quantity) / account.Leverage
	if requiredMargin > account.MarginFree {
		return nil, errors.New("insufficient margin for pending order")
	}

	pendingOrder := models.PendingOrder{
		AccountID:      input.AccountID,
		CurrencyPairID: input.CurrencyPairID,
		OrderType:      input.OrderType,
		Side:           input.Side,
		Quantity:       input.Quantity,
		Price:          input.Price,
		StopLoss:       input.StopLoss,
		TakeProfit:     input.TakeProfit,
		Status:         "pending",
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.db.Create(&pendingOrder).Error; err != nil {
		return nil, err
	}

	// Reserve margin for the pending order
	account.MarginUsed += requiredMargin
	account.MarginFree = account.Balance - account.MarginUsed
	s.db.Save(&account)

	return s.pendingOrderToResponse(&pendingOrder), nil
}

// CreateOCOOrder creates a new OCO (One Cancels Other) order
func (s *AdvancedOrderService) CreateOCOOrder(userID uint, input CreateOCOOrderInput) (*OCOOrderResponse, error) {
	// Verify account ownership
	var account models.Account
	if err := s.db.First(&account, input.AccountID).Error; err != nil {
		return nil, errors.New("account not found")
	}
	if account.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	// Verify currency pair exists
	var pair models.CurrencyPair
	if err := s.db.First(&pair, input.CurrencyPairID).Error; err != nil {
		return nil, errors.New("currency pair not found")
	}

	// Create the OCO group
	ocoGroup := models.OCOOrder{
		AccountID:      input.AccountID,
		CurrencyPairID: input.CurrencyPairID,
		Name:           input.Name,
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.db.Create(&ocoGroup).Error; err != nil {
		return nil, err
	}

	// Create buy order
	buyOrder := models.PendingOrder{
		AccountID:      input.AccountID,
		CurrencyPairID: input.CurrencyPairID,
		OrderType:      input.BuyOrderType,
		Side:           "BUY",
		Quantity:       input.BuyQuantity,
		Price:          input.BuyPrice,
		StopLoss:       input.BuyStopLoss,
		TakeProfit:     input.BuyTakeProfit,
		Status:         "pending",
		OCOGroupID:     &ocoGroup.ID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create sell order
	sellOrder := models.PendingOrder{
		AccountID:      input.AccountID,
		CurrencyPairID: input.CurrencyPairID,
		OrderType:      input.SellOrderType,
		Side:           "SELL",
		Quantity:       input.SellQuantity,
		Price:          input.SellPrice,
		StopLoss:       input.SellStopLoss,
		TakeProfit:     input.SellTakeProfit,
		Status:         "pending",
		OCOGroupID:     &ocoGroup.ID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Calculate total required margin
	totalMargin := ((input.BuyPrice * input.BuyQuantity) + (input.SellPrice * input.SellQuantity)) / account.Leverage
	if totalMargin > account.MarginFree {
		return nil, errors.New("insufficient margin for OCO order")
	}

	// Save both orders
	if err := s.db.Create(&buyOrder).Error; err != nil {
		s.db.Delete(&ocoGroup)
		return nil, err
	}
	if err := s.db.Create(&sellOrder).Error; err != nil {
		s.db.Delete(&buyOrder)
		s.db.Delete(&ocoGroup)
		return nil, err
	}

	// Reserve margin
	account.MarginUsed += totalMargin
	account.MarginFree = account.Balance - account.MarginUsed
	s.db.Save(&account)

	// Fetch the complete OCO group with orders
	var orders []models.PendingOrder
	s.db.Where("oco_group_id = ?", ocoGroup.ID).Find(&orders)

	response := OCOOrderResponse{
		ID:             ocoGroup.ID,
		AccountID:      ocoGroup.AccountID,
		CurrencyPairID: ocoGroup.CurrencyPairID,
		Name:           ocoGroup.Name,
		Status:         ocoGroup.Status,
		CreatedAt:      ocoGroup.CreatedAt,
		UpdatedAt:      ocoGroup.UpdatedAt,
	}

	for _, order := range orders {
		response.Orders = append(response.Orders, *s.pendingOrderToResponse(&order))
	}

	return &response, nil
}

// CreateOTOOrder creates a new OTO (One Triggers Other) order
func (s *AdvancedOrderService) CreateOTOOrder(userID uint, input CreateOTOOrderInput) (*OTOOrderResponse, error) {
	// Verify account ownership
	var account models.Account
	if err := s.db.First(&account, input.AccountID).Error; err != nil {
		return nil, errors.New("account not found")
	}
	if account.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	// Verify currency pair exists
	var pair models.CurrencyPair
	if err := s.db.First(&pair, input.CurrencyPairID).Error; err != nil {
		return nil, errors.New("currency pair not found")
	}

	// Validate primary order price
	if input.PrimaryOrder.Price <= 0 {
		return nil, errors.New("primary order price must be greater than 0")
	}

	// Create the OTO group
	otoGroup := models.OTOOrder{
		AccountID:      input.AccountID,
		CurrencyPairID: input.CurrencyPairID,
		Name:           input.Name,
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.db.Create(&otoGroup).Error; err != nil {
		return nil, err
	}

	// Parse expiration for primary order if provided
	var primaryExpiresAt *time.Time
	if input.PrimaryOrder.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *input.PrimaryOrder.ExpiresAt)
		if err != nil {
			s.db.Delete(&otoGroup)
			return nil, errors.New("invalid primary order expires_at format, use RFC3339")
		}
		primaryExpiresAt = &t
	}

	// Create primary order
	primaryOrder := models.PendingOrder{
		AccountID:      input.AccountID,
		CurrencyPairID: input.CurrencyPairID,
		OrderType:      input.PrimaryOrder.OrderType,
		Side:           input.PrimaryOrder.Side,
		Quantity:       input.PrimaryOrder.Quantity,
		Price:          input.PrimaryOrder.Price,
		StopLoss:       input.PrimaryOrder.StopLoss,
		TakeProfit:     input.PrimaryOrder.TakeProfit,
		Status:         "pending",
		OTOGroupID:     &otoGroup.ID,
		IsPrimary:      true,
		ExpiresAt:      primaryExpiresAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.db.Create(&primaryOrder).Error; err != nil {
		s.db.Delete(&otoGroup)
		return nil, err
	}

	// Calculate required margin for primary order
	totalMargin := (input.PrimaryOrder.Price * input.PrimaryOrder.Quantity) / account.Leverage

	// Create secondary orders
	var secondaryOrders []models.PendingOrder
	for i, secInput := range input.SecondaryOrders {
		// Validate secondary order price
		if secInput.Price <= 0 {
			// Rollback
			s.db.Delete(&primaryOrder)
			s.db.Delete(&otoGroup)
			return nil, errors.New("secondary order price must be greater than 0")
		}

		// Calculate additional margin for secondary orders (they get activated when primary executes)
		totalMargin += (secInput.Price * secInput.Quantity) / account.Leverage

		secondaryOrder := models.PendingOrder{
			AccountID:      input.AccountID,
			CurrencyPairID: input.CurrencyPairID,
			OrderType:      secInput.OrderType,
			Side:           secInput.Side,
			Quantity:       secInput.Quantity,
			Price:          secInput.Price,
			StopLoss:       secInput.StopLoss,
			TakeProfit:     secInput.TakeProfit,
			Status:         "pending",
			OTOGroupID:     &otoGroup.ID,
			IsPrimary:      false,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		secondaryOrders = append(secondaryOrders, secondaryOrder)

		// Limit to 4 secondary orders
		if i >= 3 {
			break
		}
	}

	// Check margin for all orders
	if totalMargin > account.MarginFree {
		s.db.Delete(&primaryOrder)
		s.db.Delete(&otoGroup)
		return nil, errors.New("insufficient margin for OTO order")
	}

	// Save secondary orders
	for _, order := range secondaryOrders {
		if err := s.db.Create(&order).Error; err != nil {
			s.db.Delete(&primaryOrder)
			s.db.Delete(&otoGroup)
			return nil, err
		}
	}

	// Reserve margin
	account.MarginUsed += totalMargin
	account.MarginFree = account.Balance - account.MarginUsed
	s.db.Save(&account)

	// Build response
	response := OTOOrderResponse{
		ID:             otoGroup.ID,
		AccountID:      otoGroup.AccountID,
		CurrencyPairID: otoGroup.CurrencyPairID,
		Name:           otoGroup.Name,
		Status:         otoGroup.Status,
		CreatedAt:      otoGroup.CreatedAt,
		UpdatedAt:      otoGroup.UpdatedAt,
	}

	response.PrimaryOrder = s.pendingOrderToResponse(&primaryOrder)

	for _, order := range secondaryOrders {
		orderCopy := order
		response.SecondaryOrders = append(response.SecondaryOrders, *s.pendingOrderToResponse(&orderCopy))
	}

	return &response, nil
}

// GetPendingOrders returns all pending orders for an account
func (s *AdvancedOrderService) GetPendingOrders(accountID uint) ([]PendingOrderResponse, error) {
	var orders []models.PendingOrder
	if err := s.db.Where("account_id = ? AND status = 'pending'", accountID).Find(&orders).Error; err != nil {
		return nil, err
	}

	var responses []PendingOrderResponse
	for i := range orders {
		responses = append(responses, *s.pendingOrderToResponse(&orders[i]))
	}

	return responses, nil
}

// GetOCOOrders returns all OCO orders for an account
func (s *AdvancedOrderService) GetOCOOrders(accountID uint) ([]OCOOrderResponse, error) {
	var ocoOrders []models.OCOOrder
	if err := s.db.Where("account_id = ?", accountID).Find(&ocoOrders).Error; err != nil {
		return nil, err
	}

	var responses []OCOOrderResponse
	for _, oco := range ocoOrders {
		var orders []models.PendingOrder
		s.db.Where("oco_group_id = ?", oco.ID).Find(&orders)

		response := OCOOrderResponse{
			ID:             oco.ID,
			AccountID:      oco.AccountID,
			CurrencyPairID: oco.CurrencyPairID,
			Name:           oco.Name,
			Status:         oco.Status,
			CreatedAt:      oco.CreatedAt,
			UpdatedAt:      oco.UpdatedAt,
		}

		for _, order := range orders {
			response.Orders = append(response.Orders, *s.pendingOrderToResponse(&order))
		}

		responses = append(responses, response)
	}

	return responses, nil
}

// GetOTOOrders returns all OTO orders for an account
func (s *AdvancedOrderService) GetOTOOrders(accountID uint) ([]OTOOrderResponse, error) {
	var otoOrders []models.OTOOrder
	if err := s.db.Where("account_id = ?", accountID).Find(&otoOrders).Error; err != nil {
		return nil, err
	}

	var responses []OTOOrderResponse
	for _, oto := range otoOrders {
		var primaryOrder *models.PendingOrder
		var secondaryOrders []models.PendingOrder

		s.db.Where("oto_group_id = ? AND is_primary = true", oto.ID).First(&primaryOrder)
		s.db.Where("oto_group_id = ? AND is_primary = false", oto.ID).Find(&secondaryOrders)

		response := OTOOrderResponse{
			ID:             oto.ID,
			AccountID:      oto.AccountID,
			CurrencyPairID: oto.CurrencyPairID,
			Name:           oto.Name,
			Status:         oto.Status,
			CreatedAt:      oto.CreatedAt,
			UpdatedAt:      oto.UpdatedAt,
		}

		if primaryOrder != nil {
			response.PrimaryOrder = s.pendingOrderToResponse(primaryOrder)
		}

		for _, order := range secondaryOrders {
			orderCopy := order
			response.SecondaryOrders = append(response.SecondaryOrders, *s.pendingOrderToResponse(&orderCopy))
		}

		responses = append(responses, response)
	}

	return responses, nil
}

// CancelPendingOrder cancels a pending order
func (s *AdvancedOrderService) CancelPendingOrder(orderID uint, userID uint) error {
	var order models.PendingOrder
	if err := s.db.First(&order, orderID).Error; err != nil {
		return errors.New("order not found")
	}

	// Verify ownership
	var account models.Account
	if err := s.db.First(&account, order.AccountID).Error; err != nil {
		return errors.New("account not found")
	}
	if account.UserID != userID {
		return errors.New("unauthorized")
	}

	if order.Status != "pending" {
		return errors.New("only pending orders can be cancelled")
	}

	// Release margin
	marginToRelease := (order.Price * order.Quantity) / account.Leverage
	account.MarginUsed -= marginToRelease
	if account.MarginUsed < 0 {
		account.MarginUsed = 0
	}
	account.MarginFree = account.Balance - account.MarginUsed
	s.db.Save(&account)

	// Update order status
	order.Status = "cancelled"
	order.UpdatedAt = time.Now()
	s.db.Save(&order)

	// If part of OCO group, cancel the other order
	if order.OCOGroupID != nil {
		var otherOrders []models.PendingOrder
		s.db.Where("oco_group_id = ? AND id != ?", *order.OCOGroupID, order.ID).Find(&otherOrders)
		for _, other := range otherOrders {
			other.Status = "cancelled"
			other.UpdatedAt = time.Now()
			s.db.Save(&other)
		}

		// Update OCO group status
		var ocoGroup models.OCOOrder
		s.db.First(&ocoGroup, *order.OCOGroupID)
		ocoGroup.Status = "cancelled"
		ocoGroup.UpdatedAt = time.Now()
		s.db.Save(&ocoGroup)
	}

	return nil
}

// CancelOCOOrder cancels an entire OCO order group
func (s *AdvancedOrderService) CancelOCOOrder(ocoID uint, userID uint) error {
	var ocoGroup models.OCOOrder
	if err := s.db.First(&ocoGroup, ocoID).Error; err != nil {
		return errors.New("OCO order not found")
	}

	// Verify ownership
	var account models.Account
	if err := s.db.First(&account, ocoGroup.AccountID).Error; err != nil {
		return errors.New("account not found")
	}
	if account.UserID != userID {
		return errors.New("unauthorized")
	}

	// Get all orders in the group
	var orders []models.PendingOrder
	s.db.Where("oco_group_id = ?", ocoGroup.ID).Find(&orders)

	// Release margin and cancel orders
	var totalMargin float64
	for _, order := range orders {
		if order.Status == "pending" {
			totalMargin += (order.Price * order.Quantity) / account.Leverage
			order.Status = "cancelled"
			order.UpdatedAt = time.Now()
			s.db.Save(&order)
		}
	}

	// Release margin
	account.MarginUsed -= totalMargin
	if account.MarginUsed < 0 {
		account.MarginUsed = 0
	}
	account.MarginFree = account.Balance - account.MarginUsed
	s.db.Save(&account)

	// Update OCO group status
	ocoGroup.Status = "cancelled"
	ocoGroup.UpdatedAt = time.Now()
	s.db.Save(&ocoGroup)

	return nil
}

// CancelOTOOrder cancels an entire OTO order group
func (s *AdvancedOrderService) CancelOTOOrder(otoID uint, userID uint) error {
	var otoGroup models.OTOOrder
	if err := s.db.First(&otoGroup, otoID).Error; err != nil {
		return errors.New("OTO order not found")
	}

	// Verify ownership
	var account models.Account
	if err := s.db.First(&account, otoGroup.AccountID).Error; err != nil {
		return errors.New("account not found")
	}
	if account.UserID != userID {
		return errors.New("unauthorized")
	}

	// Get all orders in the group
	var orders []models.PendingOrder
	s.db.Where("oto_group_id = ?", otoGroup.ID).Find(&orders)

	// Release margin and cancel orders
	var totalMargin float64
	for _, order := range orders {
		if order.Status == "pending" {
			totalMargin += (order.Price * order.Quantity) / account.Leverage
			order.Status = "cancelled"
			order.UpdatedAt = time.Now()
			s.db.Save(&order)
		}
	}

	// Release margin
	account.MarginUsed -= totalMargin
	if account.MarginUsed < 0 {
		account.MarginUsed = 0
	}
	account.MarginFree = account.Balance - account.MarginUsed
	s.db.Save(&account)

	// Update OTO group status
	otoGroup.Status = "cancelled"
	otoGroup.UpdatedAt = time.Now()
	s.db.Save(&otoGroup)

	return nil
}

// CheckPendingOrders checks and triggers pending orders based on current price
// This should be called periodically (e.g., on each price update)
func (s *AdvancedOrderService) CheckPendingOrders(currentPrice float64) error {
	var pendingOrders []models.PendingOrder
	s.db.Where("status = 'pending'").Find(&pendingOrders)

	tradingService := NewTradingService(s.db)

	for _, order := range pendingOrders {
		// Check if order has expired
		if order.ExpiresAt != nil && time.Now().After(*order.ExpiresAt) {
			order.Status = "expired"
			order.UpdatedAt = time.Now()
			s.db.Save(&order)
			continue
		}

		// Check if order should be triggered
		shouldTrigger := false
		switch order.OrderType {
		case "LIMIT":
			// Limit buy: trigger when price <= order price
			// Limit sell: trigger when price >= order price
			if order.Side == "BUY" && currentPrice <= order.Price {
				shouldTrigger = true
			} else if order.Side == "SELL" && currentPrice >= order.Price {
				shouldTrigger = true
			}
		case "STOP":
			// Stop buy: trigger when price >= order price
			// Stop sell: trigger when price <= order price
			if order.Side == "BUY" && currentPrice >= order.Price {
				shouldTrigger = true
			} else if order.Side == "SELL" && currentPrice <= order.Price {
				shouldTrigger = true
			}
		}

		if shouldTrigger {
			// Execute the order
			input := ExecuteTradeInput{
				AccountID:      order.AccountID,
				CurrencyPairID: order.CurrencyPairID,
				Type:           order.Side,
				Quantity:       order.Quantity,
				EntryPrice:     currentPrice,
				StopLoss:       order.StopLoss,
				TakeProfit:     order.TakeProfit,
			}

			_, err := tradingService.ExecuteTrade(order.AccountID, input)
			if err != nil {
				order.Status = "failed"
				order.UpdatedAt = time.Now()
				s.db.Save(&order)
				continue
			}

			// Mark order as executed
			order.Status = "executed"
			order.UpdatedAt = time.Now()
			s.db.Save(&order)

			// If part of OCO group, cancel other orders
			if order.OCOGroupID != nil {
				var otherOrders []models.PendingOrder
				s.db.Where("oco_group_id = ? AND id != ? AND status = 'pending'", *order.OCOGroupID, order.ID).Find(&otherOrders)
				for _, other := range otherOrders {
					other.Status = "cancelled"
					other.UpdatedAt = time.Now()
					s.db.Save(&other)
				}

				// Update OCO group status
				var ocoGroup models.OCOOrder
				s.db.First(&ocoGroup, *order.OCOGroupID)
				ocoGroup.Status = "executed"
				ocoGroup.UpdatedAt = time.Now()
				s.db.Save(&ocoGroup)
			}

			// If part of OTO group and this was the primary order, activate secondary orders
			if order.OTOGroupID != nil && order.IsPrimary {
				var secondaryOrders []models.PendingOrder
				s.db.Where("oto_group_id = ? AND is_primary = false AND status = 'pending'", *order.OTOGroupID).Find(&secondaryOrders)
				for _, sec := range secondaryOrders {
					sec.Status = "active" // Secondary orders become active
					sec.UpdatedAt = time.Now()
					s.db.Save(&sec)
				}
			}
		}
	}

	return nil
}

// pendingOrderToResponse converts a PendingOrder to its response
func (s *AdvancedOrderService) pendingOrderToResponse(order *models.PendingOrder) *PendingOrderResponse {
	var expiresAt *string
	if order.ExpiresAt != nil {
		t := order.ExpiresAt.Format(time.RFC3339)
		expiresAt = &t
	}

	return &PendingOrderResponse{
		ID:             order.ID,
		AccountID:      order.AccountID,
		CurrencyPairID: order.CurrencyPairID,
		OrderType:      order.OrderType,
		Side:           order.Side,
		Quantity:       order.Quantity,
		Price:          order.Price,
		StopLoss:       order.StopLoss,
		TakeProfit:     order.TakeProfit,
		Status:         order.Status,
		OCOGroupID:     order.OCOGroupID,
		OTOGroupID:     order.OTOGroupID,
		IsPrimary:      order.IsPrimary,
		ExpiresAt:      expiresAt,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}
}
