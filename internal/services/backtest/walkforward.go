package backtest

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
)

// WalkForwardConfig represents the configuration for walk-forward analysis
type WalkForwardConfig struct {
	StrategyName       string                 `json:"strategy_name"`
	Parameters         map[string]interface{} `json:"parameters"`
	CurrencyPairID     uint                   `json:"currency_pair_id"`
	Timeframe          string                 `json:"timeframe"`
	StartDate          time.Time              `json:"start_date"`
	EndDate            time.Time              `json:"end_date"`
	InitialCapital     float64                `json:"initial_capital"`
	TrainingPeriodDays int                    `json:"training_period_days" binding:"min=30"`
	TestingPeriodDays  int                    `json:"testing_period_days" binding:"min=7"`
	StepForwardDays    int                    `json:"step_forward_days" binding:"min=1"`
	Commission         float64                `json:"commission"`
	SlippagePips       float64                `json:"slippage_pips"`
	SpreadPips         float64                `json:"spread_pips"`
	PositionSizingMode string                 `json:"position_sizing_mode"` // "FIXED_LOT" or "PERCENT_EQUITY"
	FixedLot           float64                `json:"fixed_lot"`
	EquityPercent      float64                `json:"equity_percent"`
}

// WalkForwardFold represents a single fold in walk-forward analysis
type WalkForwardFold struct {
	Fold         int       `json:"fold"`
	TrainStart   time.Time `json:"train_start"`
	TrainEnd     time.Time `json:"train_end"`
	TestStart    time.Time `json:"test_start"`
	TestEnd      time.Time `json:"test_end"`
	TrainMetrics PerformanceMetrics `json:"train_metrics"`
	TestMetrics  PerformanceMetrics `json:"test_metrics"`
}

// WalkForwardResult represents the complete walk-forward analysis result
type WalkForwardResult struct {
	ID                  uint              `json:"id,omitempty"`
	StrategyName        string            `json:"strategy_name"`
	CurrencyPairID      uint              `json:"currency_pair_id"`
	Timeframe           string            `json:"timeframe"`
	StartDate           time.Time         `json:"start_date"`
	EndDate             time.Time         `json:"end_date"`
	TrainingPeriodDays  int               `json:"training_period_days"`
	TestingPeriodDays   int               `json:"testing_period_days"`
	StepForwardDays     int               `json:"step_forward_days"`
	NumFolds            int               `json:"num_folds"`
	Folds               []WalkForwardFold `json:"folds"`
	
	// Aggregate metrics
	AvgOutSampleReturn  float64           `json:"avg_out_sample_return"`
	AvgOutSampleSharpe  float64           `json:"avg_out_sample_sharpe"`
	AvgOutSampleDD      float64           `json:"avg_out_sample_max_drawdown"`
	CoefficientOfVariation float64        `json:"coefficient_of_variation"` // Stability measure
	StartTime           time.Time         `json:"start_time"`
	EndTime             time.Time         `json:"end_time"`
}

// WalkForwardService handles walk-forward analysis
type WalkForwardService struct {
	db *gorm.DB
}

// NewWalkForwardService creates a new walk-forward service
func NewWalkForwardService(db *gorm.DB) *WalkForwardService {
	return &WalkForwardService{db: db}
}

