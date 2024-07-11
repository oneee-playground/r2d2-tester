package job

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pkg/errors"
)

var NoErrEmptyJobs = errors.New("jobs are empty")

type Poller interface {
	Poll(ctx context.Context) (id string, job Job, err error)
	MarkAsDone(ctx context.Context, id string) (err error)
}

type poller struct {
	client   *sqs.Client
	queueURL string
}

func NewPoller(sqs *sqs.Client, queueURL string) *poller {
	return &poller{
		client:   sqs,
		queueURL: queueURL,
	}
}

func (p *poller) Poll(ctx context.Context) (id string, job Job, err error) {
	input := &sqs.ReceiveMessageInput{QueueUrl: aws.String(p.queueURL)}

	result, err := p.client.ReceiveMessage(ctx, input)
	if err != nil {
		return "", Job{}, errors.Wrap(err, "receiving message")
	}

	if len(result.Messages) == 0 {
		return "", Job{}, NoErrEmptyJobs
	}

	msg := result.Messages[0]

	var decoded Job
	if err := json.Unmarshal([]byte(*msg.Body), &job); err != nil {
		return "", Job{}, errors.Wrap(err, "failed to unmarshal job")
	}

	return *msg.MessageId, decoded, nil
}

func (p *poller) MarkAsDone(ctx context.Context, id string) (err error) {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(p.queueURL),
		ReceiptHandle: aws.String(id),
	}

	if _, err = p.client.DeleteMessage(ctx, input); err != nil {
		return errors.Wrap(err, "failed to delete message")
	}

	return nil
}
