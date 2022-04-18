package crdtverse

import (
	"testing"
	"time"

	query "github.com/ipfs/go-datastore/query"
	pv "github.com/pilinsin/p2p-verse"
)

func BaseTestSignatureStore(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)

	opts0 := &StoreOpts{}
	db0 := newStore(t, hGen, "ss/sa", "sg", "signature", baiStr, opts0)
	defer db0.Close()
	t.Log("db0 generated")

	db1 := loadStore(t, hGen, "ss/sb", db0.Address(), "signature", baiStr)
	defer db1.Close()
	t.Log("db1 generated")

	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("put done")
	time.Sleep(time.Second*30)

	checkError(t, db1.Sync())
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

	checkError(t, db1.Sync())
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

	t.Log("finished")
}
