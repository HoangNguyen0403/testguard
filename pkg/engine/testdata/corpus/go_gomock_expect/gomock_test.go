package sample_test

import (
	"testing"

	"github.com/golang/mock/gomock"
)

type UserGetter interface {
	GetUser() string
}

func TestGomockExpect(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockUserGetter(ctrl)
	mock.EXPECT().GetUser().Return("alice")
	if mock.GetUser() != "alice" {
		t.Fail()
	}
}
