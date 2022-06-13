package pubsub

import (
	"testing"
	pv "github.com/pilinsin/p2p-verse"
)

func TestPubSub(t *testing.T) {
	BaseTestPubSub(t, pv.SampleHost)
}
