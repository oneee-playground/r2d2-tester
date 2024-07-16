package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/oneee-playground/r2d2-tester/internal/work/storage"
)

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

func main() {
	storage := storage.NewFSStorage(".")

	taskID := uuid.MustParse("0c4747d5-41ea-4ac8-82c7-b18aab504671")
	sectionID := uuid.MustParse("2ee048bc-9af9-410d-8f37-80634bb73bdd")

	id := uuid.New()
	fmt.Println(id)

	bodySchema, err := loadContent("input-schema.json")
	if err != nil {
		panic(err)
	}

	template := &work.Template{
		Id: id[:],
		SchemaTable: map[uint32]*work.TemplatedSchema{
			200: {
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				BodySchema: bodySchema,
			},
		},
	}

	err = storage.InsertTemplate(context.Background(), taskID, sectionID, template)
	if err != nil {
		panic(err)
	}

	return

	// taskID := uuid.MustParse("0c4747d5-41ea-4ac8-82c7-b18aab504671")
	// sectionID := uuid.MustParse("515be74f-ab64-49e0-b10a-b0fbf14e42bf")

	// workID := uuid.New()
	// work1 := &work.Work{
	// 	Id: workID[:],
	// 	Input: &work.Input{
	// 		Method: "POST",
	// 		Path:   "/boards",
	// 		Body:   []byte(`{"title":"First Board!","description":"Hello World!"}`),
	// 	},
	// 	ExpectedValue: &work.Expected{Status: http.StatusCreated},
	// 	Timeout:       durationpb.New(time.Second),
	// }

	// err := storage.InsertWork(context.Background(), taskID, sectionID, work1)
	// if err != nil {
	// 	panic(err)
	// }

	// workID = uuid.New()
	// work2 := &work.Work{
	// 	Id: workID[:],
	// 	Input: &work.Input{
	// 		Method: "GET",
	// 		Path:   "/boards",
	// 	},
	// 	ExpectedValue: &work.Expected{
	// 		Status: http.StatusOK,
	// 		Headers: map[string]string{
	// 			"Content-Type": "application/json",
	// 		},
	// 		Body: []byte(`[{"id":1,"title":"First Board!"}]`),
	// 	},
	// 	Timeout: durationpb.New(time.Second),
	// }

	// err = storage.InsertWork(context.Background(), taskID, sectionID, work2)
	// if err != nil {
	// 	panic(err)
	// }

	// workID = uuid.New()
	// work3 := &work.Work{
	// 	Id: workID[:],
	// 	Input: &work.Input{
	// 		Method: "GET",
	// 		Path:   "/boards/1",
	// 	},
	// 	ExpectedValue: &work.Expected{
	// 		Status: http.StatusOK,
	// 		Headers: map[string]string{
	// 			"Content-Type": "application/json",
	// 		},
	// 		Body: []byte(`{"id":1,"title":"First Board!","description":"Hello World!"}`),
	// 	},
	// 	Timeout: durationpb.New(time.Second),
	// }

	// err = storage.InsertWork(context.Background(), taskID, sectionID, work3)
	// if err != nil {
	// 	panic(err)
	// }

	// workID = uuid.New()
	// work4 := &work.Work{
	// 	Id: workID[:],
	// 	Input: &work.Input{
	// 		Method: "PUT",
	// 		Path:   "/boards/1",
	// 		Body:   []byte(`{"title":"First Board?","description":"Hello World?"}`),
	// 	},
	// 	ExpectedValue: &work.Expected{Status: http.StatusOK},
	// 	Timeout:       durationpb.New(time.Second),
	// }

	// err = storage.InsertWork(context.Background(), taskID, sectionID, work4)
	// if err != nil {
	// 	panic(err)
	// }

	// workID = uuid.New()
	// work5 := &work.Work{
	// 	Id: workID[:],
	// 	Input: &work.Input{
	// 		Method: "GET",
	// 		Path:   "/boards/1",
	// 	},
	// 	ExpectedValue: &work.Expected{
	// 		Status: http.StatusOK,
	// 		Headers: map[string]string{
	// 			"Content-Type": "application/json",
	// 		},
	// 		Body: []byte(`{"id":1,"title":"First Board?","description":"Hello World?"}`),
	// 	},
	// 	Timeout: durationpb.New(time.Second),
	// }

	// err = storage.InsertWork(context.Background(), taskID, sectionID, work5)
	// if err != nil {
	// 	panic(err)
	// }

	// workID = uuid.New()
	// work6 := &work.Work{
	// 	Id: workID[:],
	// 	Input: &work.Input{
	// 		Method: "DELETE",
	// 		Path:   "/boards/1",
	// 	},
	// 	ExpectedValue: &work.Expected{Status: http.StatusNoContent},
	// 	Timeout:       durationpb.New(time.Second),
	// }

	// err = storage.InsertWork(context.Background(), taskID, sectionID, work6)
	// if err != nil {
	// 	panic(err)
	// }

	// workID = uuid.New()
	// work7 := &work.Work{
	// 	Id: workID[:],
	// 	Input: &work.Input{
	// 		Method: "GET",
	// 		Path:   "/boards",
	// 	},
	// 	ExpectedValue: &work.Expected{
	// 		Status: http.StatusOK,
	// 		Headers: map[string]string{
	// 			"Content-Type": "application/json",
	// 		},
	// 		Body: []byte(`[]`),
	// 	},
	// 	Timeout: durationpb.New(time.Second),
	// }

	// err = storage.InsertWork(context.Background(), taskID, sectionID, work7)
	// if err != nil {
	// 	panic(err)
	// }
}
