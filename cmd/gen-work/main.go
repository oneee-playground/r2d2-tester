package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/oneee-playground/r2d2-tester/internal/work/storage"
	"github.com/ryanolee/go-chaff"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	num int

	storePath         string
	taskID, sectionID uuid.UUID

	bodySchema []byte
	bodyExact  []byte
	method     string
	path       string
	headers    map[string]string
	templateID []byte
	timeout    time.Duration
)

func processParameters() {
	var (
		_num        = flag.Int("n", 1, "number of generated schema")
		_schemaPath = flag.String("schema", "", "input schema file path")
		_bodyPath   = flag.String("body", "", "input body file path")
		_storePath  = flag.String("storepath", "", "storage root path")
		_taskID     = flag.String("taskID", "", "task id")
		_sectionID  = flag.String("sectionID", "", "section id")
		_method     = flag.String("method", "", "http method")
		_path       = flag.String("path", "", "http path")
		_headers    = flag.String("headers", "", "http headers. seperated with comma. (e.g. headers=key=value,key=value")
		_templateID = flag.String("templateID", "", "template id")
		_timeout    = flag.Duration("timeout", 100*time.Millisecond, "request timeout")
	)

	flag.Parse()

	num = *_num
	storePath = *_storePath
	method = *_method
	path = *_path
	timeout = *_timeout

	if *_schemaPath != "" {
		file, err := os.Open(*_schemaPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			log.Fatal(err)
		}

		bodySchema = b
	}

	if *_bodyPath != "" {
		file, err := os.Open(*_bodyPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			log.Fatal(err)
		}

		bodyExact = b
	}

	taskID = uuid.MustParse(*_taskID)
	sectionID = uuid.MustParse(*_sectionID)

	if *_templateID != "" {
		id := uuid.MustParse(*_templateID)
		templateID = id[:]
	}

	if *_headers != "" {
		kvPairs := strings.Split(*_headers, ",")
		headerMap := make(map[string]string, len(kvPairs))
		for _, pair := range kvPairs {
			k, v, found := strings.Cut(pair, "=")
			if !found {
				log.Fatal("malformed header")
			}

			headerMap[k] = v
		}

		headers = headerMap
	}
}

func main() {
	processParameters()

	var generator chaff.Generator
	if bodySchema != nil {
		gen, err := chaff.ParseSchema([]byte(bodySchema), &chaff.ParserOptions{})
		if err != nil {
			log.Fatal(err)
		}
		generator = gen
	}

	storage := storage.NewFSStorage(storePath)

	n := num
	for i := 0; i < n; i++ {
		var body []byte
		if generator != nil {
			result := generator.Generate(&chaff.GeneratorOptions{})
			b, _ := json.Marshal(result)
			body = b
		} else {
			body = []byte(bodyExact)
		}

		id := uuid.New()

		work := &work.Work{
			Id: id[:],
			Input: &work.Input{
				Method:  method,
				Path:    path,
				Headers: headers,
				Body:    body,
			},
			TemplateId: templateID,
			Timeout:    durationpb.New(timeout),
		}

		err := storage.InsertWork(context.Background(), taskID, sectionID, work)
		if err != nil {
			log.Fatal(err)
		}
	}
}
