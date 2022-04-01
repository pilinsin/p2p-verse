package crdtverse

import(
	"testing"
	"time"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
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


func newStore(t *testing.T, baseDir, name, mode, bAddrInfo string, opts ...*StoreOpts)iStore{
	bai := pv.AddrInfoFromString(bAddrInfo)
	v := NewVerse(pv.SampleHost, baseDir, false, false, bai)
	db, err := v.NewStore(name, mode, opts...)
	checkError(t, err)
	return db
}
func loadStore(t *testing.T, baseDir, addr, mode, bAddrInfo string, opts ...*StoreOpts)iStore{
	bai := pv.AddrInfoFromString(bAddrInfo)
	v := NewVerse(pv.SampleHost, baseDir, false, false, bai)
	db, err := v.LoadStore(addr, mode, opts...)
	checkError(t, err)
	return db
}

func TestLogStore(t *testing.T){
	b, err := pv.SampleHost()
	checkError(t, err)
	bstrp, err := pv.NewBootstrap(b)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	<-time.Tick(time.Second*10)

	db0 := newStore(t, "ls/la", "lg", "log", pv.AddrInfoToString(bAddrInfo))
	defer db0.Close()
	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("db0 generated")

	<-time.Tick(time.Second*30)

	db1 := loadStore(t, "ls/lb", db0.Address(), "log", pv.AddrInfoToString(bAddrInfo))
	defer db1.Close()
	t.Log("db1 generated")
	
	<-time.Tick(time.Second*30)

	checkError(t, db1.Sync())
	v10, err := db1.Get("aaa")
	checkError(t, err)
	t.Log(string(v10))
	ok, err := db1.Has("aaa")
	t.Log(ok, err)
	
	rs1, err := db1.Query(query.Query{
		Filters: []query.Filter{KeyMatchFilter{"aaa"}},
		Limit:1,
	})
	checkError(t, err)
	for res := range rs1.Next(){
		t.Log(string(res.Value))
	}

	checkError(t, db0.Sync())
	checkError(t, db0.Put("aaa", []byte("meow meow 2 ^.^")))
	<-time.Tick(time.Second*5)


	checkError(t, db1.Sync())
	v12, err := db1.Get("aaa")
	checkError(t, err)
	t.Log(string(v12))

	rs12, err := db1.Query(query.Query{
		Filters: []query.Filter{KeyMatchFilter{"aaa"}},
	})
	checkError(t, err)
	for res := range rs12.Next(){
		t.Log(string(res.Value))
	}


	t.Log("finished")
}


func TestSignatureStore(t *testing.T){
	b, err := pv.SampleHost()
	checkError(t, err)
	bstrp, err := pv.NewBootstrap(b)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	opts0 := &StoreOpts{}
	db0 := newStore(t, "ss/sa", "sg", "signature", pv.AddrInfoToString(bAddrInfo), opts0)
	defer db0.Close()
	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))

	<-time.Tick(time.Second*30)

	db1 := loadStore(t, "ss/sb", db0.Address(), "signature", pv.AddrInfoToString(bAddrInfo))
	defer db1.Close()
	
	<-time.Tick(time.Second*30)

	checkError(t, db1.Sync())
	v10, err := db1.Get(PubKeyToStr(opts0.Pub)+"/aaa")
	checkError(t, err)
	t.Log(string(v10))
	ok, err := db1.Has(PubKeyToStr(opts0.Pub)+"/aaa")
	t.Log(ok, err)
	
	rs1, err := db1.Query(query.Query{
		Filters: []query.Filter{KeyMatchFilter{"/*/aaa"}},
		Limit:1,
	})
	checkError(t, err)
	for res := range rs1.Next(){
		t.Log(string(res.Value))
	}


	checkError(t, db0.Put("aaa", []byte("meow meow 2 ^.^")))
	<-time.Tick(time.Second*10)


	checkError(t, db1.Sync())
	v12, err := db1.Get(PubKeyToStr(opts0.Pub)+"/aaa")
	checkError(t, err)
	t.Log(string(v12))

	rs12, err := db1.Query(query.Query{
		Filters: []query.Filter{KeyMatchFilter{"/*/aaa"}},
	})
	checkError(t, err)
	for res := range rs12.Next(){
		t.Log(string(res.Value))
	}


	t.Log("finished")
}


func newAccessController(t *testing.T, baseDir, name, bAddrInfo string, keys ...string) *accessController{
	accesses := make(chan string)
	go func(){
		defer close(accesses)
		for _, key := range keys{
			accesses <- key
		}
	}()

	bai := pv.AddrInfoFromString(bAddrInfo)
	v := NewVerse(pv.SampleHost, baseDir, false, false, bai)
	ac, err := v.NewAccessController(name, accesses)
	checkError(t, err)
	return ac
}

func TestAccessController(t *testing.T){
	b, err := pv.SampleHost()
	checkError(t, err)
	bstrp, err := pv.NewBootstrap(b)
	defer bstrp.Close()
	bAddrInfo := bstrp.AddrInfo()
	t.Log("bootstrap AddrInfo: ", bAddrInfo)

	<-time.Tick(time.Second*5)

	priv, pub, _ := p2pcrypto.GenerateEd25519Key(nil)
	pid := PubKeyToStr(pub)
	t.Log("pid:", pid)
	ac := newAccessController(t, "ac/c", "ac", pv.AddrInfoToString(bAddrInfo), pid)
	t.Log("new : name:", ac.name, ", salt:", ac.salt)

	opts0 := &StoreOpts{Priv: priv, Pub: pub, Ac: ac}
	db0 := newStore(t, "ac/aa", "us", "updatableSignature", pv.AddrInfoToString(bAddrInfo), opts0)
	defer db0.Close()
	t.Log("db0 generated")
	checkError(t, db0.Put("aaa", []byte("meow meow ^.^")))
	t.Log("put done")
	v00, err := db0.Get(PubKeyToStr(opts0.Pub)+"/aaa")
	checkError(t, err)
	t.Log("db0.Get:", string(v00))
	<-time.Tick(time.Second*30)

	db1 := loadStore(t, "ac/ab", db0.Address(), "updatableSignature", pv.AddrInfoToString(bAddrInfo))
	defer db1.Close()
	t.Log("db1 generated")
	<-time.Tick(time.Minute)

	checkError(t, db1.Sync())
	t.Log("aaaaaa")
	rs1, err := db1.Query()
	t.Log("bbbbbbb")
	checkError(t, err)
	for res := range rs1.Next(){
		t.Log(res.Key, string(res.Value))
	}
	v10, err := db1.Get(PubKeyToStr(opts0.Pub)+"/aaa")
	checkError(t, err)
	t.Log("db1.Get:", string(v10))


	t.Log("finished")
}