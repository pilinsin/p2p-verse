package crdtverse

import (
	pv "github.com/pilinsin/p2p-verse"
	"testing"
)

func TestTimeLimit(t *testing.T) {
	BaseTestTimeLimit(t, pv.SampleHost)
}
