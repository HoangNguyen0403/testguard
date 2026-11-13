package sample_test

import (
	"testing"

	"example.com/app/internal/testutil"
)

func TestCrossPkgHelper(t *testing.T) {
	testutil.AssertEqual(t, 3, 1+2)
}
