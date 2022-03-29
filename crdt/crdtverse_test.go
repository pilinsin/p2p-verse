package crdtverse

import(
	"testing"
	"time"

	query "github.com/ipfs/go-datastore/query"
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

func TestCRDT(t *testing.T){
	b, err := pv.SampleHost()
	checkError(t, err)
	bstrp, err := pv.NewBootstrap(b)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	v0 := NewVerse(pv.SampleHost, "a", false, false, bAddrInfo)
	opts0 := &StoreOpts{}
	db0, err := v0.NewStore("testDB", "updatableSignature", opts0)
	checkError(t, err)
	defer db0.Close()
	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))

	v1 := NewVerse(pv.SampleHost, "b", false, false, bAddrInfo)
	opts1 := &StoreOpts{}
	db1, err := v1.NewStore("testDB", "updatableSignature", opts1)
	checkError(t, err)
	defer db1.Close()

	<-time.Tick(time.Second*30)

	checkError(t, db1.Sync())
	v10, err := db1.Get(PubKeyToStr(opts0.Pub)+"/aaa")
	checkError(t, err)
	t.Log(string(v10))
	
	rs1, err := db1.Query(query.Query{
		Filters: []query.Filter{dataFilter{"aaa"}},
		Limit:1,
	})
	checkError(t, err)
	resList, err := rs1.Rest()
	checkError(t, err)
	t.Log(len(resList))

	checkError(t, db0.Put("aaa", []byte("meow meow 2 ^.^")))

	<-time.Tick(time.Minute)

	checkError(t, db1.Sync())
	v12, err := db1.Get(PubKeyToStr(opts0.Pub)+"/aaa")
	checkError(t, err)
	t.Log(string(v12))

	rs12, err := db1.Query(query.Query{
		Filters: []query.Filter{dataFilter{"aaa"}},
	})
	checkError(t, err)
	resList2, err := rs12.Rest()
	checkError(t, err)
	t.Log(len(resList2))


	<- time.Tick(time.Second*3)
	t.Log("finished")
}