package services

import (
	"testing"
)

// TestCalculatePnL tests the P&L calculation for BUY trades
func TestCalculatePnL_Buy(t *testing.T) {
	tests := []struct {
		name       string
		entryPrice float64
		exitPrice  float64
		quantity   float64
		wantPnL    float64
	}{
		{
			name:       "Profitable BUY trade",
			entryPrice: 1.0850,
			exitPrice:  1.0900,
			quantity:   10000,
			wantPnL:    50.0, // (1.0900 - 1.0850) * 10000 = 50
		},
		{
			name:       "Losing BUY trade",
			entryPrice: 1.0900,
			exitPrice:  1.0850,
			quantity:   10000,
			wantPnL:    -50.0,
		},
		{
			name:       "Break-even BUY trade",
			entryPrice: 1.0850,
			exitPrice:  1.0850,
			quantity:   10000,
			wantPnL:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePnL("BUY", tt.entryPrice, tt.exitPrice, tt.quantity)
			if got != tt.wantPnL {
				t.Errorf("CalculatePnL() = %v, want %v", got, tt.wantPnL)
			}
		})
	}
}

// TestCalculatePnL_Sell tests the P&L calculation for SELL trades
func TestCalculatePnL_Sell(t *testing.T) {
	tests := []struct {
		name       string
		entryPrice float64
		exitPrice  float64
		quantity   float64
		wantPnL    float64
	}{
		{
			name:       "Profitable SELL trade",
			entryPrice: 1.0900,
			exitPrice:  1.0850,
			quantity:   10000,
			wantPnL:    50.0, // (1.0900 - 1.0850) * 10000 = 50
		},
		{
			name:       "Losing SELL trade",
			entryPrice: 1.0850,
			exitPrice:  1.0900,
			quantity:   10000,
			wantPnL:    -50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePnL("SELL", tt.entryPrice, tt.exitPrice, tt.quantity)
			if got != tt.wantPnL {
				t.Errorf("CalculatePnL() = %v, want %v", got, tt.wantPnL)
			}
		})
	}
}

// TestCalculateMargin tests margin calculation
func TestCalculateMargin(t *testing.T) {
	tests := []struct {
		name      string
		price     float64
		quantity  float64
		leverage  float64
		wantMargin float64
	}{
		{
			name:       "100:1 leverage",
			price:      1.0850,
			quantity:   10000,
			leverage:   100,
			wantMargin: 108.5, // (1.0850 * 10000) / 100 = 108.5
		},
		{
			name:       "50:1 leverage",
			price:      1.0850,
			quantity:   10000,
			leverage:   50,
			wantMargin: 217.0,
		},
		{
			name:       "10:1 leverage",
			price:      1.0850,
			quantity:   10000,
			leverage:   10,
			wantMargin: 1085.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateMargin(tt.price, tt.quantity, tt.leverage)
			if got != tt.wantMargin {
				t.Errorf("CalculateMargin() = %v, want %v", got, tt.wantMargin)
			}
		})
	}
}

// TestCalculateCommission tests commission calculation
func TestCalculateCommission(t *testing.T) {
	tests := []struct {
		name      string
		price     float64
		quantity  float64
		rate      float64
		wantComm  float64
	}{
		{
			name:      "Standard commission (2 pips)",
			price:     1.0850,
			quantity:  10000,
			rate:      0.0002,
			wantComm:  2.17, // 1.0850 * 10000 * 0.0002
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCommission(tt.price, tt.quantity, tt.rate)
			if got != tt.wantComm {
				t.Errorf("CalculateCommission() = %v, want %v", got, tt.wantComm)
			}
		})
	}
}

// TestCalculatePnLPercent tests P&L percentage calculation
func TestCalculatePnLPercent(t *testing.T) {
	tests := []struct {
		name      string
		pnl       float64
		entryPrice float64
		quantity  float64
		wantPercent float64
	}{
		{
			name:        "5% profit",
			pnl:         50.0,
			entryPrice:  1.0850,
			quantity:    10000,
			wantPercent: 0.4608, // 50 / (1.0850 * 10000) * 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePnLPercent(tt.pnl, tt.entryPrice, tt.quantity)
			if got != tt.wantPercent {
				t.Errorf("CalculatePnLPercent() = %v, want %v", got, tt.wantPercent)
			}
		})
	}
}
