package pubsub

import (
	pv "github.com/pilinsin/p2p-verse"
	"testing"
)

func TestPubSub(t *testing.T) {
	BaseTestPubSub(t, pv.SampleHost)
}
