package pubsub

import (
	"testing"

	"fmt"
	pv "github.com/pilinsin/p2p-verse"
	"time"
)

func checkError(t *testing.T, err error, args ...interface{}) {
	if err != nil {
		args0 := make([]interface{}, len(args)+1)
		args0[0] = err
		copy(args0[1:], args)

		t.Fatal(args0...)
	}
}

//go test -test.v=true .
func BaseTestPubSub(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	<-time.Tick(time.Second * 5)

	N := 10

	ps0, err01 := NewPubSub(hGen, bAddrInfo)
	checkError(t, err01)
	defer ps0.Close()
	tpc0, err02 := ps0.JoinTopic("test topic")
	checkError(t, err02)
	go func() {
		defer tpc0.Close()
		itr := 0
		for {
			mess, err := tpc0.GetAll()
			t.Log(itr, err)
			if err == nil && len(mess) > 0 {
				itr += len(mess)
				for _, mes := range mess {
					t.Log(string(mes.Data))
				}
			}

			if itr >= N {
				return
			}
		}
	}()

	<-time.Tick(time.Second * 10)

	ps1, err11 := NewPubSub(hGen, bAddrInfo)
	checkError(t, err11)
	defer ps1.Close()

	tpc1, err12 := ps1.JoinTopic("test topic")
	checkError(t, err12)
	defer tpc1.Close()
	t.Log("topic peers list  :", tpc1.ListPeers())
	for i := 0; i < N; i++ {
		tpc1.Publish([]byte(fmt.Sprintln("message ", i)))
	}

	<-time.Tick(time.Second * 10)
	t.Log("finished")
}
