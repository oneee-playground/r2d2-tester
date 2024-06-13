package event

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type SQSEventPublisher struct {
	client *sqs.Client
	logger *zap.Logger

	queueURL string
}

var (
	_ Publisher = (*SQSEventPublisher)(nil)
)

func NewSQSEventBus(client *sqs.Client, logger *zap.Logger, queueURL string) *SQSEventPublisher {
	return &SQSEventPublisher{
		client:   client,
		logger:   logger,
		queueURL: queueURL,
	}
}

func (p *SQSEventPublisher) Publish(ctx context.Context, e TestEvent) error {
	payload, err := json.Marshal(e)
	if err != nil {
		return errors.Wrap(err, "marshalling payload")
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(p.queueURL),
		MessageBody: aws.String(string(payload)),
	}

	if _, err := p.client.SendMessage(ctx, input); err != nil {
		return errors.Wrap(err, "sending message")
	}

	return nil
}
