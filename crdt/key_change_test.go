package crdtverse

import (
	"os"
	"testing"
	"time"

	query "github.com/ipfs/go-datastore/query"
	pv "github.com/pilinsin/p2p-verse"
)

func TestKeyChange(t *testing.T) {
	testKeyChange(t, pv.SampleHost)
}

func testKeyChange(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)

	opts0 := &StoreOpts{}
	tmp := newStore(t, hGen, "ss/sa", "sg", "signature", baiStr, opts0)
	db0 := tmp.(ISignatureStore)
	t.Log("db0 generated")
	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))

	priv, pub, err := generateKeyPair()
	checkError(t, err)
	db0.ResetKeyPair(priv, pub)
	checkError(t, db0.Put("aaa", []byte("meow meow 2 ^.^")))

	rs, err := db0.Query(query.Query{
		Filters: []query.Filter{KeyExistFilter{"aaa"}},
	})
	checkError(t, err)
	ress, err := rs.Rest()
	checkError(t, err)
	assertError(t, len(ress) == 2, "the number of records must be 2, but now is", len(ress))

	db0.Close()
	time.Sleep(time.Second * 30)
	os.RemoveAll("ss")
	t.Log("finished")
}
