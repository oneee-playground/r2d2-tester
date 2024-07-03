package server

import (
	"context"
	"time"

	"github.com/oneee-playground/r2d2-tester/internal/event"
	"github.com/oneee-playground/r2d2-tester/internal/exec"
	"github.com/oneee-playground/r2d2-tester/internal/job"
	"go.uber.org/zap"
)

type Server struct {
	log *zap.Logger

	jobPoller    job.Poller
	pollInterval time.Duration

	eventPublisher event.Publisher
}

func (s *Server) Run(ctx context.Context) error {
	s.log.Info("Server running")

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		id, received, err := s.jobPoller.Poll(ctx)
		if err != nil {
			if err != job.NoErrEmptyJobs {
				s.log.Error("failed to poll a job", zap.Error(err))
			}
			continue
		}

		submissionID := received.Submission.ID

		s.log.Info("polled job", zap.String("submissionID", submissionID.String()))

		submissionLog := s.log.With(zap.String("submissionID", submissionID.String()))

		start := time.Now()

		opts := exec.ExecOpts{
			Log: submissionLog,
			// HTTPClient:  ,
			// WorkStorage: ,
			// Docker:      ,
			// MetricStorage: ,
		}

		err = exec.NewExecutor(opts).Execute(ctx, received)
		if err != nil {
			s.log.Error("failed to execute a job", zap.Error(err))
		}

		event := event.TestEvent{
			ID:      submissionID,
			Success: err == nil,
			Took:    time.Since(start),
		}

		if err != nil {
			event.Extra = err.Error()
		}

		if err := s.jobPoller.MarkAsDone(ctx, id); err != nil {
			s.log.Error("failed to mark a job as done", zap.Error(err))
			continue
		}

		if err := s.eventPublisher.Publish(ctx, event); err != nil {
			s.log.Error("failed to execute a job", zap.Error(err))
		}
	}
}
