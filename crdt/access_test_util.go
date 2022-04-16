package crdtverse

import (
	"testing"

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
	pid := PubKeyToStr(pub)
	ac := newAccessController(t, hGen, "ac/c", "ac", baiStr, pid)
	opts0 := &StoreOpts{Priv: priv, Pub: pub, Ac: ac}
	db0 := newStore(t, hGen, "ac/aa", "us", "updatableSignature", baiStr, opts0)
	defer db0.Close()
	t.Log("db0 generated")
	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("put done")

	db1 := loadStore(t, hGen, "ac/ab", db0.Address(), "updatableSignature", baiStr)
	defer db1.Close()
	t.Log("db1 generated")

	checkError(t, db1.Sync())
	v10, err := db1.Get(PubKeyToStr(opts0.Pub) + "/aaa")
	checkError(t, err)
	t.Log("db1.Get:", string(v10))

	t.Log("finished")
}