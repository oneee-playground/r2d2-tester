package exec

import (
	"context"
	"net/http"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/oneee-playground/r2d2-tester/internal/job"
	"github.com/oneee-playground/r2d2-tester/internal/metric"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"
)

type process struct {
	ID       string
	Hostname string
	Port     uint16
	Image    string
}

type ExecOpts struct {
	ExecNetwork string
	TestNetwork string

	Log           *zap.Logger
	HTTPClient    *http.Client
	WorkStorage   work.Storage
	MetricStorage *metric.Storage
	Docker        client.APIClient
}

type Executor struct {
	processes      []*process
	primaryProcess *process

	metrics *metric.WriteSession

	ExecOpts
}

func NewExecutor(opts ExecOpts) *Executor {
	e := &Executor{ExecOpts: opts}
	return e
}

func (e *Executor) Execute(ctx context.Context, jobToExec job.Job) error {
	e.Log.Info("execution started")

	start := time.Now()
	taskID := jobToExec.TaskID

	defer e.teardownResources(ctx)
	if err := e.setupResources(ctx, taskID, jobToExec.Resources, jobToExec.Submission); err != nil {
		return errors.Wrap(err, "setting up resources")
	}

	// TODO: Add health check for containers. You can do it.
	// Replaced by time sleep inside setupResources
	// time.Sleep(5 * time.Second)

	ctx, cancel := context.WithCancelCause(ctx)
	session, errchan := e.MetricStorage.WriteSession(taskID.String(), jobToExec.Submission.ID.String())
	go func() {
		if err, ok := <-errchan; ok {
			cancel(err)
		}
	}()

	defer session.Close()
	e.metrics = session

	e.startMetricCollection(ctx, cancel)

	for idx, section := range jobToExec.Sections {
		e.setTimestamp(time.Now(), section.ID, "start-exec")
		e.Log.Info("started execution of section",
			zap.Int("index", idx),
			zap.String("id", section.ID.String()),
		)

		templates, err := e.fetchTemplates(ctx, taskID, section.ID)
		if err != nil {
			return err
		}

		stream, errchan := e.WorkStorage.Stream(ctx, taskID, section.ID)

		e.Log.Info("determined section type", zap.String("type", string(section.Type)))

		start := time.Now()
		e.setTimestamp(start, section.ID, "start-request")

		switch section.Type {
		case job.TypeScenario:
			err = e.testScenario(ctx, section.ID, templates, stream, errchan)
		case job.TypeLoad:
			var dueMissed int
			dueMissed, err = e.testLoad(ctx, section.ID, section.RPM, templates, stream, errchan)

			if dueMissed > 0 {
				e.Log.Info("test has missed dues", zap.Int("missed", dueMissed))
			}
		}

		e.setTimestamp(time.Now(), section.ID, "request-done")

		e.Log.Info("section execution done", zap.Duration("took", time.Since(start)))

		if err != nil {
			cancel(err)
			return errors.Wrapf(err, "testing %s", section.Type)
		}
	}

	e.Log.Info("execution done", zap.Duration("took", time.Since(start)))

	return nil
}

func (e *Executor) fetchTemplates(ctx context.Context, taskID, sectionID uuid.UUID) (map[uuid.UUID]template, error) {
	rawTemplates, err := e.WorkStorage.FetchTemplates(ctx, taskID, sectionID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching templates")
	}

	templates := make(map[uuid.UUID]template, len(rawTemplates))
	for key, val := range rawTemplates {
		schema, err := processTemplate(val)
		if err != nil {
			return nil, err
		}

		templates[key] = schema
	}

	return templates, nil
}

type schema struct {
	headers    map[string]string
	jsonSchema *gojsonschema.Schema
}

type template struct {
	schemaTable map[int]schema
}

func processTemplate(workTemplate *work.Template) (template, error) {
	t := template{schemaTable: make(map[int]schema, len(workTemplate.SchemaTable))}

	for status, val := range workTemplate.SchemaTable {
		var s *gojsonschema.Schema

		if len(val.BodySchema) > 0 {
			loader := gojsonschema.NewBytesLoader(val.BodySchema)

			schema, err := gojsonschema.NewSchema(loader)
			if err != nil {
				return template{}, errors.Wrap(err, "creating schema")
			}

			s = schema
		}

		t.schemaTable[int(status)] = schema{
			headers:    val.Headers,
			jsonSchema: s,
		}
	}

	return t, nil
}
