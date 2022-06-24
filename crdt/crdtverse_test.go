package crdtverse

import (
	"testing"

	pv "github.com/pilinsin/p2p-verse"
)

func TestLogStore(t *testing.T) {
	BaseTestLogStore(t, pv.SampleHost)
}
