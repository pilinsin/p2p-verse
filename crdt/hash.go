package crdtverse

import (
	"crypto/rand"
	"encoding/base64"
	"errors"

	query "github.com/ipfs/go-datastore/query"
	pv "github.com/pilinsin/p2p-verse"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	"golang.org/x/crypto/argon2"
	proto "google.golang.org/protobuf/proto"
)

func MakeHashKey(bHashStr string, salt []byte) string {
	hb := []byte(base64.URLEncoding.EncodeToString([]byte(bHashStr)))
	hash := argon2.IDKey(hb, salt, 1, 64*1024, 4, 128)
	return base64.URLEncoding.EncodeToString(hash)
}

type hashValidator struct{}

func (v *hashValidator) Validate(key string, val []byte) bool {
	data := &pb.HashData{}
	if err := proto.Unmarshal(val, data); err != nil {
		return false
	}

	hKey := MakeHashKey(data.GetBaseHash(), data.GetSalt())
	return key[1:] == hKey
}

func getHashOpts(opts ...*StoreOpts) ([]byte, *accessController) {
	if len(opts) == 0 {
		salt := make([]byte, 8)
		rand.Read(salt)
		return salt, nil
	}
	if opts[0].Salt == nil {
		opts[0].Salt = make([]byte, 8)
		rand.Read(opts[0].Salt)
	}
	return opts[0].Salt, opts[0].Ac
}

type hashStore struct {
	*logStore
	salt []byte
	ac   *accessController
}

func (cv *crdtVerse) NewHashStore(name string, opts ...*StoreOpts) (IStore, error) {
	salt, ac := getHashOpts(opts...)
	st, err := cv.newCRDT(name, &hashValidator{})
	if err != nil {
		return nil, err
	}
	return &hashStore{st, salt, ac}, nil
}

func (s *hashStore) Close() {
	if s.ac != nil {
		s.ac.Close()
	}
	s.logStore.Close()
}
func (s *hashStore) Cancel(){
	s.logStore.Cancel()
}
func (s *hashStore) Address() string {
	if s.ac != nil {
		return s.name + "/" + s.ac.Address()
	}
	return s.name
}
func (s *hashStore) verify(key string) error {
	if s.ac != nil {
		ok, err := s.ac.Has(key)
		if !ok || err != nil {
			return errors.New("permission error")
		}
	}
	return nil
}
func (s *hashStore) Put(bHashStr string, val []byte) error {
	key := MakeHashKey(bHashStr, s.salt)
	if err := s.verify(key); err != nil {
		return err
	}

	hd := &pb.HashData{
		BaseHash: bHashStr,
		Salt:     s.salt,
		Value:    val,
	}
	m, err := proto.Marshal(hd)
	if err != nil {
		return err
	}
	return s.logStore.Put(key, m)
}
func (s *hashStore) get(key string) ([]byte, error) {
	if err := s.verify(key); err != nil {
		return nil, err
	}

	m, err := s.logStore.Get(key)
	if err != nil {
		return nil, err
	}
	hd := &pb.HashData{}
	if err := proto.Unmarshal(m, hd); err != nil {
		return nil, err
	}
	return hd.GetValue(), nil
}
func (s *hashStore) Get(key string) ([]byte, error) {
	data, err := s.get(key)
	if err == nil {
		return data, nil
	}

	key = MakeHashKey(key, s.salt)
	return s.get(key)
}
func (s *hashStore) GetSize(key string) (int, error) {
	val, err := s.Get(key)
	if err != nil {
		return -1, err
	}
	return len(val), nil
}
func (s *hashStore) has(key string) (bool, error) {
	if err := s.verify(key); err != nil {
		return false, err
	}
	return s.logStore.Has(key)
}
func (s *hashStore) Has(key string) (bool, error) {
	ok, err := s.has(key)
	if ok && err == nil {
		return true, nil
	}

	key = MakeHashKey(key, s.salt)
	return s.has(key)
}
func (s *hashStore) Query(qs ...query.Query) (query.Results, error) {
	var q query.Query
	if len(qs) == 0 {
		q = query.Query{}
	} else {
		q = qs[0]
	}
	if s.ac != nil {
		q.Filters = append(q.Filters, acFilter{s.ac})
	}
	rs, err := s.logStore.Query(q)
	if err != nil {
		return nil, err
	}
	if q.KeysOnly {
		return rs, nil
	}

	ch := make(chan query.Result)
	go func() {
		defer close(ch)
		for r := range rs.Next() {
			hd := &pb.HashData{}
			if err := proto.Unmarshal(r.Value, hd); err != nil {
				continue
			}

			r.Value = hd.GetValue()
			ch <- r
		}
	}()
	return query.ResultsWithChan(query.Query{}, ch), nil
}

func (s *hashStore) InitPut(key string) error {
	key = MakeHashKey(key, s.salt)
	val := pv.RandBytes(8)

	hd := &pb.HashData{
		BaseHash: key,
		Salt:     s.salt,
		Value:    val,
	}
	m, err := proto.Marshal(hd)
	if err != nil {
		return err
	}
	return s.logStore.Put(key, m)
}
func (s *hashStore) LoadCheck() bool {
	rs, err := s.logStore.Query(query.Query{
		KeysOnly: true,
		Limit:    1,
	})
	if err != nil {
		return false
	}
	resList, err := rs.Rest()
	return len(resList) > 0 && err == nil
}
