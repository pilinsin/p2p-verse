package crdtverse

import(
	"context"
	"strings"
	"path/filepath"
	"os"
	"io"

	pv "github.com/pilinsin/p2p-verse"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
	crdt "github.com/ipfs/go-ds-crdt"
)

type hostGenerator func(...io.Reader) (host.Host, error)

type crdtVerse struct{
	hGenerator hostGenerator
	dirPath string
	save bool
	useMemory bool
	bootstraps []peer.AddrInfo
}
func NewVerse(hGen hostGenerator, dir string, save, useMemory bool, bootstraps ...peer.AddrInfo) *crdtVerse{
	return &crdtVerse{hGen, dir, save, useMemory, bootstraps}
}
func (cv *crdtVerse) newCRDT(name string, v iValidator) (*logStore, error){
	h, err := cv.hGenerator()
	if err != nil{return nil, err}

	dirAddr := filepath.Join(cv.dirPath, name)
	if err := os.MkdirAll(dirAddr, 0700); err != nil{return nil, err}
	dsCancel := func(){}
	if !cv.save{dsCancel = func(){os.RemoveAll(dirAddr)}}

	ctx, cancel := context.WithCancel(context.Background())
	sp, err := cv.setupStore(ctx, h, name, v)
	if err != nil{return nil, err}
	st := &logStore{ctx, cancel, dsCancel, name, sp.dht, sp.dStore, sp.dt}
	return st, nil
}
func (cv *crdtVerse) NewStore(name, mode string, opts ...*StoreOpts) (iStore, error){
	switch mode {
	case "updatable":
		return cv.NewUpdatableStore(name, opts...)
	case "signature":
		return cv.NewSignatureStore(name, opts...)
	case "updatableSignature":
		return cv.NewUpdatableSignatureStore(name, opts...)
	case "hash":
		return cv.NewHashStore(name, opts...)
	default:
		return cv.NewLogStore(name, opts...)
	}
}
func (cv *crdtVerse) LoadStore(addr, mode string, opts ...*StoreOpts) (iStore, error){
	switch mode {
	case "updatable":
		return cv.LoadUpdatableStore(addr, opts...)
	case "signature":
		return cv.LoadSignatureStore(addr, opts...)
	case "updatableSignature":
		return cv.LoadUpdatableSignatureStore(addr, opts...)
	case "hash":
		return cv.LoadHashStore(addr, opts...)
	default:
		return cv.LoadLogStore(addr, opts...)
	}
}


type iValidator interface{
	Validate(string, []byte) bool
	Select(string, [][]byte) bool
}

type logValidator struct{}
func (v *logValidator) Validate(key string, val []byte) bool{
	return true
}
func (v *logValidator) Select(key string, vals [][]byte) bool{
	return len(vals) == 1
}


//data key: <pid>/<category>/<tKey>
type KeyMatchFilter struct{
	key string
}
func (f KeyMatchFilter) Filter(e query.Entry) bool{
	keys := strings.Split(strings.TrimPrefix(e.Key, "/"), "/")
	fKeys := strings.Split(strings.TrimPrefix(f.key, "/"), "/")
	if len(keys) < len(fKeys){return false}
	
	for idx := range fKeys{
		if fKeys[idx] != "*" && fKeys[idx] != keys[idx]{
			return false
		}
	}
	return true
}

type iStore interface{
	Close()
	Address() string
	Sync() error
	Repair() error
	Put(string, []byte) error
	Get(string) ([]byte, error)
	GetSize(string) (int, error)
	Has(string) (bool, error)
	Query(...query.Query) (query.Results, error)
}

type StoreOpts struct{
	Salt []byte
	Priv p2pcrypto.PrivKey
	Pub p2pcrypto.PubKey
	Ac *accessController
	Tc *timeController
}

type logStore struct{
	ctx context.Context
	cancel func()
	dsCancel func()
	name string
	dht *pv.DiscoveryDHT
	dStore ds.Datastore
	dt *crdt.Datastore
}
func (cv *crdtVerse) NewLogStore(name string, _ ...*StoreOpts) (iStore, error){
	return cv.newCRDT(name, &logValidator{})
}
func (cv *crdtVerse) LoadLogStore(addr string, _ ...*StoreOpts) (iStore, error){
	addr = strings.Split(strings.TrimPrefix(addr, "/"), "/")[0]
	return cv.NewLogStore(addr)
}
func (s *logStore) Close(){
	s.cancel()
	s.dt.Close()
	s.dStore.Close()
	s.dht.Close()
	s.dsCancel()
}
func (s *logStore) Address() string{
	return s.name
}
func (s *logStore) Sync() error{
	return s.dt.Sync(s.ctx, ds.NewKey("/"))
}
func (s *logStore) Repair() error{
	return s.dt.Repair()
}
func (s *logStore) Put(key string, val []byte) error{
	return s.dt.Put(s.ctx, ds.NewKey(key), val)
}
func (s *logStore) Get(key string) ([]byte, error){
	return s.dt.Get(s.ctx, ds.NewKey(key))
}
func (s *logStore) GetSize(key string) (int, error){
	return s.dt.GetSize(s.ctx, ds.NewKey(key))
}
func (s *logStore) Has(key string) (bool, error){
	return s.dt.Has(s.ctx, ds.NewKey(key))
}
func (s *logStore) Query(qs ...query.Query) (query.Results, error){
	var q query.Query
	if len(qs) == 0{
		q = query.Query{}
	}else{
		q = qs[0]
	}
	return s.dt.Query(s.ctx, q)
}



