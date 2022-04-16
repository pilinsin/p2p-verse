package crdtverse

import (
	"testing"
	pv "github.com/pilinsin/p2p-verse"
)

func TestUpdatableSignatureStore(t *testing.T) {
	BaseTestUpdatableSignatureStore(t, pv.SampleHost)
}