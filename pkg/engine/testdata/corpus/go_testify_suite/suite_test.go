package sample_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type MySuite struct {
	suite.Suite
}

func (s *MySuite) TestAdd() {
	s.Equal(4, 2+2)
}

func TestMySuite(t *testing.T) {
	suite.Run(t, new(MySuite))
}
