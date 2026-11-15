package sample_test

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestGomega(t *testing.T) {
	g := NewWithT(t)
	g.Expect(1 + 2).To(Equal(3))
}
