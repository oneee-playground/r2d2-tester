package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/docker/docker/client"
	influxdb2 "github.com/influxdata/influxdb-client-go"
	conf "github.com/oneee-playground/r2d2-tester/internal/config"
	"github.com/oneee-playground/r2d2-tester/internal/event"
	"github.com/oneee-playground/r2d2-tester/internal/job"
	"github.com/oneee-playground/r2d2-tester/internal/metric"
	"github.com/oneee-playground/r2d2-tester/internal/server"
	"github.com/oneee-playground/r2d2-tester/internal/work/storage"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	conf.LoadFromEnv()

	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(os.Stdout), zap.DebugLevel,
	))

	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logger.Fatal("failed to initialize docker client", zap.Error(err))
	}

	httpClient := &http.Client{}
	storage := storage.NewFSStorage(conf.WorkStoragePath)

	awsConfig := aws.Config{
		Region:      "ap-northeast-2",
		Credentials: credentials.NewStaticCredentialsProvider(conf.AccessKeyID, conf.SecretAccessKey, ""),
	}

	sqsClient := sqs.NewFromConfig(awsConfig)

	influxCilent := influxdb2.NewClientWithOptions(conf.InfluxURL, conf.InfluxToken, influxdb2.DefaultOptions())

	jobPoller := job.NewPoller(sqsClient, conf.JobQueueURL)
	eventPublisher := event.NewSQSEventBus(sqsClient, logger, conf.EventQueueURL)
	metricStorage := metric.NewStorage(influxCilent)

	serverOpts := server.ServerOpts{
		JobPoller:      jobPoller,
		PollInterval:   10 * time.Second,
		Docker:         dockerClient,
		EventPublisher: eventPublisher,
		HTTPClient:     httpClient,
		WorkStorage:    storage,
		MetricStorage:  metricStorage,
	}

	srv := server.New(logger, serverOpts)
	if err := srv.Run(context.TODO()); err != nil {
		logger.Fatal("serve failed", zap.Error(err))
	}
}
