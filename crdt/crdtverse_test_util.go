package crdtverse

import (
	"testing"
	"time"
	"os"

	query "github.com/ipfs/go-datastore/query"
	pv "github.com/pilinsin/p2p-verse"
)

func BaseTestLogStore(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)

	db0 := newStore(t, hGen, "ls/la", "lg", "log", baiStr)
	t.Log("db0 generated")

	db1 := loadStore(t, hGen, "ls/lb", db0.Address(), "log", baiStr)
	t.Log("db1 generated")

	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	time.Sleep(time.Second*30)

	checkError(t, db1.Sync())
	v10, err := db1.Get("aaa")
	checkError(t, err)
	t.Log(string(v10))
	ok, err := db1.Has("aaa")
	t.Log(ok, err)

	rs1, err := db1.Query(query.Query{
		Filters: []query.Filter{KeyMatchFilter{"aaa"}},
		Limit:   1,
	})
	checkError(t, err)
	for res := range rs1.Next() {
		t.Log(string(res.Value))
	}

	checkError(t, db0.Sync())
	checkError(t, db0.Put("aaa", []byte("meow meow 2 ^.^")))
	time.Sleep(time.Second * 5)

	checkError(t, db1.Sync())
	v12, err := db1.Get("aaa")
	checkError(t, err)
	t.Log(string(v12))

	rs12, err := db1.Query(query.Query{
		Filters: []query.Filter{KeyMatchFilter{"aaa"}},
	})
	checkError(t, err)
	for res := range rs12.Next() {
		t.Log(string(res.Value))
	}

	db0.Close()
	db1.Close()
	time.Sleep(time.Second*30)
	os.RemoveAll("ls")
	t.Log("finished")
}
