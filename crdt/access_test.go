package crdtverse

import(
	"testing"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	pv "github.com/pilinsin/p2p-verse"
)


func TestAccessController(t *testing.T){
	b, err := pv.SampleHost()
	checkError(t, err)
	bstrp, err := pv.NewBootstrap(b)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)
	baiStr := pv.AddrInfoToString(bAddrInfo)


	priv, pub, _ := p2pcrypto.GenerateEd25519Key(nil)
	pid := PubKeyToStr(pub)
	ac := newAccessController(t, "ac/c", "ac", baiStr, pid)
	opts0 := &StoreOpts{Priv: priv, Pub: pub, Ac: ac}
	db0 := newStore(t, "ac/aa", "us", "updatableSignature", baiStr, opts0)
	defer db0.Close()
	t.Log("db0 generated")
	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("put done")


	db1 := loadStore(t, "ac/ab", db0.Address(), "updatableSignature", baiStr)
	defer db1.Close()
	t.Log("db1 generated")

	checkError(t, db1.Sync())
	v10, err := db1.Get(PubKeyToStr(opts0.Pub)+"/aaa")
	checkError(t, err)
	t.Log("db1.Get:", string(v10))


	t.Log("finished")
}
