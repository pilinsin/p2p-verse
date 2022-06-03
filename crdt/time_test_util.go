package crdtverse

import (
	"os"
	"testing"
	"time"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	pv "github.com/pilinsin/p2p-verse"
)

func BaseTestTimeController(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)

	priv, pub, _ := p2pcrypto.GenerateEd25519Key(nil)
	pid := PubKeyToStr(pub)
	ac := newAccessController(t, hGen, "tc/c", "ac", baiStr, pid)
	begin := time.Now()
	end := begin.Add(time.Hour)
	tc := newTimeController(t, hGen, "tc/t", "tc", baiStr, begin, end)
	opts0 := &StoreOpts{Priv: priv, Pub: pub, Ac: ac, Tc: tc}
	db0 := newStore(t, hGen, "tc/ta", "us", "updatableSignature", baiStr, opts0)
	t.Log("db0 generated")

	db1 := loadStore(t, hGen, "tc/tb", db0.Address(), "updatableSignature", baiStr)
	t.Log("db1 generated")

	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("put done")

	time.Sleep(time.Second*30)

	v10, err := db1.Get(PubKeyToStr(opts0.Pub) + "/aaa")
	checkError(t, err)
	t.Log("db1.Get:", string(v10))

	db0.Close()
	db1.Close()
	time.Sleep(time.Second)
	os.RemoveAll("tc")
	t.Log("finished")
}
