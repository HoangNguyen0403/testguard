package sample

import (
	"example.test/mocks"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestMockInClosure(t *testing.T) {
	cases := []struct {
		name string
		mock func(m *mocks.MockRepo)
	}{
		{
			name: "success",
			mock: func(m *mocks.MockRepo) {
				m.EXPECT().Fetch().Return(2)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks.NewMockRepo(ctrl)
			tc.mock(m)
		})
	}
}
