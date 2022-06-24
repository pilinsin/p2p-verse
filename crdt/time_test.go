package crdtverse

import (
	"testing"

	pv "github.com/pilinsin/p2p-verse"
)

func TestTimeLimit(t *testing.T) {
	BaseTestTimeLimit(t, pv.SampleHost)
}
