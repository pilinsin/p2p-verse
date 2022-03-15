package crdtverse

import(
	ds "github.com/ipfs/go-datastore"
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)


func makeCidKey(val []byte) (string, error){
	format := cid.V1Builder{Codec: cid.Codecs["cbor"], MhType: mh.SHA3}
	c, err := format.Sum(val)
	if err != nil{return "", err}
	return c.String(), nil
}

type cidValidator struct{}
func (v *cidValidator) Validate(key string, val []byte) bool{
	vKey, err := makeCidKey(val)
	return err == nil && key[1:] == vKey
}
func (v *cidValidator) Select(key string, vals [][]byte) bool{
	return len(vals) == 1
}

type cidStore struct{
	*logStore
}
func (cv *crdtVerse) NewCidStore(name string) (*cidStore, error){
	st, err := cv.newCRDT(name, &cidValidator{})
	if err != nil{return nil, err}
	return &cidStore{st}, nil
}
func (s *cidStore) MakeCidKey(data []byte) (string, error){
	return makeCidKey(data)
}
func (s *cidStore) Put(val []byte) error{
	key, err := makeCidKey(val)
	if err != nil{return err}
	return s.dt.Put(s.ctx, ds.NewKey(key), val)
}
