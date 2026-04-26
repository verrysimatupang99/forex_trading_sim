package backtest

import (
	"time"

	"github.com/google/uuid"

	"forex-trading-sim/internal/services/strategies"
)

// TradeExecutor simulates trade execution with slippage and spread
type TradeExecutor struct {
	commission   float64   // Commission per trade (as decimal, e.g., 0.0002 = 2 pips)
	slippagePips float64   // Slippage in pips
	spreadPips   float64   // Spread in pips
}

// NewTradeExecutor creates a new trade executor
func NewTradeExecutor(commission, slippagePips, spreadPips float64) *TradeExecutor {
	return &TradeExecutor{
		commission:   commission,
		slippagePips: slippagePips,
		spreadPips:   spreadPips,
	}
}

// Execute executes a trade order and returns a backtest trade
func (e *TradeExecutor) Execute(order *strategies.Order, portfolio *Portfolio) *strategies.BacktestTrade {
	if order == nil {
		return nil
	}

	// Apply slippage to entry price
	entryPrice := e.applySlippage(order.EntryPrice, order.Side)

	// Apply spread (for market orders, we always get worse price)
	entryPrice = e.applySpread(entryPrice, order.Side)

	// Calculate commission
	commission := e.calculateCommission(entryPrice, order.Quantity)

	// Check if enough capital
	totalCost := entryPrice*order.Quantity + commission
	if order.Side == strategies.SignalBuy && totalCost > portfolio.Cash {
		// Not enough capital
		return nil
	}

	// Execute based on order side
	if order.Side == strategies.SignalBuy {
		portfolio.Cash -= totalCost
	} else {
		portfolio.Cash -= commission // Pay commission for sell
	}

	// Create open position
	position := OpenPosition{
		ID:           uuid.New().String(),
		Type:         order.Side,
		EntryPrice:   entryPrice,
		Quantity:     order.Quantity,
		StopLoss:     order.StopLoss,
		TakeProfit:   order.TakeProfit,
		OpenedAt:     order.Timestamp,
		CurrencyPair: order.CurrencyPair,
	}

	portfolio.OpenPositions = append(portfolio.OpenPositions, position)

	// Return the trade (will be updated when closed)
	trade := &strategies.BacktestTrade{
		ID:             position.ID,
		Type:           order.Side,
		EntryPrice:     entryPrice,
		Quantity:       order.Quantity,
		Commission:     commission,
		EntryTime:      order.Timestamp,
		StopLoss:       order.StopLoss,
		TakeProfit:     order.TakeProfit,
		Strategy:       "",
		CurrencyPair:   order.CurrencyPair,
	}

	return trade
}

// applySlippage adjusts the price for slippage
func (e *TradeExecutor) applySlippage(price float64, side strategies.SignalType) float64 {
	slippage := e.slippagePips * 0.0001 // Convert pips to price

	if side == strategies.SignalBuy {
		// Buy orders get filled at higher price
		return price + slippage
	}
	// Sell orders get filled at lower price
	return price - slippage
}

// applySpread applies the spread to the price
func (e *TradeExecutor) applySpread(price float64, side strategies.SignalType) float64 {
	spread := e.spreadPips * 0.0001 // Convert pips to price

	if side == strategies.SignalBuy {
		// Buy orders: ask price = bid + spread
		return price + spread
	}
	// Sell orders: bid price = ask - spread
	return price - spread
}

// calculateCommission calculates the commission for a trade
func (e *TradeExecutor) calculateCommission(price, quantity float64) float64 {
	return price * quantity * e.commission
}

// ClosePosition closes an open position and returns the realized trade
func (e *TradeExecutor) ClosePosition(position *OpenPosition, exitPrice float64, timestamp time.Time) *strategies.BacktestTrade {
	// Apply slippage to exit price
	exitPrice = e.applySlippage(exitPrice, position.Type)
	exitPrice = e.applySpread(exitPrice, position.Type)

	// Calculate P&L
	var pnl float64
	if position.Type == strategies.SignalBuy {
		pnl = (exitPrice - position.EntryPrice) * position.Quantity
	} else {
		pnl = (position.EntryPrice - exitPrice) * position.Quantity
	}

	// Calculate commission
	commission := e.calculateCommission(exitPrice, position.Quantity)
	pnl -= commission

	// Calculate P&L percentage
	pnlPercent := 0.0
	if position.EntryPrice*position.Quantity > 0 {
		pnlPercent = (pnl / (position.EntryPrice * position.Quantity)) * 100
	}

	trade := &strategies.BacktestTrade{
		ID:           position.ID,
		Type:         position.Type,
		EntryPrice:   position.EntryPrice,
		ExitPrice:    exitPrice,
		Quantity:     position.Quantity,
		PnL:          pnl,
		PnLPercent:   pnlPercent,
		Commission:   commission,
		EntryTime:    position.OpenedAt,
		ExitTime:     timestamp,
		StopLoss:     position.StopLoss,
		TakeProfit:   position.TakeProfit,
		CurrencyPair: position.CurrencyPair,
	}

	return trade
}

// GetConfig returns the executor configuration
func (e *TradeExecutor) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"commission":   e.commission,
		"slippagePips": e.slippagePips,
		"spreadPips":   e.spreadPips,
	}
}
