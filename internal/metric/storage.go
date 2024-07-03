package metric

import (
	influxdb2 "github.com/influxdata/influxdb-client-go"
	"github.com/influxdata/influxdb-client-go/api"
	"github.com/influxdata/influxdb-client-go/api/write"
)

type Storage struct {
	client influxdb2.Client
}

func NewStorage(client influxdb2.Client) *Storage {
	return &Storage{client: client}
}

func (s *Storage) WriteSession(org, bucket string) (*WriteSession, <-chan error) {
	writer := s.client.WriteAPI(org, bucket)
	return &WriteSession{writer: writer}, writer.Errors()
}

type WriteSession struct {
	writer api.WriteAPI
}

func (ws *WriteSession) Write(point *write.Point) {
	ws.writer.WritePoint(point)
}

func (ws *WriteSession) Flush() {
	ws.writer.Flush()
}

func (ws *WriteSession) Close() {
	ws.writer.Flush()
	ws.writer.Close()
}
