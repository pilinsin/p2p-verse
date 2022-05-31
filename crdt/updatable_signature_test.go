package crdtverse

import (
	pv "github.com/pilinsin/p2p-verse"
	"testing"
)

func TestUpdatableSignatureStore(t *testing.T) {
	BaseTestUpdatableSignatureStore(t, pv.SampleHost)
}
