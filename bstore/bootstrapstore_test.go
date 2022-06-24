package bootstrapstore

import (
	"os"
	"testing"
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

func TestBootstrapStore(t *testing.T) {
	stName := "storeName: st.Address()"
	b, err := pv.NewBootstrap(pv.SampleHost)
	checkError(t, err)
	bAddr := pv.AddrInfoToString(b.AddrInfo())
	t.Log(bAddr)

	bs1, err := NewBootstrapStore("dir1")
	checkError(t, err)

	bs2, err := NewBootstrapStore("dir2")
	checkError(t, err)

	checkError(t, bs1.Put(stName, bAddr))
	time.Sleep(time.Second * 10)

	ais1, err := bs1.Get(stName)
	checkError(t, err)
	t.Log("bs1.Get:", ais1)

	ais2, err := bs2.Get(stName)
	checkError(t, err)
	t.Log("bs2.Get:", ais2)

	bs1.Close()
	bs2.Close()
	b.Close()
	os.RemoveAll("dir1")
	os.RemoveAll("dir2")
}
