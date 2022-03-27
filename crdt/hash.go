package crdtverse

import(
	"errors"
	"strings"
	"encoding/base64"
	"crypto/rand"

	pv "github.com/pilinsin/p2p-verse"
	"golang.org/x/crypto/argon2"
	query "github.com/ipfs/go-datastore/query"
)

type hashData struct{
	BaseHash string
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


func MakeHashKey(bHashStr string, salt []byte) string{
	hb := []byte(base64.URLEncoding.EncodeToString([]byte(bHashStr)))
	hash := argon2.IDKey(hb, salt, 1, 64*1024, 4, 128)
	return base64.URLEncoding.EncodeToString(hash)
}

type hashValidator struct{}
func (v *hashValidator) Validate(key string, val []byte) bool{
	data := hashData{}
	if err := pv.Unmarshal(val, &data); err != nil{return false}

	hKey := MakeHashKey(data.BaseHash, data.Salt)
	return key[1:] == hKey
}
func (v *hashValidator) Select(key string, vals [][]byte) bool{
	return len(vals) == 1
}


func getHashOpts(opts ...*StoreOpts) ([]byte, *accessController){
	if len(opts) == 0{
		salt := make([]byte, 8)
		rand.Read(salt)
		return salt, nil
	}
	if opts[0].Salt == nil{
		opts[0].Salt = make([]byte, 8)
		rand.Read(opts[0].Salt)
	}
	return opts[0].Salt, opts[0].Ac
}

type hashStore struct{
	*logStore
	salt []byte
	ac *accessController
}
func (cv *crdtVerse) NewHashStore(name string, opts ...*StoreOpts) (iStore, error){
	salt, ac := getHashOpts(opts...)
	st, err := cv.newCRDT(name, &hashValidator{})
	if err != nil{return nil, err}
	return &hashStore{st, salt, ac}, nil
}
func (cv *crdtVerse) LoadHashStore(addr string, opts ...*StoreOpts) (iStore, error){
	addrs := strings.Split(strings.TrimPrefix(addr, "/"), "/")
	s, err := cv.NewHashStore(addrs[0], opts...)
	if err != nil{return nil, err}
	if len(addrs) >= 2{
		ac, err := cv.LoadAccessController(addrs[1])
		if err != nil{return nil, err}
		s.(*hashStore).ac = ac
	}
	return s, nil
}
func (s *hashStore) Close(){
	if s.ac != nil{s.ac.Close()}
	s.logStore.Close()
}
func (s *hashStore) Address() string{
	if s.ac != nil{return s.name + "/" + s.ac.Address()}
	return s.name
}
func (s *hashStore) verify(key string) error{
	if s.ac != nil{
		ok, err := s.ac.Has(key)
		if !ok || err != nil{
			return errors.New("permission error")
		}
	}
	return nil
}
func (s *hashStore) Put(bHashStr string, val []byte) error{
	key := MakeHashKey(bHashStr, s.salt)
	if err := s.verify(key); err != nil{return err}

	hd := &hashData{bHashStr, s.salt, val}
	m, err := pv.Marshal(hd)
	if err != nil{return err}
	return s.logStore.Put(key, m)
}
func (s *hashStore) Get(key string) ([]byte, error){
	if err := s.verify(key); err != nil{return nil, err}

	m, err := s.logStore.Get(key)
	if err != nil{return nil, err}
	hd := hashData{}
	if err := pv.Unmarshal(m, &hd); err != nil{return nil, err}
	return hd.Value, nil
}
func (s *hashStore) GetSize(key string) (int, error){
	if err := s.verify(key); err != nil{return -1, err}

	val, err := s.Get(key)
	if err != nil{return -1, err}
	return len(val), nil
}
func (s *hashStore) Has(key string) (bool, error){
	if err := s.verify(key); err != nil{return false, err}

	return s.logStore.Has(key)
}
func (s *hashStore) Query(qs ...query.Query) (query.Results, error){
	var q query.Query
	if len(qs) == 0{
		q = query.Query{}
	}else{
		q = qs[0]
	}
	if s.ac != nil{
		q.Filters = append(q.Filters, acFilter{s.ac})
	}
	return s.logStore.Query(q)
}
