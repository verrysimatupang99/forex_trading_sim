package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"forex-trading-sim/config"
	"forex-trading-sim/internal/database"
	"forex-trading-sim/internal/handlers"
	"forex-trading-sim/internal/middleware"
	"forex-trading-sim/internal/services"
)

// @title Forex Trading Simulator API
// @version 1.1
// @description API for forex trading simulator with ML predictions
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api
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

	log.Println("Initializing services...")

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
	predictionHandler := handlers.NewPredictionHandler(predictionService, db)
	backtestHandler := handlers.NewBacktestHandler(db)
	advancedOrdersHandler := handlers.NewAdvancedOrdersHandler(advancedOrderService)
	currencyHandler := handlers.NewCurrencyHandler(currencyConverter, db)

	// Setup router with custom middleware
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	// Readiness check
	r.GET("/ready", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(503, gin.H{"status": "not ready"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})

	// Serve static web files
	r.Static("/web", "./web")

	// Serve root index.html
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	// DB middleware to make DB available in handlers
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// API versioning middleware
	r.Use(middleware.VersionHandler())

	// Public routes with rate limiting
	api := r.Group("/api")
	{
		// Apply stricter rate limiting to auth endpoints
		auth := api.Group("/v1/auth")
		auth.Use(middleware.AuthRateLimitMiddleware())
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Public data endpoints with standard rate limiting
		v1 := api.Group("/v1")
		v1.Use(middleware.RateLimitMiddleware())
		v1.Use(middleware.InputSanitizer())
		{
			v1.GET("/historical-data", handlers.GetHistoricalData)
			v1.GET("/technical-indicators", handlers.GetTechnicalIndicators)
			v1.GET("/currency-pairs", handlers.GetCurrencyPairs)
		}
	}

	// Protected routes
	protected := api.Group("/v1")
	protected.Use(middleware.JWTAuth())
	protected.Use(middleware.RateLimitMiddleware())
	protected.Use(middleware.InputSanitizer())
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

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("=================================================")
	log.Printf("Forex Trading Simulator API v1.1 starting...")
	log.Printf("Server port: %s", port)
	log.Printf("API Documentation: http://localhost:%s/swagger/index.html", port)
	log.Printf("Supported API Versions: v1.0, v1.1")
	log.Printf("Current Version: v1.1")
	log.Printf("=================================================")
	
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// GetDBFromContext retrieves the database from gin context
func GetDBFromContext(c *gin.Context) *gorm.DB {
	db, exists := c.Get("db")
	if !exists || db == nil {
		return nil
	}
	return db.(*gorm.DB)
}
