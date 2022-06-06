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
	pid := PubKeyToStr(pub)
	ac := newAccessController(t, hGen, "tc/c", "ac", baiStr, pid)
	begin := time.Now()
	end := begin.Add(time.Second*30)
	opts0 := &StoreOpts{Priv: priv, Pub: pub, Ac: ac, TimeLimit: end}
	db0 := newStore(t, hGen, "tc/ta", "us", "updatableSignature", baiStr, opts0)
	t.Log("db0 generated")

	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("put done")

	time.Sleep(time.Minute)
	
	db1 := loadStore(t, hGen, "tc/tb", db0.Address(), "updatableSignature", baiStr)
	t.Log("db1 generated")


	t.Log(db1.Sync())
	rs, err := db1.Query()
	t.Log("db1.Query:", err)
	for res := range rs.Next(){
		t.Log(res.Key, res.Value)
	}
	v10, err := db1.Get(PubKeyToStr(opts0.Pub) + "/aaa")
	t.Log("db1.Get:", err)
	assertError(t, err != nil, "db1.Get must return error in the test case", "val:", v10)

	db0.Close()
	db1.Close()
	time.Sleep(time.Second)
	os.RemoveAll("tc")
	t.Log("finished")
}
