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

	<-time.Tick(time.Second*5)

	h0, err0 := pv.SampleHost()
	checkError(t, err0)
	db0, err01 := NewDB(context.Background(), h0, Const, bAddrInfo)
	checkError(t, err01)
	defer db0.Close()

	<-time.Tick(time.Second*5)

	err02 := db0.Put("testkey", []byte("meow meow ^o^"))
	t.Log("db0.Put", err02)
	checkError(t, err02)
	err022 := db0.Put("testkey", []byte("meow meow2 ^o^"))
	t.Log("db0.Put", err022)
	checkError(t, err022)

	<-time.Tick(time.Second*5)

	h1, err1 := pv.SampleHost()
	checkError(t, err1)
	db1, err11 := NewDB(context.Background(), h1, Simple, bAddrInfo)
	checkError(t, err11)
	defer db1.Close()

	t.Log("h0 id             : ", h0.ID())
	t.Log("h0 address        : ", h0.Addrs())
	t.Log("h0 connected peers:", h0.Network().Peers())
	t.Log("h1 id             : ", h1.ID())
	t.Log("h0 address        :", h1.Addrs())
	t.Log("h1 connected peers:", h1.Network().Peers())

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	v, err12 := db1.GetWait(ctx, "testkey")
	checkError(t, err12)
	t.Log("db1.Get", string(v))

	err13 := db1.Put("testkey", []byte("meow meow meow ^.^ !!!"))
	checkError(t, err13)
	t.Log("db1.Put", err13)

	v02, err03 := db0.GetWait(ctx, "testkey")
	checkError(t, err03)
	t.Log("db0.Get", string(v02))

	<-time.Tick(10*time.Second)
	t.Log("finished")
}

