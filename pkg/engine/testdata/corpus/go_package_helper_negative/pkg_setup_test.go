package sample_test

import "testing"

func TestExternalSetupOnly(t *testing.T) {
	setupDB(t)
	Do()
}
