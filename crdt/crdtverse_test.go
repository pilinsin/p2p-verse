package crdtverse

import(
	"testing"
	"time"

	pv "github.com/pilinsin/p2p-verse"
)

func checkError(t *testing.T, err error, args ...interface{}){
	if err != nil{
		args0 := make([]interface{}, len(args)+1)
		args0[0] = err
		copy(args0[1:], args)

		t.Fatal(args0...)
	}
}

func TestCRDT(t *testing.T){
	b, err := pv.SampleHost()
	checkError(t, err)
	bstrp, err := pv.NewBootstrap(b)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	h0, err := pv.SampleHost()
	checkError(t, err)
	priv0 := h0.Peerstore().PrivKey(h0.ID())
	db0, err := NewVerse("a", false, h0, bAddrInfo).NewSignatureStore("testDB", priv0)
	checkError(t, err)
	defer db0.Close()
	checkError(t, db0.Put([]byte("meow meow ^.^")))

	h1, err := pv.SampleHost()
	checkError(t, err)
	priv1 := h1.Peerstore().PrivKey(h1.ID())
	db1, err := NewVerse("b", false, h1, bAddrInfo).NewSignatureStore("testDB", priv1)
	checkError(t, err)
	defer db1.Close()

	t.Log("h0 id             : ", h0.ID())
	t.Log("h0 connected peers:", h0.Network().Peers())
	t.Log("h1 id             : ", h1.ID())
	t.Log("h1 connected peers:", h1.Network().Peers())

	checkError(t, db1.Sync())
	v, err := db1.Get(h0.ID().Pretty())
	checkError(t, err)
	t.Log(string(v))

	checkError(t, db0.Put([]byte("meow meow 2 ^.^")))

	<-time.Tick(time.Minute)

	checkError(t, db1.Sync())
	v2, err := db1.Get(h0.ID().Pretty())
	checkError(t, err)
	t.Log(string(v2))

	rs, err := db1.Query()
	checkError(t, err)
	defer rs.Close()
	for res := range rs.Next(){
		sd, err := UnmarshalSignedData(res.Value)
		if err != nil{continue}
		t.Log(string(sd.Value))
	}
}