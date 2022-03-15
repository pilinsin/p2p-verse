package crdtverse

import(
	"encoding/base64"

	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
	pv "github.com/pilinsin/p2p-verse"
	"golang.org/x/crypto/argon2"
)

type hashData struct{
	BaseHash []byte
	Salt []byte
	Value []byte
}
func UnmarshalHashData(m []byte) (*hashData, error){
	hsv := &hashData{}
	if err := pv.Unmarshal(m, hsv); err != nil{
		return nil, err
	}
	return hsv, nil
}


func makeHashKey(bHash, salt []byte) string{
	hash := argon2.IDKey(bHash, salt, 1, 64*1024, 4, 128)
	key := base64.StdEncoding.EncodeToString(hash)
	return key
}

type hashValidator struct{}
func (v *hashValidator) Validate(key string, val []byte) bool{
	data := hashData{}
	if err := pv.Unmarshal(val, &data); err != nil{return false}

	hKey := makeHashKey(data.BaseHash, data.Salt)
	return key[1:] == hKey
}
func (v *hashValidator) Select(key string, vals [][]byte) bool{
	return len(vals) == 1
}

type hashStore struct{
	*logStore
	salt []byte
}
func (cv *crdtVerse) NewHashStore(name string, salt []byte) (*hashStore, error){
	st, err := cv.newCRDT(name, &hashValidator{})
	if err != nil{return nil, err}
	return &hashStore{st, salt}, nil
}
func (s *hashStore) MakeHashKey(bHashStr string) string{
	hb := []byte(base64.StdEncoding.EncodeToString([]byte(bHashStr)))
	return makeHashKey(hb, s.salt)
}
func (s *hashStore) Put(bHashStr string, val []byte) error{
	hb := []byte(base64.StdEncoding.EncodeToString([]byte(bHashStr)))
	key := makeHashKey(hb, s.salt)
	hd := &hashData{hb, s.salt, val}
	m, err := pv.Marshal(hd)
	if err != nil{return err}
	return s.dt.Put(s.ctx, ds.NewKey(key), m)
}
func (s *hashStore) Get(key string) ([]byte, error){
	m, err := s.dt.Get(s.ctx, ds.NewKey(key))
	if err != nil{return nil, err}
	hd := hashData{}
	if err := pv.Unmarshal(m, &hd); err != nil{return nil, err}
	return hd.Value, nil
}
func (s *hashStore) GetSize(key string) (int, error){
	return s.dt.GetSize(s.ctx, ds.NewKey(key))
}
func (s *hashStore) Has(key string) (bool, error){
	return s.dt.Has(s.ctx, ds.NewKey(key))
}
func (s *hashStore) Query(q query.Query) (query.Results, error){
	return s.dt.Query(s.ctx, q)
}