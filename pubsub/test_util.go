package pubsub

import (
	"testing"

	"fmt"
	"time"

	pv "github.com/pilinsin/p2p-verse"
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
			if len(tpc0.ListPeers()) == 0 {
				continue
			}

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

	ps1, err11 := NewPubSub(hGen, bAddrInfo)
	checkError(t, err11)
	defer ps1.Close()
	tpc1, err12 := ps1.JoinTopic("test topic")
	checkError(t, err12)
	defer tpc1.Close()
	time.Sleep(time.Second * 2)

	t.Log("topic peers list  :", tpc1.ListPeers())
	for i := 0; i < N; i++ {
		err := tpc1.Publish([]byte(fmt.Sprintln("message ", i)))
		checkError(t, err)
	}

	time.Sleep(time.Second * 10)
	t.Log("finished")
}
