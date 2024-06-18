package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oneee-playground/r2d2-tester/internal/util/proto"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/types/known/durationpb"
)

type FSStorageSuite struct {
	suite.Suite
	base    string
	storage *FSStorage
}

func TestFSStorageSuite(t *testing.T) {
	suite.Run(t, new(FSStorageSuite))
}

func (s *FSStorageSuite) SetupTest() {
	s.base = s.T().TempDir()
	s.storage = NewFSStorage(s.base)
}

func (s *FSStorageSuite) TestInsertWork() {
	w := &work.Work{
		Id:         uuid.Nil[:],
		TemplateId: uuid.Nil[:],
		Timeout:    durationpb.New(time.Hour),
	}

	err := s.storage.InsertWork(context.Background(), uuid.Nil, uuid.Nil, w)
	if !s.NoError(err) {
		return
	}

	path := filepath.Join(s.base, uuid.Nil.String(), uuid.Nil.String(), _filepathWorkPrefix)

	file, err := os.Open(path)
	if !s.NoError(err) {
		return
	}
	defer file.Close()

	dst := new(work.Work)
	err = proto.NewDecoder(file).Decode(dst)
	if !s.NoError(err) {
		return
	}

	s.Equal(w.Id, dst.Id)
	s.Equal(w.TemplateId, dst.TemplateId)
	s.Equal(w.Timeout.AsDuration(), dst.Timeout.AsDuration())
}

func (s *FSStorageSuite) TestInsertTemplate() {
	t := &work.Template{
		Id:          uuid.Nil[:],
		SchemaTable: nil,
	}

	err := s.storage.InsertTemplate(context.Background(), uuid.Nil, uuid.Nil, t)
	if !s.NoError(err) {
		return
	}

	path := filepath.Join(s.base, uuid.Nil.String(), uuid.Nil.String(), _filepathTemplatePrefix)

	file, err := os.Open(path)
	if !s.NoError(err) {
		return
	}
	defer file.Close()

	dst := new(work.Template)
	err = proto.NewDecoder(file).Decode(dst)
	if !s.NoError(err) {
		return
	}

	s.Equal(t.Id, dst.Id)
	s.Equal(t.SchemaTable, dst.SchemaTable)
}

func (s *FSStorageSuite) TestFetchTemplates() {
	t := &work.Template{
		Id:          uuid.Nil[:],
		SchemaTable: nil,
	}

	err := s.storage.InsertTemplate(context.Background(), uuid.Nil, uuid.Nil, t)
	s.Require().NoError(err)

	ts, err := s.storage.FetchTemplates(context.Background(), uuid.Nil, uuid.Nil)
	if !s.NoError(err) {
		return
	}

	s.Len(ts, 1)
	for _, tmpl := range ts {
		s.Equal(t.Id, tmpl.Id)
		s.Equal(t.SchemaTable, tmpl.SchemaTable)
	}
}

func (s *FSStorageSuite) TestStream() {
	defer goleak.VerifyNone(s.T())

	w := &work.Work{
		Id:         uuid.Nil[:],
		TemplateId: uuid.Nil[:],
		Timeout:    durationpb.New(time.Hour),
	}

	cnt := 10

	for i := 0; i < cnt; i++ {
		err := s.storage.InsertWork(context.Background(), uuid.Nil, uuid.Nil, w)
		s.Require().NoError(err)
	}

	stream, errchan := s.storage.Stream(context.Background(), uuid.Nil, uuid.Nil)

loop:
	for {
		select {
		case got, ok := <-stream:
			if !ok {
				break loop
			}
			s.Equal(w.Id, got.Id)
			s.Equal(w.TemplateId, got.TemplateId)
			s.Equal(w.Timeout.AsDuration(), got.Timeout.AsDuration())
			cnt--
		case err := <-errchan:
			s.Fail("err received from errchan", err)
			return
		}
	}

	s.Equal(cnt, 0)
}
