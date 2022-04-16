package ipfsverse

import (
	"testing"
	pv "github.com/pilinsin/p2p-verse"
)

func TestIpfs(t *testing.T) {
	BaseTestIpfs(t, pv.SampleHost)
}
