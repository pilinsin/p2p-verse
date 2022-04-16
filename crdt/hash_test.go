package crdtverse

import (
	"testing"
	pv "github.com/pilinsin/p2p-verse"
)

func TestHashStore(t *testing.T) {
	BaseTestHashStore(t, pv.SampleHost)
}
