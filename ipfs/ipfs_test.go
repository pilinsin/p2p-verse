package ipfsverse

import (
	"time"
	"testing"
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
func TestIpfs(t *testing.T) {
	b, err := pv.SampleHost()
	checkError(t, err)
	bstrp, err := pv.NewBootstrap(b)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	ipfs, kwCid, err := NewIpfs(pv.SampleHost, "ipfs", "dht-kw", true, false, bAddrInfo)
	checkError(t, err)

	c, err := ipfs.Add([]byte("meow meow ^.^"))
	checkError(t, err)
	t.Log("add,", c)
	ipfs.Close()
	time.Sleep(time.Minute)

	ipfs, err = LoadIpfs(pv.SampleHost, "ipfs", "dht-kw", kwCid, false, false, bAddrInfo)
	checkError(t, err)
	defer ipfs.Close()

	ipfs2, err := LoadIpfs(pv.SampleHost, "ipfs2", "dht-kw", kwCid, false, false, bAddrInfo)
	checkError(t, err)
	defer ipfs2.Close()

	v, err := ipfs2.Get(c)
	checkError(t, err)
	t.Log("get,", string(v))

	t.Log("finished")
}
