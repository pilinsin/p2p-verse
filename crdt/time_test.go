package crdtverse

import (
	"testing"
	pv "github.com/pilinsin/p2p-verse"
)

func TestTimeController(t *testing.T){
	BaseTestTimeController(t, pv.SampleHost)
}
