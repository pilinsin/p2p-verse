package dberse

import(
	"errors"

	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

func MakeCidKey(val []byte) (string, error){
	format := cid.V1Builder{Codec: cid.Codecs["cbor"], MhType: mh.SHA3}
	c, err := format.Sum(val)
	if err != nil{return "", err}
	return c.String(), nil
}
type cidValidator struct{}
func (v cidValidator) Validate(key0 string, val []byte) error{
	ns, _, key, err := splitKey(key0)
	if err != nil{return err}
	if ns != v.Type(){return errors.New("invalid key")}

	c, err := MakeCidKey(val)
	if err != nil{return err}
	
	if key != c{return errors.New("invalid key")}
	return nil	
}
func (v cidValidator) Select(key string, vals [][]byte) (int, error){
	if len(vals) > 1{
		return -1, nil
	}
	return 0, nil
}
func (v cidValidator) Type() string{
	return "/cid"
}



func (d *dBerse) NewCidStore(name string) (iStore, error){
	db, err := d.newStore(name, Cid)
	if err != nil{return nil, err}
	return &cidStore{db}, nil
}
type cidStore struct{
	*store
}
func (d *cidStore) Put(val []byte, keys ...string) error{
	key, err := MakeCidKey(val)
	if err != nil{return err}
	return d.store.Put(val, key)
}