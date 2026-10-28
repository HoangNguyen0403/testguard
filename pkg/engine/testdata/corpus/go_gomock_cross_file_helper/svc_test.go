package sample

import (
	"testing"

	"github.com/golang/mock/gomock"
)

type Svc struct {
	Table string
}

func (s *Svc) Do() {}

func TestSvc(t *testing.T) {
	type tc struct {
		name   string
		doMock func(*Svc)
	}
	cases := []tc{{name: "ok", doMock: func(s *Svc) {
		m := shared.GetMockRepository(s.Table, s)
		m.EXPECT().First(gomock.Any()).Return(nil, nil)
	}}}
	ctrl := gomock.NewController(t)
	for _, c := range cases {
		srv := newMockService(ctrl)
		t.Run(c.name, func(t *testing.T) { c.doMock(srv); srv.Do() })
	}
}
