package crdtverse
/*

import(
	host "github.com/libp2p/go-libp2p-core/host"
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)


func MakeCidKey(val []byte) string{
	format := cid.V1Builder{Codec: cid.Codecs["cbor"], MhType: mh.SHA3}
	c, _ := format.Sum(val)
	return c.String()
}

type cidValidator struct{}
func newCidValidator(iValidator) iValidator{return &cidValidator{}}
func (v *cidValidator) Validate(key string, val []byte) bool{
	vKey := MakeCidKey(val)
	return key[1:] == vKey
}
func (v *cidValidator) Select(key string, vals [][]byte) bool{
	return len(vals) == 1
}
func (v *cidValidator) Type() string{return "cid"}


type cidStore struct{
	*logStore
}
func (cv *crdtVerse) NewCidStore(self host.Host, name string, _ ...*StoreOpts) (iStore, error){
	st, err := cv.newCRDT(self, name, &cidValidator{})
	if err != nil{return nil, err}
	return &cidStore{st}, nil
}
func (s *cidStore) Put(_ string, val []byte) error{
	key := MakeCidKey(val)
	return s.logStore.Put(key, val)
}

*/