package crdtverse

import (
	pv "github.com/pilinsin/p2p-verse"
	"testing"
)

func TestHashStore(t *testing.T) {
	BaseTestHashStore(t, pv.SampleHost)
}
