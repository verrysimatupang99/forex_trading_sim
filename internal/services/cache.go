package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheService handles Redis caching
type CacheService struct {
	client *redis.Client
}

// NewCacheService creates a new cache service
func NewCacheService(host, port string) (*CacheService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CacheService{client: client}, nil
}

// Get retrieves a value from cache
func (s *CacheService) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil // Key not found
	}
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// Set stores a value in cache with expiration
func (s *CacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, data, expiration).Err()
}

// Delete removes a key from cache
func (s *CacheService) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (s *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	count, err := s.client.Exists(ctx, key).Result()
	return count > 0, err
}

// GetOrFetch retrieves from cache or fetch using the provided function
func (s *CacheService) GetOrFetch(ctx context.Context, key string, dest interface{}, expiration time.Duration, fetchFn func() (interface{}, error)) error {
	// Try to get from cache first
	err := s.Get(ctx, key, dest)
	if err == nil {
		return nil // Found in cache
	}

	// Fetch fresh data
	data, err := fetchFn()
	if err != nil {
		return err
	}

	// Store in cache
	if err := s.Set(ctx, key, data, expiration); err != nil {
		// Log but don't fail
		fmt.Printf("Failed to cache data: %v\n", err)
	}

	// Unmarshal to dest
	if dataBytes, ok := data.([]byte); ok {
		return json.Unmarshal(dataBytes, dest)
	}
	
	return json.Unmarshal([]byte(fmt.Sprintf("%v", data)), dest)
}

// PriceCacheKeys generates cache keys for price data
func PriceCacheKeys(pairID uint, timeframe string) map[string]string {
	return map[string]string{
		"latest":    fmt.Sprintf("price:%d:%s:latest", pairID, timeframe),
		"historical": fmt.Sprintf("price:%d:%s:history", pairID, timeframe),
		"bid":       fmt.Sprintf("price:%d:%s:bid", pairID, timeframe),
		"ask":       fmt.Sprintf("price:%d:%s:ask", pairID, timeframe),
	}
}

// CachePrice caches the latest price
func (s *CacheService) CachePrice(ctx context.Context, pairID uint, price float64, timeframe string) error {
	keys := PriceCacheKeys(pairID, timeframe)
	return s.Set(ctx, keys["latest"], price, 30*time.Second)
}

// GetCachedPrice gets cached price
func (s *CacheService) GetCachedPrice(ctx context.Context, pairID uint, timeframe string) (float64, error) {
	keys := PriceCacheKeys(pairID, timeframe)
	var price float64
	err := s.Get(ctx, keys["latest"], &price)
	return price, err
}

// InvalidatePriceCache removes price cache for a pair
func (s *CacheService) InvalidatePriceCache(ctx context.Context, pairID uint, timeframe string) error {
	keys := PriceCacheKeys(pairID, timeframe)
	for _, key := range keys {
		if err := s.client.Del(ctx, key).Err(); err != nil {
			return err
		}
	}
	return nil
}

// CachePredictions caches ML predictions
func (s *CacheService) CachePredictions(ctx context.Context, pairID uint, predictions interface{}) error {
	key := fmt.Sprintf("predictions:%d", pairID)
	return s.Set(ctx, key, predictions, 5*time.Minute)
}

// GetCachedPredictions gets cached predictions
func (s *CacheService) GetCachedPredictions(ctx context.Context, pairID uint, dest interface{}) error {
	key := fmt.Sprintf("predictions:%d", pairID)
	return s.Get(ctx, key, dest)
}

// Close closes the Redis connection
func (s *CacheService) Close() error {
	return s.client.Close()
}
