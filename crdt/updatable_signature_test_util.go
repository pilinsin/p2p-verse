package crdtverse

import (
	"os"
	"testing"
	"time"

	query "github.com/ipfs/go-datastore/query"
	pv "github.com/pilinsin/p2p-verse"
)

func BaseTestUpdatableSignatureStore(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)

	opts0 := &StoreOpts{}
	db0 := newStore(t, hGen, "us/sa", "us", "updatableSignature", baiStr, opts0)
	t.Log("db0 generated")

	db1 := newStore(t, hGen, "us/sb", db0.Address(), "updatableSignature", baiStr)
	t.Log("db1 generated")

	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("put done")
	time.Sleep(time.Second * 10)

	v10, err := db1.Get(PubKeyToStr(opts0.Pub) + "/aaa")
	checkError(t, err)
	t.Log(string(v10))
	ok, err := db1.Has(PubKeyToStr(opts0.Pub) + "/aaa")
	t.Log(ok, err)

	rs1, err := db1.Query(query.Query{
		Filters: []query.Filter{KeyExistFilter{"aaa"}},
		Limit:   1,
	})
	checkError(t, err)
	for res := range rs1.Next() {
		t.Log(string(res.Value))
	}

	checkError(t, db0.Put("aaa", []byte("meow meow 2 ^.^")))
	time.Sleep(time.Second * 10)

	v12, err := db1.Get(PubKeyToStr(opts0.Pub) + "/aaa")
	checkError(t, err)
	t.Log(string(v12))

	rs12, err := db1.Query(query.Query{
		Filters: []query.Filter{KeyExistFilter{"aaa"}},
	})
	checkError(t, err)
	for res := range rs12.Next() {
		t.Log(string(res.Value))
	}

	db0.Close()
	db1.Close()
	time.Sleep(time.Second)
	os.RemoveAll("us")
	t.Log("finished")
}
