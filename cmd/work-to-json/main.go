package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/oneee-playground/r2d2-tester/internal/job"
	"github.com/oneee-playground/r2d2-tester/internal/work/storage"
)

func main() {
	taskIDString := flag.String("taskID", "", "")
	sectionIDString := flag.String("sectionID", "", "")
	sectionType := flag.String("type", string(job.TypeScenario), "section type")
	location := flag.String("loc", "./", "storage location directory")

	flag.Parse()

	if *taskIDString == "" || *sectionIDString == "" {
		log.Fatal("argument is not enough")
	}

	taskID, err := uuid.Parse(*taskIDString)
	if err != nil {
		log.Fatal("taskID is malformed", err)
	}

	sectionID, err := uuid.Parse(*sectionIDString)
	if err != nil {
		log.Fatal("sectionID is malformed", err)
	}

	storage := storage.NewFSStorage(*location)
	templates, err := storage.FetchTemplates(context.Background(), taskID, sectionID)
	if err != nil {
		log.Fatal("fetching tempaltes", err)
	}

	cnt := 1
	if *sectionType == string(job.TypeLoad) {
		cnt = -3
	}

	stream, errchan := storage.Stream(context.Background(), taskID, sectionID)

	for cnt != 0 {
		select {
		case err := <-errchan:
			log.Fatal(err)
		case work, ok := <-stream:
			if !ok {
				break
			}
			input, _ := json.Marshal(work.Input)
			fmt.Printf("input: %s\n", input)

			var expected []byte
			if len(work.TemplateId) > 0 {
				expected, _ = json.Marshal(templates[uuid.UUID(work.TemplateId)].SchemaTable)
			} else {
				expected, _ = json.Marshal(work.ExpectedValue)
			}

			fmt.Printf("expected: %s\n", expected)
		}
		cnt++
	}
}
