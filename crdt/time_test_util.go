package crdtverse

import (
	"os"
	"testing"
	"time"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	pv "github.com/pilinsin/p2p-verse"
)

func BaseTestTimeLimit(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)

	priv, pub, _ := p2pcrypto.GenerateEd25519Key(nil)
	begin := time.Now()
	end := begin.Add(time.Second * 30)
	opts0 := &StoreOpts{Priv: priv, Pub: pub, TimeLimit: end}
	db0 := newStore(t, hGen, "tc/ta", "us", "updatableSignature", baiStr, opts0)
	pid := PubKeyToStr(pub)
	db0 = newAccessStore(t, db0, pid)
	t.Log("db0 generated")

	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("put done")
	time.Sleep(time.Minute)

	db1 := newStore(t, hGen, "tc/tb", db0.Address(), "updatableSignature", baiStr)
	t.Log("db1 generated")

	assertError(t, !db1.isInTime(), "db1.inTime must be false")
	mustBeEmpty := "db1 is timeout, so db1 can't get any data"
	rs, err := db1.Query()
	checkError(t, err)
	resList, err := rs.Rest()
	checkError(t, err)
	assertError(t, len(resList) == 0 || err != nil, mustBeEmpty)
	v10, err := db1.Get(PubKeyToStr(opts0.Pub) + "/aaa")
	assertError(t, v10 == nil || err != nil, mustBeEmpty)

	db0.Close()
	db1.Close()
	time.Sleep(time.Second)
	os.RemoveAll("tc")
	t.Log("finished")
}
