package crdtverse

import (
	pv "github.com/pilinsin/p2p-verse"
	"testing"
)

func TestLogStore(t *testing.T) {
	BaseTestLogStore(t, pv.SampleHost)
}
