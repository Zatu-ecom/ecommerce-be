package cron

import (
	"fmt"
	"runtime/debug"
	"time"

	"ecommerce-be/common/log"

	robfigCron "github.com/robfig/cron/v3"
)

// Scheduler manages all recurring background jobs in the system
type Scheduler struct {
	cron *robfigCron.Cron
}

var DefaultScheduler *Scheduler

// Init initializes the global cron scheduler
func Init() {
	if DefaultScheduler != nil {
		return
	}

	logger := robfigCron.PrintfLogger(panicLogger{})

	c := robfigCron.New(
		robfigCron.WithSeconds(),
		robfigCron.WithChain(
			robfigCron.Recover(logger),
		),
	)

	DefaultScheduler = &Scheduler{
		cron: c,
	}
}

// RegisterJob adds a new recurring job to the global scheduler.
// schedule: standard cron expression with 6 fields:
//
//	seconds(0-59), minutes(0-59), hours(0-23), day of month(1-31), month(1-12), day of week(0-6)
//	Example: "0 0 * * * *" (runs at minute 0 of every hour)
//
// name: unique identifier used for logging job execution and tracking
func RegisterJob(schedule string, name string, cmd func()) error {
	if DefaultScheduler == nil {
		return fmt.Errorf("cron scheduler not initialized")
	}

	wrappedCmd := func() {
		log.Info(fmt.Sprintf("[CRON] Starting job: %s", name))
		start := time.Now()
		cmd()
		log.Info(fmt.Sprintf("[CRON] Completed job: %s (took %v)", name, time.Since(start)))
	}

	_, err := DefaultScheduler.cron.AddFunc(schedule, wrappedCmd)
	if err != nil {
		return fmt.Errorf("failed to register cron job %s: %w", name, err)
	}

	log.Info(fmt.Sprintf("Registered cron job: %s (Schedule: %s)", name, schedule))
	return nil
}

// RegisterIntervalJob registers a job to run at a specific duration interval.
// interval: how frequently to run the job (e.g., 1*time.Minute, 24*time.Hour).
//
//	This uses the "@every <duration>" syntax under the hood.
//
// name: unique identifier used for logging job execution and tracking
func RegisterIntervalJob(interval time.Duration, name string, cmd func()) error {
	// robfig/cron supports the @every syntax (e.g., "@every 1m30s")
	schedule := fmt.Sprintf("@every %s", interval.String())
	return RegisterJob(schedule, name, cmd)
}

// RegisterDailyJob registers a job to run every day at a specific time and timezone.
// hour: the hour to run using 24-hour format (0-23). For example, 0 = Midnight, 12 = Noon, 18 = 6 PM.
// minute: the minute of the hour (0-59) to run at.
// tz: the timezone location name (e.g., "Asia/Kolkata", "UTC", "America/New_York").
//
//	If empty strings "" is provided, it defaults to the local system time of the server.
//
// name: unique identifier used for logging job execution and tracking
func RegisterDailyJob(hour int, minute int, tz string, name string, cmd func()) error {
	tzPrefix := ""
	if tz != "" {
		tzPrefix = fmt.Sprintf("CRON_TZ=%s ", tz)
	}
	// robfig/cron WithSeconds expects 6 fields: sec min hour dom mon dow
	// "0 30 21 * * *" means 0 seconds, 30 past, 9 PM, every day
	schedule := fmt.Sprintf("%s0 %d %d * * *", tzPrefix, minute, hour)
	return RegisterJob(schedule, name, cmd)
}

// Start begins execution of all registered jobs in background goroutines
func Start() {
	if DefaultScheduler != nil {
		log.Info("Starting cron scheduler")
		DefaultScheduler.cron.Start()
	}
}

// Stop gracefully stops the scheduler, waiting for running jobs to complete
func Stop() {
	if DefaultScheduler != nil {
		log.Info("Stopping cron scheduler (waiting for running jobs...)")
		ctx := DefaultScheduler.cron.Stop()
		<-ctx.Done()
		log.Info("Cron scheduler stopped cleanly")
	}
}

// panicLogger implements the robfig/cron.Logger interface to route panics to our logger
type panicLogger struct{}

func (l panicLogger) Printf(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	// In the robfig/cron recovery chain, panics are passed here
	log.Error("CRON JOB PANIC", fmt.Errorf("%s\nStack:\n%s", formatted, string(debug.Stack())))
}
