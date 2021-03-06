package crdtverse

import (
	"testing"

	peer "github.com/libp2p/go-libp2p-core/peer"
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

func newStore(t *testing.T, hGen pv.HostGenerator, baseDir, name, mode, bAddrInfo string, opts ...*StoreOpts) IStore {
	bai := pv.AddrInfoFromString(bAddrInfo)
	v := NewVerse(hGen, baseDir, false, bai)
	db, err := v.NewStore(name, mode, opts...)
	assertError(t, db != nil, "newStore error:", err)
	return db
}

func newAccessStore(t *testing.T, st IStore, keys ...string) IAccessStore {
	accesses := make(chan string)
	go func() {
		defer close(accesses)
		for _, key := range keys {
			accesses <- key
		}
	}()

	v := NewVerse(nil, "", false, peer.AddrInfo{})
	ac, err := v.NewAccessStore(st, accesses)
	checkError(t, err)
	return ac
}
