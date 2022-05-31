package ipfsverse

import (
	pv "github.com/pilinsin/p2p-verse"
	"testing"
)

func TestIpfs(t *testing.T) {
	BaseTestIpfs(t, pv.SampleHost)
}
