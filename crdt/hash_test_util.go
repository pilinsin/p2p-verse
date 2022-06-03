package crdtverse

import (
	"os"
	"testing"
	"time"

	pv "github.com/pilinsin/p2p-verse"
)

func BaseTestHashStore(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)

	opts := &StoreOpts{}
	db0 := newStore(t, hGen, "hs/ha", "hs", "hash", baiStr, opts)
	t.Log("db0 generated")

	db1 := loadStore(t, hGen, "hs/hb", db0.Address(), "hash", baiStr, opts)
	t.Log("db1 generated")

	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	time.Sleep(time.Second * 10)

	v10, err := db1.Get("aaa")
	checkError(t, err)
	t.Log(string(v10))
	ok, err := db1.Has("aaa")
	t.Log(ok, err)

	checkError(t, db0.Put("aaa", []byte("meow meow 2 ^.^")))
	time.Sleep(time.Second * 10)

	v12, err := db1.Get("aaa")
	checkError(t, err)
	t.Log(string(v12))

	db0.Close()
	db1.Close()
	time.Sleep(time.Second)
	os.RemoveAll("hs")
	t.Log("finished")
}