// Run executes walk-forward analysis
func (s *WalkForwardService) Run(config WalkForwardConfig) (*WalkForwardResult, error) {
	// Validate configuration
	if err := s.validateConfig(config); err != nil {
		return nil, err
	}

	// Generate folds
	folds := s.generateFolds(config)
	if len(folds) == 0 {
		return nil, errors.New("no folds generated - check date range and period settings")
	}

	result := &WalkForwardResult{
		StrategyName:       config.StrategyName,
		CurrencyPairID:     config.CurrencyPairID,
		Timeframe:          config.Timeframe,
		StartDate:          config.StartDate,
		EndDate:            config.EndDate,
		TrainingPeriodDays: config.TrainingPeriodDays,
		TestingPeriodDays:  config.TestingPeriodDays,
		StepForwardDays:    config.StepForwardDays,
		NumFolds:           len(folds),
		Folds:              make([]WalkForwardFold, 0, len(folds)),
		StartTime:          time.Now(),
	}

	// Run folds in parallel for better performance
	var wg sync.WaitGroup
	resultsChan := make(chan WalkForwardFold, len(folds))
	
	for i, fold := range folds {
		wg.Add(1)
		go func(foldIdx int, foldData WalkForwardFold) {
			defer wg.Done()
			
			// Run training period
			trainResult, err := s.runFold(foldData, config, true)
			if err != nil {
				// Log error but continue
				return
			}
			foldData.TrainMetrics = trainResult
			
			// Run testing period (out-of-sample)
			testResult, err := s.runFold(foldData, config, false)
			if err != nil {
				// Log error but continue
				return
			}
			foldData.TestMetrics = testResult
			
			resultsChan <- foldData
		}(i, fold)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for foldResult := range resultsChan {
		result.Folds = append(result.Folds, foldResult)
	}

	// Calculate aggregate metrics
	result.calculateAggregateMetrics()
	result.EndTime = time.Now()

	return result, nil
}

// validateConfig validates the walk-forward configuration
func (s *WalkForwardService) validateConfig(config WalkForwardConfig) error {
	if config.EndDate.Before(config.StartDate) {
		return errors.New("end_date must be after start_date")
	}

	if config.TrainingPeriodDays < 30 {
		return errors.New("training_period_days must be at least 30")
	}

	if config.TestingPeriodDays < 7 {
		return errors.New("testing_period_days must be at least 7")
	}

	if config.StepForwardDays < 1 {
		return errors.New("step_forward_days must be at least 1")
	}

	totalPeriod := config.TrainingPeriodDays + config.TestingPeriodDays
	dateRange := int(config.EndDate.Sub(config.StartDate).Hours() / 24)
	
	if dateRange < totalPeriod {
		return errors.New("date range too short for training and testing periods")
	}

	return nil
}

// generateFolds generates the walk-forward folds
func (s *WalkForwardService) generateFolds(config WalkForwardConfig) []WalkForwardFold {
	var folds []WalkForwardFold
	
	trainingDuration := time.Duration(config.TrainingPeriodDays) * 24 * time.Hour
	testingDuration := time.Duration(config.TestingPeriodDays) * 24 * time.Hour
	stepDuration := time.Duration(config.StepForwardDays) * 24 * time.Hour
	
	currentStart := config.StartDate
	foldIdx := 0
	
	for {
		trainEnd := currentStart.Add(trainingDuration)
		testStart := trainEnd
		testEnd := testStart.Add(testingDuration)
		
		// Check if we have enough data
		if testEnd.After(config.EndDate) {
			break
		}
		
		fold := WalkForwardFold{
			Fold:       foldIdx,
			TrainStart: currentStart,
			TrainEnd:   trainEnd,
			TestStart:  testStart,
			TestEnd:    testEnd,
		}
		
		folds = append(folds, fold)
		foldIdx++
		
		// Move start date forward by step
		currentStart = currentStart.Add(stepDuration)
		
		// Safety limit to prevent infinite loops
		if foldIdx > 100 {
			break
		}
	}
	
	return folds
}

// runFold runs a single fold (either training or testing period)
func (s *WalkForwardService) runFold(fold WalkForwardFold, config WalkForwardConfig, isTraining bool) (PerformanceMetrics, error) {
	var startDate, endDate time.Time
	
	if isTraining {
		startDate = fold.TrainStart
		endDate = fold.TrainEnd
	} else {
		startDate = fold.TestStart
		endDate = fold.TestEnd
	}

	// Create strategy - strategy is created inside engine via SetStrategy

	// Create backtest config
	backtestConfig := BacktestConfig{
		StrategyName:   config.StrategyName,
		Parameters:     config.Parameters,
		CurrencyPairID: config.CurrencyPairID,
		Timeframe:      config.Timeframe,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: config.InitialCapital,
		Commission:     config.Commission,
		SlippagePips:   config.SlippagePips,
		SpreadPips:     config.SpreadPips,
		PositionSizing: PositionSizingConfig{
			Mode:          config.PositionSizingMode,
			FixedLot:      config.FixedLot,
			EquityPercent: config.EquityPercent,
		},
	}

	// Create and run engine
	engine, err := NewBacktestEngine(s.db, backtestConfig)
	if err != nil {
		return PerformanceMetrics{}, err
	}
	
	err = engine.SetStrategy(config.StrategyName, config.Parameters)
	if err != nil {
		return PerformanceMetrics{}, err
	}
	
	result, err := engine.Run()
	if err != nil {
		return PerformanceMetrics{}, err
	}

	return result.Metrics, nil
}

// calculateAggregateMetrics calculates aggregate statistics across all folds
func (r *WalkForwardResult) calculateAggregateMetrics() {
	if len(r.Folds) == 0 {
		return
	}

	// Calculate average out-of-sample (test) metrics
	var totalReturn, totalSharpe, totalDD float64
	returns := make([]float64, 0, len(r.Folds))
	
	for _, fold := range r.Folds {
		totalReturn += fold.TestMetrics.TotalReturn
		totalSharpe += fold.TestMetrics.SharpeRatio
		totalDD += fold.TestMetrics.MaxDrawdown
		returns = append(returns, fold.TestMetrics.TotalReturn)
	}
	
	n := float64(len(r.Folds))
	r.AvgOutSampleReturn = totalReturn / n
	r.AvgOutSampleSharpe = totalSharpe / n
	r.AvgOutSampleDD = totalDD / n
	
	// Calculate Coefficient of Variation (CV) for stability measure
	if len(returns) > 1 {
		mean := r.AvgOutSampleReturn
		var sumSquaredDiff float64
		for _, ret := range returns {
			diff := ret - mean
			sumSquaredDiff += diff * diff
		}
		stdDev := sqrt(sumSquaredDiff / (n - 1))
		
		if mean != 0 {
			r.CoefficientOfVariation = (stdDev / mean) * 100 // As percentage
		}
	}
}

// Save saves the walk-forward result to database
func (r *WalkForwardResult) Save(db *gorm.DB) error {
	// Serialize folds to JSON
	foldsJSON, err := json.Marshal(r.Folds)
	if err != nil {
		return err
	}

	record := models.WalkForwardAnalysis{
		StrategyName:       r.StrategyName,
		CurrencyPairID:     r.CurrencyPairID,
		Timeframe:          r.Timeframe,
		StartDate:          r.StartDate,
		EndDate:            r.EndDate,
		TrainingPeriodDays: r.TrainingPeriodDays,
		TestingPeriodDays:  r.TestingPeriodDays,
		StepForwardDays:    r.StepForwardDays,
		NumFolds:           r.NumFolds,
		Folds:              string(foldsJSON),
		AvgOutSampleReturn: r.AvgOutSampleReturn,
		AvgOutSampleSharpe: r.AvgOutSampleSharpe,
		AvgOutSampleDD:     r.AvgOutSampleDD,
		CoefficientOfVariation: r.CoefficientOfVariation,
		StartTime:          r.StartTime,
		EndTime:            r.EndTime,
	}

	return db.Create(&record).Error
}

// sqrt calculates square root
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
