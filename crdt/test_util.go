package crdtverse

import (
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

func newStore(t *testing.T, hGen pv.HostGenerator, baseDir, name, mode, bAddrInfo string, opts ...*StoreOpts) IStore {
	bai := pv.AddrInfoFromString(bAddrInfo)
	v := NewVerse(hGen, baseDir, false, false, bai)
	db, err := v.NewStore(name, mode, opts...)
	checkError(t, err)
	return db
}
func loadStore(t *testing.T, hGen pv.HostGenerator, baseDir, addr, mode, bAddrInfo string, opts ...*StoreOpts) IStore {
	bai := pv.AddrInfoFromString(bAddrInfo)
	v := NewVerse(hGen, baseDir, false, false, bai)
	db, err := v.LoadStore(addr, mode, opts...)
	checkError(t, err)
	return db
}

func newAccessController(t *testing.T, hGen pv.HostGenerator, baseDir, name, bAddrInfo string, keys ...string) *accessController {
	accesses := make(chan string)
	go func() {
		defer close(accesses)
		for _, key := range keys {
			accesses <- key
		}
	}()

	bai := pv.AddrInfoFromString(bAddrInfo)
	v := NewVerse(hGen, baseDir, false, false, bai)
	ac, err := v.NewAccessController(name, accesses)
	checkError(t, err)
	return ac
}

func newTimeController(t *testing.T, hGen pv.HostGenerator, baseDir, name, bAddrInfo string, begin, end time.Time) *timeController {
	bai := pv.AddrInfoFromString(bAddrInfo)
	v := NewVerse(hGen, baseDir, false, false, bai)
	tc, err := v.NewTimeController(name, begin, end, time.Minute*2, time.Second*10, 1)
	checkError(t, err)
	return tc
}
