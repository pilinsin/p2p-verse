package crdtverse

import(
	"context"
	"os"

	pv "github.com/pilinsin/p2p-verse"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
	crdt "github.com/ipfs/go-ds-crdt"
)

type crdtVerse struct{
	ctx context.Context
	cancel func()
	dirPath string
	save bool
	h host.Host
	bootstraps []peer.AddrInfo
}
func NewVerse(dir string, save bool, self host.Host, bootstraps ...peer.AddrInfo) *crdtVerse{
	ctx, cancel := context.WithCancel(context.Background())
	return &crdtVerse{ctx, cancel, dir, save, self, bootstraps}
}
func (cv *crdtVerse) newCRDT(name string, v validator) (*logStore, error){
	if err := os.MkdirAll(cv.dirPath, 0700); err != nil{return	nil, err}
	dsCancel := func(){}
	if !cv.save{dsCancel = func(){os.RemoveAll(cv.dirPath)}}

	sp, err := cv.setupStore(name, v)
	if err != nil{return nil, err}
	return &logStore{cv.ctx, cv.cancel, dsCancel, sp.dht, sp.dStore, sp.dt}, nil
}


type validator interface{
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

//hashValidator
//cidValidator
//signatureValidator
//accessValidator(Element.Value.Key)?

type logStore struct{
	ctx context.Context
	cancel func()
	dsCancel func()
	dht *pv.DiscoveryDHT
	dStore ds.Datastore
	dt *crdt.Datastore
}
func (cv *crdtVerse) NewStore(name string) (*logStore, error){
	return cv.newCRDT(name, &logValidator{})
}
func (s *logStore) Close(){
	s.cancel()
	s.dt.Close()
	s.dStore.Close()
	s.dht.Close()
	s.dsCancel()
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



