package crdtverse

import (
	"testing"
	pv "github.com/pilinsin/p2p-verse"
)
func TestSignatureStore(t *testing.T) {
	BaseTestSignatureStore(t, pv.SampleHost)
}
