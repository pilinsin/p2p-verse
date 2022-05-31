package crdtverse

import (
	pv "github.com/pilinsin/p2p-verse"
	"testing"
)

func TestSignatureStore(t *testing.T) {
	BaseTestSignatureStore(t, pv.SampleHost)
}
