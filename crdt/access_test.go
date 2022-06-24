package crdtverse

import (
	"testing"

	pv "github.com/pilinsin/p2p-verse"
)

func TestAccessController(t *testing.T) {
	BaseTestAccessController(t, pv.SampleHost)
}
