package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"forex-trading-sim/config"
	"forex-trading-sim/internal/database"
	"forex-trading-sim/internal/handlers"
	"forex-trading-sim/internal/middleware"
	"forex-trading-sim/internal/services"
)

// @title Forex Trading Simulator API
// @version 1.0
// @description API for forex trading simulator with ML predictions
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
func main() {
	// Load configuration with validation
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize services
	authService := services.NewAuthService(db)
	userService := services.NewUserService(db)
	tradingService := services.NewTradingService(db)
	predictionService := services.NewPredictionService(db)
	advancedOrderService := services.NewAdvancedOrderService(db)
	currencyConverter := services.NewCurrencyConverter(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	tradingHandler := handlers.NewTradingHandler(tradingService)
	predictionHandler := handlers.NewPredictionHandler(predictionService)
	backtestHandler := handlers.NewBacktestHandler(db)
	advancedOrdersHandler := handlers.NewAdvancedOrdersHandler(advancedOrderService)
	currencyHandler := handlers.NewCurrencyHandler(currencyConverter, db)

	// Setup router
	r := gin.Default()

	// Public routes
	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Public data endpoints
		api.GET("/historical-data", handlers.GetHistoricalData)
		api.GET("/technical-indicators", handlers.GetTechnicalIndicators)
		api.GET("/currency-pairs", handlers.GetCurrencyPairs)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.JWTAuth())
	{
		// User management
		users := protected.Group("/users")
		{
			users.GET("/me", userHandler.GetProfile)
			users.PUT("/me", userHandler.UpdateProfile)
		}

		// Trading
		trading := protected.Group("/trading")
		{
			trading.GET("/accounts", tradingHandler.GetAccounts)
			trading.POST("/accounts", tradingHandler.CreateAccount)
			trading.GET("/accounts/:id/balance", tradingHandler.GetBalance)
			trading.POST("/trade", tradingHandler.ExecuteTrade)
			trading.GET("/positions", tradingHandler.GetPositions)
			trading.GET("/trades", tradingHandler.GetTradeHistory)
			trading.DELETE("/positions/:id", tradingHandler.ClosePosition)
		}

		// Predictions
		predictions := protected.Group("/predictions")
		{
			predictions.POST("/predict", predictionHandler.Predict)
			predictions.GET("/history", predictionHandler.GetPredictionHistory)
		}

		// Backtesting
		backtest := protected.Group("/backtest")
		{
			backtest.POST("/run", backtestHandler.RunBacktest)
			backtest.GET("/results", backtestHandler.GetBacktestResults)
			backtest.GET("/results/:id", backtestHandler.GetBacktestResult)
			backtest.GET("/equity-curve/:id", backtestHandler.GetEquityCurve)
			backtest.GET("/trades/:id", backtestHandler.GetBacktestTrades)
		}

		// Walk-Forward Analysis
		walkforward := protected.Group("/walkforward")
		{
			walkforward.POST("/run", backtestHandler.RunWalkForward)
			walkforward.GET("/results", backtestHandler.GetWalkForwardResults)
		}

		// Advanced Orders (OCO, OTO, Pending Orders)
		orders := protected.Group("/orders")
		{
			// Pending Orders
			orders.POST("/pending", advancedOrdersHandler.CreatePendingOrder)
			orders.GET("/pending", advancedOrdersHandler.GetPendingOrders)
			orders.DELETE("/pending/:id", advancedOrdersHandler.CancelPendingOrder)

			// OCO Orders
			orders.POST("/oco", advancedOrdersHandler.CreateOCOOrder)
			orders.GET("/oco", advancedOrdersHandler.GetOCOOrders)
			orders.DELETE("/oco/:id", advancedOrdersHandler.CancelOCOOrder)

			// OTO Orders
			orders.POST("/oto", advancedOrdersHandler.CreateOTOOrder)
			orders.GET("/oto", advancedOrdersHandler.GetOTOOrders)
			orders.DELETE("/oto/:id", advancedOrdersHandler.CancelOTOOrder)
		}

		// Currency Conversion & Multi-Currency
		currency := protected.Group("/currency")
		{
			// Exchange Rates
			currency.GET("/rate", currencyHandler.GetExchangeRate)
			currency.GET("/rates", currencyHandler.GetRates)
			currency.POST("/rate", currencyHandler.UpdateRate)
			currency.GET("/cross", currencyHandler.GetCrossRate)
			currency.POST("/refresh", currencyHandler.RefreshRates)

			// Currency Conversion
			currency.POST("/convert", currencyHandler.ConvertCurrency)

			// Multi-Currency Accounts
			currency.POST("/account", currencyHandler.CreateMultiCurrencyAccount)
			currency.GET("/account/:id", currencyHandler.GetMultiCurrencyAccount)
			currency.GET("/exposure/:account_id", currencyHandler.GetCurrencyExposure)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
