package test

import (
	"testing"

	peer "github.com/libp2p/go-libp2p-core/peer"

	pv "github.com/pilinsin/p2p-verse"
	crdt "github.com/pilinsin/p2p-verse/crdt"
	ipfs "github.com/pilinsin/p2p-verse/ipfs"
	pubsub "github.com/pilinsin/p2p-verse/pubsub"
)

func checkError(t *testing.T, err error, args ...interface{}) {
	if err != nil {
		args0 := make([]interface{}, len(args)+1)
		args0[0] = err
		copy(args0[1:], args)

		t.Fatal(args0...)
	}
}
func assertError(t *testing.T, cond bool, mess ...interface{}) {
	if !cond {
		t.Fatal(mess...)
	}
}

func bootstrapsToBAddrs(bs []pv.IBootstrap) []peer.AddrInfo {
	bAddrs := make([]peer.AddrInfo, len(bs))
	for idx, b := range bs {
		bAddrs[idx] = b.AddrInfo()
	}
	return bAddrs
}

func testBootstrap(t *testing.T) {
	bs := make([]pv.IBootstrap, 0)
	for i := 0; i < 3; i++ {
		b, err := pv.NewBootstrap(pv.SampleHost, bootstrapsToBAddrs(bs)...)
		checkError(t, err)
		bs = append(bs, b)
		defer b.Close()
		t.Log("bootstrap", i, "ID:", b.AddrInfo().ID, "Peers:", b.ConnectedPeers())
		if i > 0 {
			assertError(t, len(b.ConnectedPeers()) > 0, "failed to connect to other bootstraps")
		}
	}
	bAddrs := bootstrapsToBAddrs(bs)

	is, err := ipfs.NewIpfsStore(pv.SampleHost, "ipfs", true, bAddrs...)
	checkError(t, err)

	c, err := is.Add([]byte("meow meow ^.^"))
	checkError(t, err)
	t.Log("add,", c)
	is.Close()

	is, err = ipfs.NewIpfsStore(pv.SampleHost, "ipfs", false, bAddrs...)
	checkError(t, err)
	defer is.Close()

	is2, err := ipfs.NewIpfsStore(pv.SampleHost, "ipfs2", false, bAddrs...)
	checkError(t, err)
	defer is2.Close()

	v, err := is2.Get(c)
	checkError(t, err)
	t.Log("get,", string(v))

	t.Log("finished")
}

func TestP2pVerse(t *testing.T) {
	t.Log("===== bootstrap =====")
	testBootstrap(t)
	t.Log("===== pubsub =====")
	pubsub.BaseTestPubSub(t, pv.SampleHost)
	t.Log("===== ipfs =====")
	ipfs.BaseTestIpfs(t, pv.SampleHost)
	t.Log("===== log =====")
	crdt.BaseTestLogStore(t, pv.SampleHost)
	t.Log("===== hash =====")
	crdt.BaseTestHashStore(t, pv.SampleHost)
	t.Log("===== signature =====")
	crdt.BaseTestSignatureStore(t, pv.SampleHost)
	t.Log("===== updatablesignature =====")
	crdt.BaseTestUpdatableSignatureStore(t, pv.SampleHost)
	t.Log("===== access =====")
	crdt.BaseTestAccessController(t, pv.SampleHost)
	t.Log("===== time =====")
	crdt.BaseTestTimeLimit(t, pv.SampleHost)

}
