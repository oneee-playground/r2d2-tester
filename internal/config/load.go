package config

import (
	"os"
)

func LoadFromEnv() {
	WorkStoragePath = os.Getenv("WORK_STORAGE_PATH")

	InfluxURL = os.Getenv("INFLUX_URL")
	InfluxToken = os.Getenv("INFLUX_TOKEN")

	JobQueueURL = os.Getenv("AWS_SQS_JOB_QUEUE_URL")
	EventQueueURL = os.Getenv("AWS_SQS_TEST_EVENT_QUEUE_URL")

	AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
}
