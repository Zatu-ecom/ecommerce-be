package scheduler

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"ecommerce-be/common/cachekit"
	"ecommerce-be/common/cachekit/provider"
	"ecommerce-be/common/config"
	"ecommerce-be/common/constants"
	"ecommerce-be/common/log"

	"github.com/gin-gonic/gin"
)

const (
	defaultPoolSize = 5
	pollInterval    = 500 * time.Millisecond
)

// StartRedisWorkerPool starts a background worker pool that processes delayed/scheduled jobs.
//
// How it works:
//  1. Jobs live on the durable KV role, claimed atomically (exactly one pod per job)
//  2. A dispatcher goroutine polls the DelayQueue looking for due jobs
//  3. Due jobs are sent to a buffered channel where worker goroutines pick them up
//  4. Multiple workers process jobs concurrently, preventing slow jobs from blocking others
//
// Why we need this:
//   - Delayed execution: Schedule tasks to run at a specific future time (e.g., reservation expiry)
//   - Decoupled processing: HTTP requests return immediately, heavy work happens in background
//   - Reliability: Jobs persist on durable KV, survive server restarts
//   - Scalability: Multiple workers process jobs concurrently, configurable via WORKER_POOL_SIZE env
//   - Non-blocking: Long-running jobs don't block other jobs from being processed
//
// Startup gate: without reachable durable KV the pool does NOT start (jobs are
// correctness infrastructure — silently running no workers would drop expiries).
// Callers (main.go) run this in a goroutine at application startup.
//
// Configuration:
//
//	WORKER_POOL_SIZE=10  # Number of concurrent workers (default: 5)
func StartRedisWorkerPool() {
	cfg := config.Get()
	if cfg == nil {
		log.Error("scheduler: config not loaded, worker pool not started", nil)
		return
	}
	queue := provider.NewDurable(cfg.Redis)

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Ping capability is exposed through a lightweight probe: Poll with an
	// empty window fails fast when the backend is unreachable.
	if err := probeQueue(pingCtx, queue); err != nil {
		log.Error("scheduler: durable KV unreachable, worker pool not started: "+err.Error(), err)
		return
	}

	poolSize := getPoolSize()
	jobChannel := make(chan claimedJob, poolSize*2)

	// Start worker pool
	for i := 1; i <= poolSize; i++ {
		go jobWorker(i, queue, jobChannel)
	}

	log.Info("Redis worker pool started with " + strconv.Itoa(poolSize) + " workers")

	// Start dispatcher (runs in current goroutine)
	jobDispatcher(queue, jobChannel)
}

// claimedJob pairs a claimed payload with its transport ID for post-run cleanup.
type claimedJob struct {
	jobID   string
	payload []byte
}

// probeQueue verifies the queue is reachable without disturbing it.
func probeQueue(ctx context.Context, queue cachekit.DelayQueue) error {
	_, err := queue.Poll(ctx, 1)
	return err
}

// getPoolSize reads worker pool size from config, defaults to 5
func getPoolSize() int {
	cfg := config.Get()
	if cfg == nil {
		return defaultPoolSize
	}

	poolSize := cfg.Scheduler.WorkerPoolSize
	if poolSize <= 0 {
		return defaultPoolSize
	}

	return poolSize
}

// jobWorker is a goroutine that continuously processes jobs from the channel
func jobWorker(id int, queue cachekit.DelayQueue, jobs <-chan claimedJob) {
	workerID := strconv.Itoa(id)

	for claimed := range jobs {
		var job ScheduledJob
		if err := json.Unmarshal(claimed.payload, &job); err != nil {
			log.Error("Worker "+workerID+" failed to unmarshal job payload: "+err.Error(), err)
			continue
		}
		ctx := GetContextWithKeys(job)
		log.InfoWithContext(
			ctx,
			"Worker "+workerID+" processing job: "+job.Command+" (jobId: "+job.JobID.String()+")",
		)

		if err := Dispatch(job, ctx); err != nil {
			log.ErrorWithContext(
				ctx,
				"Worker "+workerID+" failed to dispatch job "+job.Command+" (jobId: "+job.JobID.String()+"): "+err.Error(),
				err,
			)
		}

		// Clean up the job index after processing (regardless of success/failure).
		// Cancel is idempotent: already-gone jobs are a nil no-op.
		if claimed.jobID != "" {
			if err := queue.Cancel(context.Background(), claimed.jobID); err != nil {
				log.ErrorWithContext(ctx, "Worker "+workerID+" failed to clean up job "+claimed.jobID+": "+err.Error(), err)
			}
		}
	}
}

// jobDispatcher polls the queue for due jobs and sends them to the worker channel.
// Claiming already happened inside Poll, so every payload here is ours alone,
// on this pod or any other.
func jobDispatcher(queue cachekit.DelayQueue, jobs chan<- claimedJob) {
	ctx := context.Background()

	for {
		payloads, err := queue.Poll(ctx, 10)
		if err != nil {
			log.Error("Failed to poll delayed jobs: "+err.Error(), err)
			time.Sleep(pollInterval)
			continue
		}

		if len(payloads) == 0 {
			time.Sleep(pollInterval)
			continue
		}

		for _, payload := range payloads {
			var probe struct {
				JobID string `json:"jobId"`
			}
			jobID := ""
			if uerr := json.Unmarshal(payload, &probe); uerr == nil {
				jobID = probe.JobID
			}
			// Send to worker channel (blocks when workers are saturated —
			// claimed jobs wait in memory, never back in the queue).
			jobs <- claimedJob{jobID: jobID, payload: payload}
		}
	}
}

func GetContextWithKeys(job ScheduledJob) context.Context {
	ctx := &gin.Context{
		Keys: map[string]any{
			constants.USER_ID_KEY:        job.UserID,
			constants.SELLER_ID_KEY:      job.SellerID,
			constants.CORRELATION_ID_KEY: job.CorrelationId,
		},
	}
	return ctx
}
