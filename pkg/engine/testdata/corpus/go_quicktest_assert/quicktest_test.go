package sample_test

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestQuicktest(t *testing.T) {
	c := qt.New(t)
	c.Assert(1+2, qt.Equals, 3)
}
