package helper

import (
	"context"
	"fmt"

	"ecommerce-be/common/log"
)

// ============================================================================
// Batch Processing Helpers (Concurrent batch fetching with goroutines)
// ============================================================================

// BatchResult holds the result from a single batch fetch operation
// K is the key type (e.g., uint for IDs)
// V is the value type (e.g., string for names, *Entity for objects)
type BatchResult[K comparable, V any] struct {
	Data map[K]V
	Err  error
}

// BatchFetchFunc is the function signature for fetching a single batch
// Takes a slice of IDs and returns a map of ID -> Value
type BatchFetchFunc[K comparable, V any] func(batchIDs []K) (map[K]V, error)

// BatchFetch fetches items in batches concurrently using goroutines
// - ctx: context for logging and cancellation
// - ids: slice of IDs to fetch
// - batchSize: number of items per batch (e.g., 100)
// - fetchFunc: function to fetch a single batch
//
// Features:
// - Skips concurrency for small sets (≤ batchSize)
// - Uses buffered channels for non-blocking sends
// - Includes panic recovery to prevent blocking
// - Continues on individual batch errors (graceful degradation)
// - Logs errors with context for debugging
//
// Example usage:
//
//	userMap, err := helper.BatchFetch(ctx, userIDs, 100, func(batchIDs []uint) (map[uint]string, error) {
//	    // Fetch users from DB or API
//	    return fetchUsersFromDB(batchIDs)
//	})
func BatchFetch[K comparable, V any](
	ctx context.Context,
	ids []K,
	batchSize int,
	fetchFunc BatchFetchFunc[K, V],
) (map[K]V, error) {
	result := make(map[K]V)

	if len(ids) == 0 {
		return result, nil
	}

	// For small sets, fetch directly without goroutine overhead
	if len(ids) <= batchSize {
		return fetchFunc(ids)
	}

	// Calculate total batches
	totalBatches := (len(ids) + batchSize - 1) / batchSize
	resultsChan := make(chan BatchResult[K, V], totalBatches)

	// Launch goroutines for each batch
	launchBatchFetchers(ids, batchSize, totalBatches, fetchFunc, resultsChan)

	// Collect results from all batches
	return collectBatchResults(ctx, resultsChan, totalBatches)
}

// launchBatchFetchers launches goroutines to fetch batches concurrently
func launchBatchFetchers[K comparable, V any](
	ids []K,
	batchSize int,
	totalBatches int,
	fetchFunc BatchFetchFunc[K, V],
	resultsChan chan<- BatchResult[K, V],
) {
	for i := 0; i < totalBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		batchIDs := ids[start:end]
		batchNum := i + 1
		go fetchBatch(batchIDs, batchNum, fetchFunc, resultsChan)
	}
}

// fetchBatch fetches a single batch and sends result to channel
// Includes panic recovery to prevent blocking the collector
func fetchBatch[K comparable, V any](
	batchIDs []K,
	batchNum int,
	fetchFunc BatchFetchFunc[K, V],
	resultsChan chan<- BatchResult[K, V],
) {
	// Recover from panic to ensure we always send a result
	// This prevents collectBatchResults from blocking forever
	defer recoverBatchPanic(batchNum, resultsChan)()

	data, err := fetchFunc(batchIDs)
	if err != nil {
		resultsChan <- BatchResult[K, V]{
			Err: fmt.Errorf("batch %d failed: %w", batchNum, err),
		}
		return
	}

	resultsChan <- BatchResult[K, V]{Data: data, Err: nil}
}

// collectBatchResults collects results from all batch goroutines
func collectBatchResults[K comparable, V any](
	ctx context.Context,
	resultsChan <-chan BatchResult[K, V],
	totalBatches int,
) (map[K]V, error) {
	allResults := make(map[K]V)

	for i := 0; i < totalBatches; i++ {
		result := <-resultsChan
		if result.Err != nil {
			// Log error but continue - graceful degradation
			log.ErrorWithContext(ctx, "Batch fetch failed", result.Err)
			continue
		}

		for k, v := range result.Data {
			allResults[k] = v
		}
	}

	return allResults, nil
}

// recoverBatchPanic returns a function that recovers from panic and sends error to channel
// Usage: defer recoverBatchPanic(batchNum, resultsChan)()
func recoverBatchPanic[K comparable, V any](
	batchNum int,
	resultsChan chan<- BatchResult[K, V],
) func() {
	return func() {
		if r := recover(); r != nil {
			resultsChan <- BatchResult[K, V]{
				Err: fmt.Errorf("batch %d panicked: %v", batchNum, r),
			}
		}
	}
}
