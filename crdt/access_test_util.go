package crdtverse

import (
	"os"
	"testing"
	"time"

	pv "github.com/pilinsin/p2p-verse"
)

func testHashAccess(t *testing.T, hGen pv.HostGenerator, baiStr string) {
	opts := &StoreOpts{}
	db0 := newStore(t, hGen, "ahs/ha", "hs", "hash", baiStr, opts)
	db0 = newAccessStore(t, db0, db0.(*hashStore).putKey("aaa"))
	t.Log("db0 generated")

	db1 := newStore(t, hGen, "ahs/hb", db0.Address(), "hash", baiStr)
	t.Log("db1 generated")

	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	time.Sleep(time.Second * 10)

	v10, err := db1.Get("aaa")
	checkError(t, err)
	t.Log(string(v10))
	ok, err := db1.Has("aaa")
	t.Log(ok, err)

	assertError(t, db0.Put("aaa", []byte("meow meow 2 ^.^")) == ErrAlreadyExist, "2nd Put must be fail")
	time.Sleep(time.Second * 10)

	v12, err := db1.Get("aaa")
	checkError(t, err)
	t.Log(string(v12))

	rs, err := db1.Query()
	checkError(t, err)
	resList, err := rs.Rest()
	checkError(t, err)
	assertError(t, len(resList) > 0, "valid data must be exist")

	db0.Close()
	db1.Close()
	time.Sleep(time.Second)
	os.RemoveAll("ahs")
}

func testSignatureAccess(t *testing.T, hGen pv.HostGenerator, baiStr string) {
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

	rs, err := db1.Query()
	checkError(t, err)
	resList, err := rs.Rest()
	checkError(t, err)
	assertError(t, len(resList) > 0, "valid data must be exist")

	db0.Close()
	db1.Close()
	time.Sleep(time.Second)
	os.RemoveAll("ac")
}

func BaseTestAccessController(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)

	t.Log("===== hash access =====")
	testHashAccess(t, hGen, baiStr)
	t.Log("===== signature access =====")
	testSignatureAccess(t, hGen, baiStr)
	t.Log("finished")
}
