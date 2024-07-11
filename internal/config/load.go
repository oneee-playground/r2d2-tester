package config

import (
	"os"
)

func LoadFromEnv() {
	WorkStoragePath = os.Getenv("WORK_STORAGE_PATH")

	InfluxURL = os.Getenv("INFLUX_URL")
	InfluxToken = os.Getenv("INFLUX_PATH")

	JobQueueURL = os.Getenv("JOB_QUEUE_URL")
	EventQueueURL = os.Getenv("EVENT_QUEUE_URL")
}
