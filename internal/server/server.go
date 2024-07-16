package server

import (
	"context"
	"net/http"
	"time"

	"github.com/docker/docker/client"
	"github.com/oneee-playground/r2d2-tester/internal/event"
	"github.com/oneee-playground/r2d2-tester/internal/exec"
	"github.com/oneee-playground/r2d2-tester/internal/job"
	"github.com/oneee-playground/r2d2-tester/internal/metric"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"go.uber.org/zap"
)

type ServerOpts struct {
	JobPoller    job.Poller
	PollInterval time.Duration

	EventPublisher event.Publisher
	HTTPClient     *http.Client
	WorkStorage    work.Storage
	MetricStorage  *metric.Storage
	Docker         client.APIClient
}

type Server struct {
	ServerOpts
	log *zap.Logger
}

func New(log *zap.Logger, opts ServerOpts) *Server {
	return &Server{
		log:        log,
		ServerOpts: opts,
	}
}

func (s *Server) Run(ctx context.Context) error {
	s.log.Info("Server running")

	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		id, received, err := s.JobPoller.Poll(ctx)
		if err != nil {
			if err != job.NoErrEmptyJobs {
				s.log.Error("failed to poll a job", zap.Error(err))
			}
			continue
		}

		s.log.Info("polled job", zap.Any("job", received))

		submissionID := received.Submission.ID

		s.log.Info("polled job", zap.String("submissionID", submissionID.String()))

		submissionLog := s.log.With(zap.String("submissionID", submissionID.String()))

		start := time.Now()

		opts := exec.ExecOpts{
			Log:           submissionLog,
			HTTPClient:    s.HTTPClient,
			WorkStorage:   s.WorkStorage,
			Docker:        s.Docker,
			MetricStorage: s.MetricStorage,
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

		if err := s.JobPoller.MarkAsDone(ctx, id); err != nil {
			s.log.Error("failed to mark a job as done", zap.Error(err))
			continue
		}

		if err := s.EventPublisher.Publish(ctx, event); err != nil {
			s.log.Error("failed to execute a job", zap.Error(err))
		}
	}
}
