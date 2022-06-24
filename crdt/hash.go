package crdtverse

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	query "github.com/ipfs/go-datastore/query"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	"golang.org/x/crypto/argon2"
	proto "google.golang.org/protobuf/proto"
)

func MakeHashKey(bHashStr string, salt []byte) string {
	hash := argon2.IDKey([]byte(bHashStr), salt, 1, 64*1024, 4, 64)
	return base64.URLEncoding.EncodeToString(hash)
}

type hashValidator struct {
	iValidator
}

func newHashValidator(s IStore) iValidator {
	return &hashValidator{newBaseValidator(s)}
}
func (v *hashValidator) Validate(key string, val []byte) bool {
	if ok := v.iValidator.Validate(key, val); !ok {
		return false
	}

	key = strings.TrimPrefix(key, "/")
	data := &pb.HashData{}
	if err := proto.Unmarshal(val, data); err != nil {
		return false
	}
	hKey := MakeHashKey(data.GetBaseHash(), data.GetSalt())
	return key == hKey
}

func getHashOpts(opts ...*StoreOpts) ([]byte, time.Time) {
	if len(opts) == 0 {
		salt := make([]byte, 8)
		rand.Read(salt)
		return salt, time.Time{}
	}
	if opts[0].Salt == nil {
		opts[0].Salt = make([]byte, 8)
		rand.Read(opts[0].Salt)
	}
	return opts[0].Salt, opts[0].TimeLimit
}

type hashStore struct {
	*baseStore
	salt []byte
}

func (cv *crdtVerse) NewHashStore(name string, opts ...*StoreOpts) (IStore, error) {
	st := &baseStore{}
	if err := cv.initCRDT(name, newHashValidator(st), st); err != nil {
		return nil, err
	}

	salt, tl := getHashOpts(opts...)
	st.timeLimit = tl
	st.setTimeLimit()
	return &hashStore{st, salt}, nil
}

func (s *hashStore) putKey(key string) string{
	return MakeHashKey(key, s.salt)
}
func (s *hashStore) Put(bHashStr string, val []byte) error {
	key := MakeHashKey(bHashStr, s.salt)

	hd := &pb.HashData{
		BaseHash: bHashStr,
		Salt:     s.salt,
		Value:    val,
	}
	m, err := proto.Marshal(hd)
	if err != nil {
		return err
	}
	return s.baseStore.Put(key, m)
}
func (s *hashStore) get(key string) ([]byte, error) {
	m, err := s.baseStore.Get(key)
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
func (s *hashStore) Has(key string) (bool, error) {
	ok, err := s.baseStore.Has(key)
	if ok && err == nil {
		return true, nil
	}

	key = MakeHashKey(key, s.salt)
	return s.baseStore.Has(key)
}
func (s *hashStore) Query(qs ...query.Query) (query.Results, error) {
	var q query.Query
	if len(qs) == 0 {
		q = query.Query{}
	} else {
		q = qs[0]
	}
	
	rs, err := s.baseStore.Query(q)
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

func (s *hashStore) initPut() error{
	key := MakeHashKey(s.name, s.salt)

	hd := &pb.HashData{
		BaseHash: s.name,
		Salt:     s.salt,
		Value:    []byte(s.name),
	}
	m, err := proto.Marshal(hd)
	if err != nil {
		return err
	}
	return s.baseStore.Put(key, m)
}
func (s *hashStore) loadCheck() bool{
	if !s.inTime{return true}

	rs, err := s.baseStore.Query(query.Query{
		KeysOnly: true,
		Limit: 1,
	})
	if err != nil{return false}

	resList, err := rs.Rest()
	return len(resList) > 0 && err == nil
}