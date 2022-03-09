package pubsub

import(
	"testing"

	"fmt"
	"time"
	"context"
	pv "github.com/pilinsin/p2p-verse"
	"github.com/pilinsin/go-libp2p-i2p"
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
func TestPubSub(t *testing.T){
	N := 10

	b, err := pv.SampleHost()
	checkError(t, err)
	bstrp, err := pv.NewBootstrap(b)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	<-time.Tick(time.Second*5)

	h0, err0 := pv.SampleHost()
	checkError(t, err0)
	ps0, err01 := NewPubSub(context.Background(), h0, bAddrInfo)
	checkError(t, err01)
	tpc0, err02 := ps0.JoinTopic("test topic")
	checkError(t, err02)
	go func(){
		defer tpc0.Close()
		itr := 0
		for{
			mess, err := tpc0.GetAll()
			t.Log(itr, err)
			if err == nil && len(mess)>0{
				itr += len(mess)
				for _, mes := range mess{
					t.Log(string(mes.Data))
				}
			}
	
			if itr >= N{return}
		}
	}()

	<-time.Tick(time.Second*5)

	h1, err1 := pv.SampleHost()
	checkError(t, err1)
	ps1, err11 := NewPubSub(context.Background(), h1, bAddrInfo)
	checkError(t, err11)

	t.Log("h0 id             : ", h0.ID())
	t.Log("h0 address        : ", h0.Addrs())
	t.Log("h0 connected peers:", h0.Network().Peers())
	t.Log("h1 id             : ", h1.ID())
	t.Log("h0 address        :", h1.Addrs())
	t.Log("h1 connected peers:", h1.Network().Peers())

	tpc1, err12 := ps1.JoinTopic("test topic")
	checkError(t, err12)
	defer tpc1.Close()
	t.Log("topic peers list  :", tpc1.ListPeers())
	for i := 0; i<N; i++{
		tpc1.Publish([]byte(fmt.Sprintln("message ", i)))
	}


	<-time.Tick(10*time.Second)
	t.Log("finished")
}

