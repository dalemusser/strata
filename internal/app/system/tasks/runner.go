// internal/app/system/tasks/runner.go
package tasks

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Job represents a scheduled background task.
type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

// Runner manages background job execution.
type Runner struct {
	logger *zap.Logger
	jobs   []Job
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// New creates a new task runner.
func New(logger *zap.Logger) *Runner {
	return &Runner{
		logger: logger,
	}
}

// Register adds a job to the runner.
func (r *Runner) Register(job Job) {
	r.jobs = append(r.jobs, job)
}

// Start begins executing all registered jobs.
// Call Stop to gracefully shutdown.
func (r *Runner) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	for _, job := range r.jobs {
		r.wg.Add(1)
		go r.runJob(ctx, job)
	}

	r.logger.Info("background task runner started",
		zap.Int("job_count", len(r.jobs)))
}

// Stop gracefully stops all running jobs.
func (r *Runner) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	r.logger.Info("background task runner stopped")
}

// runJob executes a single job on its interval.
func (r *Runner) runJob(ctx context.Context, job Job) {
	defer r.wg.Done()

	// Run immediately on startup
	r.executeJob(ctx, job)

	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Debug("job stopped", zap.String("job", job.Name))
			return
		case <-ticker.C:
			r.executeJob(ctx, job)
		}
	}
}

// executeJob runs a job and logs the result.
func (r *Runner) executeJob(ctx context.Context, job Job) {
	start := time.Now()
	r.logger.Debug("job starting", zap.String("job", job.Name))

	if err := job.Run(ctx); err != nil {
		r.logger.Error("job failed",
			zap.String("job", job.Name),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err))
		return
	}

	r.logger.Debug("job completed",
		zap.String("job", job.Name),
		zap.Duration("duration", time.Since(start)))
}

// RunOnce executes a job immediately (useful for testing or manual triggers).
func (r *Runner) RunOnce(ctx context.Context, name string) error {
	for _, job := range r.jobs {
		if job.Name == name {
			return job.Run(ctx)
		}
	}
	return nil
}
