package crdtverse

import (
	pv "github.com/pilinsin/p2p-verse"
	"testing"
)

func TestAccessController(t *testing.T) {
	BaseTestAccessController(t, pv.SampleHost)
}
