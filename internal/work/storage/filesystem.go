package storage

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	protofmt "github.com/oneee-playground/r2d2-tester/internal/util/proto"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const (
	_filepathWorkPrefix     = "work"
	_filepathTemplatePrefix = "tmpl"
)

type FSStorage struct {
	root string
}

var _ work.Storage = (*FSStorage)(nil)

func NewFSStorage(root string) *FSStorage {
	return &FSStorage{root: root}
}

func (s *FSStorage) FetchTemplates(ctx context.Context, taskID uuid.UUID, sectionID uuid.UUID) (templates map[uuid.UUID]*work.Template, err error) {
	path := filepath.Join(s.root, taskID.String(), sectionID.String(), _filepathTemplatePrefix)

	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "opening template path")
	}
	defer file.Close()

	dec := protofmt.NewDecoder(bufio.NewReader(file))

	templates = make(map[uuid.UUID]*work.Template)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		dst := new(work.Template)

		err := dec.Decode(dst)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "decoding template")
		}

		templateID, err := uuid.FromBytes(dst.Id)
		if err != nil {
			return nil, errors.Wrap(err, "parsing uuid for template")
		}

		templates[templateID] = dst
	}

	return templates, nil
}

func (s *FSStorage) Stream(ctx context.Context, taskID uuid.UUID, sectionID uuid.UUID) (<-chan *work.Work, <-chan error) {
	// TODO: Specify buffer size as proper one.
	stream := make(chan *work.Work)
	errchan := make(chan error, 1)

	go func() {
		defer close(stream)

		path := filepath.Join(s.root, taskID.String(), sectionID.String(), _filepathWorkPrefix)

		file, err := os.Open(path)
		if err != nil {
			errchan <- errors.Wrap(err, "opening work path")
			return
		}
		defer file.Close()

		dec := protofmt.NewDecoder(bufio.NewReader(file))
		for {
			dst := new(work.Work)

			err := dec.Decode(dst)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				errchan <- errors.Wrap(err, "decoding work")
				return
			}

			select {
			case <-ctx.Done():
				errchan <- errors.Wrap(err, "streaming templates")
				return
			case stream <- dst:
			}
		}
	}()

	return stream, errchan
}

func (s *FSStorage) InsertWork(ctx context.Context, taskID uuid.UUID, sectionID uuid.UUID, work *work.Work) error {
	path := filepath.Join(s.root, taskID.String(), sectionID.String(), _filepathWorkPrefix)
	return s.insertRaw(path, work)
}

func (s *FSStorage) InsertTemplate(ctx context.Context, taskID uuid.UUID, sectionID uuid.UUID, template *work.Template) error {
	path := filepath.Join(s.root, taskID.String(), sectionID.String(), _filepathTemplatePrefix)
	return s.insertRaw(path, template)
}

func (s *FSStorage) insertRaw(path string, m proto.Message) error {
	if err := os.MkdirAll(filepath.Dir(path), 0744); err != nil {
		return errors.Wrap(err, "mkdir all")
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.Wrap(err, "opening file")
	}
	defer file.Close()

	b, err := protofmt.MarshalWithSize(m)
	if err != nil {
		return errors.Wrap(err, "marshaling work")
	}

	_, err = file.Write(b)
	if err != nil {
		return errors.Wrap(err, "writing to file")
	}

	return nil
}
