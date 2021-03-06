package ipfsverse

import (
	"bytes"
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
func assertError(t *testing.T, cond bool, args ...interface{}) {
	if !cond {
		t.Fatal(args...)
	}
}

//go test -test.v=true .
func BaseTestIpfs(t *testing.T, hGen pv.HostGenerator) {
	bstrp, err := pv.NewBootstrap(hGen)
	checkError(t, err)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	ipfs, err := NewIpfsStore(hGen, "ipfs", true, bAddrInfo)
	checkError(t, err)

	c, err := ipfs.Add([]byte("meow meow ^.^"))
	checkError(t, err)
	c_2, err := ipfs.AddReader(bytes.NewBufferString("meow meow ^.^"))
	checkError(t, err)
	assertError(t, c == c_2, "Add and AddReader must return the same cid")
	t.Log("add,", c)
	ipfs.Close()

	ipfs, err = NewIpfsStore(hGen, "ipfs", false, bAddrInfo)
	checkError(t, err)
	defer ipfs.Close()

	ipfs2, err := NewIpfsStore(hGen, "ipfs2", false, bAddrInfo)
	checkError(t, err)
	defer ipfs2.Close()

	v, err := ipfs2.Get(c)
	checkError(t, err)
	r, err := ipfs2.GetReader(c)
	checkError(t, err)
	b := &bytes.Buffer{}
	_, err = b.ReadFrom(r)
	checkError(t, err)
	assertError(t, bytes.Equal(v, b.Bytes()), "Get and GetReader must return the same bytes")
	t.Log("get,", string(v))

	cg, err := NewCidGetter()
	checkError(t, err)
	defer cg.Close()
	c2, err := cg.Get([]byte("meow meow ^.^"))
	checkError(t, err)
	assertError(t, c == c2, "different cid for the same []byte")

	t.Log("finished")
}
