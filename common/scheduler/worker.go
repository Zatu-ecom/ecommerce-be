package scheduler

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/log"

	"github.com/google/uuid"

	"github.com/go-redis/redis/v8"
)

const (
	defaultPoolSize       = 5
	pollInterval          = 500 * time.Millisecond
	delayedJobsKey        = "delayed_jobs"
	scheduledJobKeyPrefix = "scheduled_job:"
)

// StartRedisWorkerPool starts a background worker pool that processes delayed/scheduled jobs from Redis.
//
// How it works:
//  1. Jobs are stored in a Redis Sorted Set ("delayed_jobs") with score = Unix timestamp when job should execute
//  2. A dispatcher goroutine polls Redis every 500ms looking for jobs whose execution time has passed (score <= now)
//  3. Due jobs are sent to a buffered channel where worker goroutines pick them up for processing
//  4. Multiple workers process jobs concurrently, preventing slow jobs from blocking others
//
// Why we need this:
//   - Delayed execution: Schedule tasks to run at a specific future time (e.g., reservation expiry)
//   - Decoupled processing: HTTP requests return immediately, heavy work happens in background
//   - Reliability: Jobs persist in Redis, survive server restarts
//   - Scalability: Multiple workers process jobs concurrently, configurable via WORKER_POOL_SIZE env
//   - Non-blocking: Long-running jobs don't block other jobs from being processed
//
// Configuration:
//
//	WORKER_POOL_SIZE=10  # Number of concurrent workers (default: 5)
//
// Usage:
//
//	go scheduler.StartRedisWorkerPool() // Start in a goroutine at application startup
//
// To schedule a job:
//
//	rdb.ZAdd(ctx, "delayed_jobs", &redis.Z{
//	    Score:  float64(time.Now().Add(15*time.Minute).Unix()), // Execute 15 min from now
//	    Member: jobJSON,
//	})
func StartRedisWorkerPool() {
	poolSize := getPoolSize()
	jobChannel := make(chan ScheduledJob, poolSize*2)

	// Start worker pool
	for i := 1; i <= poolSize; i++ {
		go jobWorker(i, jobChannel)
	}

	log.Info("Redis worker pool started with " + strconv.Itoa(poolSize) + " workers")

	// Start dispatcher (runs in current goroutine)
	jobDispatcher(jobChannel)
}

// getPoolSize reads WORKER_POOL_SIZE from environment, defaults to 5
func getPoolSize() int {
	poolSizeStr := os.Getenv("WORKER_POOL_SIZE")
	if poolSizeStr == "" {
		return defaultPoolSize
	}

	poolSize, err := strconv.Atoi(poolSizeStr)
	if err != nil || poolSize <= 0 {
		log.Warn("Invalid WORKER_POOL_SIZE, using default: " + strconv.Itoa(defaultPoolSize))
		return defaultPoolSize
	}

	return poolSize
}

// jobWorker is a goroutine that continuously processes jobs from the channel
func jobWorker(id int, jobs <-chan ScheduledJob) {
	workerID := strconv.Itoa(id)
	rdb, _ := cache.GetRedisClient()
	ctx := context.Background()

	for job := range jobs {
		log.Info(
			"Worker " + workerID + " processing job: " + job.Command + " (jobId: " + job.JobID.String() + ")",
		)

		if err := Dispatch(job); err != nil {
			log.Error(
				"Worker "+workerID+" failed to dispatch job "+job.Command+": "+err.Error(),
				err,
			)
		}

		// Clean up the job key after processing (regardless of success/failure)
		if job.JobID != uuid.Nil && rdb != nil {
			jobKey := scheduledJobKeyPrefix + job.JobID.String()
			rdb.Del(ctx, jobKey)
		}
	}
}

// jobDispatcher polls Redis for due jobs and sends them to the worker channel
func jobDispatcher(jobs chan<- ScheduledJob) {
	ctx := context.Background()
	rdb, err := cache.GetRedisClient()
	if err != nil {
		log.Error("Failed to start dispatcher: "+err.Error(), err)
		close(jobs)
		return
	}

	for {
		now := time.Now().Unix()

		// Fetch jobs that are due (score <= current timestamp)
		results, err := rdb.ZRangeByScore(ctx, delayedJobsKey, &redis.ZRangeBy{
			Min:   "0",
			Max:   strconv.FormatInt(now, 10),
			Count: 10, // Fetch multiple jobs at once for efficiency
		}).Result()
		if err != nil {
			log.Error("Failed to fetch jobs from Redis: "+err.Error(), err)
			time.Sleep(pollInterval)
			continue
		}

		if len(results) == 0 {
			time.Sleep(pollInterval)
			continue
		}

		// Process each due job
		for _, jobData := range results {
			// Atomically remove the job - only process if we successfully removed it
			// This prevents duplicate processing in multi-instance environments
			if rdb.ZRem(ctx, delayedJobsKey, jobData).Val() == 1 {
				var job ScheduledJob
				if err := json.Unmarshal([]byte(jobData), &job); err != nil {
					log.Error("Failed to unmarshal job: "+err.Error(), err)
					continue
				}

				// Send to worker channel (non-blocking with buffered channel)
				jobs <- job
			}
		}
	}
}
