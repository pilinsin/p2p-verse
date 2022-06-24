package crdtverse

import (
	"os"
	"testing"
	"time"

	pv "github.com/pilinsin/p2p-verse"
)

func BaseTestAccessController(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)

	priv, pub, _ := generateKeyPair()
	opts0 := &StoreOpts{Priv: priv, Pub: pub}
	db0 := newStore(t, hGen, "ac/aa", "us", "updatableSignature", baiStr, opts0)
	pid := PubKeyToStr(pub)
	db0 = newAccessStore(t, db0, pid)
	t.Log("db0 generated")

	db1 := newStore(t, hGen, "ac/ab", db0.Address(), "updatableSignature", baiStr)
	t.Log("db1 generated")

	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("put done")
	time.Sleep(time.Second * 10)

	v10, err := db1.Get(PubKeyToStr(opts0.Pub) + "/aaa")
	checkError(t, err)
	t.Log("db1.Get:", string(v10))

	db0.Close()
	db1.Close()
	time.Sleep(time.Second)
	os.RemoveAll("ac")
	t.Log("finished")
}
