package crdtverse

import (
	pv "github.com/pilinsin/p2p-verse"
	"testing"
)

func TestTimeController(t *testing.T) {
	BaseTestTimeController(t, pv.SampleHost)
}
