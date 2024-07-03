package exec_test

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	influxdb2 "github.com/influxdata/influxdb-client-go"
	"github.com/oneee-playground/r2d2-tester/internal/exec"
	"github.com/oneee-playground/r2d2-tester/internal/job"
	"github.com/oneee-playground/r2d2-tester/internal/metric"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/oneee-playground/r2d2-tester/internal/work/storage"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ExecSuite struct {
	suite.Suite
	log           *zap.Logger
	httpClient    *http.Client
	workStorage   work.Storage
	metricStroage *metric.Storage
	docker        client.APIClient

	influxClient influxdb2.Client

	tempdir                  string
	execNetwork, testNetwork string
}

func TestExecSuite(t *testing.T) {
	suite.Run(t, new(ExecSuite))
}

func (s *ExecSuite) SetupSuite() {
	s.execNetwork = "exec-network"
	s.testNetwork = "test-network"

	s.tempdir = s.T().TempDir()
	s.Require().NoError(generateTestData("testdata", s.tempdir))

	s.httpClient = http.DefaultClient

	s.workStorage = storage.NewFSStorage(s.tempdir)

	s.log = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(os.Stdout),
		zap.DebugLevel,
	), zap.AddCaller())

	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	s.Require().NoError(err)

	s.docker = client
	_, err = s.docker.NetworkCreate(context.Background(), s.execNetwork, network.CreateOptions{
		Driver:   network.NetworkBridge,
		Internal: true,
	})
	s.Require().NoError(err)

	options := influxdb2.DefaultOptions()

	// It's okay. It is only tested in local.
	const token = "Qgfb1CR5cmJFD7AlwwKDULCR7TRi7SWYIyqI096U-J3mqM8wbxEQErPPJbH8Ei830OYvz2J56buiRD1F3mp7Yg=="

	s.influxClient = influxdb2.NewClientWithOptions("http://influxdb:8086", token, options)

	s.metricStroage = metric.NewStorage(s.influxClient)
}

func (s *ExecSuite) TearDownSuite() {
	s.Require().NoError(s.docker.NetworkRemove(context.Background(), s.execNetwork))
}

func (s *ExecSuite) TestExecutor() {
	defer goleak.VerifyNone(s.T())

	opts := exec.ExecOpts{
		Log:           s.log,
		HTTPClient:    s.httpClient,
		WorkStorage:   s.workStorage,
		Docker:        s.docker,
		MetricStorage: s.metricStroage,
		ExecNetwork:   s.execNetwork,
		TestNetwork:   s.testNetwork,
	}

	job := job.Job{
		TaskID: uuid.Nil,
		Resources: []job.Resource{
			{
				Name:      "app",
				Port:      4000,
				CPU:       1,
				Memory:    100 * 1024 * 1024,
				IsPrimary: true,
			},
		},
		Sections: []job.Section{
			{
				ID:   uuid.Nil,
				Type: job.TypeScenario,
			},
			{
				ID:   uuid.Nil,
				Type: job.TypeLoad,
				RPM:  1000,
			},
		},
		Submission: job.Submission{
			ID:         uuid.Nil,
			Repository: "oneee-playground/hello-docker",
			CommitHash: "4d699f27bf2b5e67b3bc0a6195ef75ad6ac04112",
		},
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := exec.NewExecutor(opts).Execute(ctx, job)
	s.NoError(err)

	s.influxClient.Close()
	s.docker.Close()
}
