package exec_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/oneee-playground/r2d2-tester/internal/work/storage"
	"google.golang.org/protobuf/types/known/durationpb"
)

func generateTestData(datadir, tempdir string) error {
	storage := storage.NewFSStorage(tempdir)

	b, err := loadContent(filepath.Join(datadir, "test-schema.json"))
	if err != nil {
		return err
	}

	template := &work.Template{
		Id: uuid.Nil[:],
		SchemaTable: map[uint32]*work.TemplatedSchema{
			http.StatusTeapot: {
				Headers:    nil,
				BodySchema: nil,
			},
			http.StatusOK: {
				Headers:    nil,
				BodySchema: b,
			},
		},
	}

	err = storage.InsertTemplate(context.Background(), uuid.Nil, uuid.Nil, template)
	if err != nil {
		return err
	}

	b, err = loadContent(filepath.Join(datadir, "expected-body.txt"))
	if err != nil {
		return err
	}

	work1 := &work.Work{
		Id: uuid.Nil[:],
		Input: &work.Input{
			Method: "GET",
			Path:   "/",
			Body:   nil,
		},
		TemplateId: nil,
		ExpectedValue: &work.Expected{
			Status: http.StatusOK,
			Body:   b,
		},
		Timeout: durationpb.New(time.Minute),
	}

	err = storage.InsertWork(context.Background(), uuid.Nil, uuid.Nil, work1)
	if err != nil {
		return err
	}

	work2 := &work.Work{
		Id: uuid.Nil[:],
		Input: &work.Input{
			Method: "GET",
			Path:   "/vary",
			Body:   nil,
		},
		TemplateId: uuid.Nil[:],
		Timeout:    durationpb.New(time.Minute),
	}

	err = storage.InsertWork(context.Background(), uuid.Nil, uuid.Nil, work2)
	if err != nil {
		return err
	}

	work3 := &work.Work{
		Id: uuid.Nil[:],
		Input: &work.Input{
			Method: "POST",
			Path:   "/vary",
			Body:   nil,
		},
		TemplateId: uuid.Nil[:],
		Timeout:    durationpb.New(time.Minute),
	}

	err = storage.InsertWork(context.Background(), uuid.Nil, uuid.Nil, work3)
	if err != nil {
		return err
	}

	return nil
}

func loadContent(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return b, nil
}
