package dberse

import(
	"testing"

	"time"
	"context"
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

//go test -test.v=true .
func TestDB(t *testing.T){
	b, err := pv.SampleHost()
	checkError(t, err)
	bstrp, err := pv.NewBootstrap(b)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	//<-time.Tick(time.Second*5)

	h0, err0 := pv.SampleHost()
	checkError(t, err0)
	d0 := NewDBerse(context.Background(), h0, bAddrInfo)
	priv0 := h0.Peerstore().PrivKey(h0.ID())
	db_0, err_01 := d0.NewSignatureStore("testDB", priv0, priv0.GetPublic())
	checkError(t, err_01)
	db0, err01 := d0.WithCatalog("catalog", db_0)
	checkError(t, err01)
	defer db0.Close()

	//<-time.Tick(time.Second*5)
	//cidKey, _ := MakeCidKey([]byte("meow meow ^o^"))
	//t.Log("cidKey:", cidKey)
	err02 := db0.Put([]byte("meow meow ^o^"))
	t.Log("db0.Put", err02)
	checkError(t, err02)
	err022 := db0.Put([]byte("meow meow2 ^o^"))
	t.Log("db0.Put", err022)
	checkError(t, err022)

	//<-time.Tick(time.Second*5)

	h1, err1 := pv.SampleHost()
	checkError(t, err1)
	d1 := NewDBerse(context.Background(), h1, bAddrInfo)
	priv1 := h1.Peerstore().PrivKey(h1.ID())
	db_1, err_11 := d1.NewSignatureStore("testDB", priv1, priv1.GetPublic())
	checkError(t, err_11)
	db1, err11 := d1.WithCatalog("catalog", db_1)
	checkError(t, err11)
	defer db1.Close()

	t.Log("h0 id             : ", h0.ID())
	t.Log("h0 address        : ", h0.Addrs())
	t.Log("h0 connected peers:", h0.Network().Peers())
	t.Log("h1 id             : ", h1.ID())
	t.Log("h1 address        :", h1.Addrs())
	t.Log("h1 connected peers:", h1.Network().Peers())

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	v, err12 := db1.GetWait(ctx, h0.ID().Pretty())
	checkError(t, err12)
	t.Log("db1.Get", string(v))

	err13 := db1.Put([]byte("meow meow meow ^.^ !!!"))
	checkError(t, err13)
	t.Log("db1.Put", err13)

	v02, err03 := db0.GetWait(ctx, h0.ID().Pretty())
	checkError(t, err03)
	t.Log("db0.Get", string(v02))

	for key := range db0.GetKeys(){
		t.Log("key:", key)
	}
	//<-time.Tick(10*time.Second)
	t.Log("finished")
}

