package crdtverse

import (
	"time"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	query "github.com/ipfs/go-datastore/query"
	pv "github.com/pilinsin/p2p-verse"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	"golang.org/x/crypto/argon2"
	proto "google.golang.org/protobuf/proto"
)

func MakeHashKey(bHashStr string, salt []byte) string {
	hash := argon2.IDKey([]byte(bHashStr), salt, 1, 64*1024, 4, 64)
	return base64.URLEncoding.EncodeToString(hash)
}

type hashValidator struct{
	iValidator
}
func newHashValidator(s IStore) iValidator{
	return &hashValidator{newLogValidator(s)}
}
func (v *hashValidator) Validate(key string, val []byte) bool {
	if ok := v.iValidator.Validate(key, val); !ok{return false}

	key = strings.TrimPrefix(key, "/")
	data := &pb.HashData{}
	if err := proto.Unmarshal(val, data); err != nil {
		return false
	}
	hKey := MakeHashKey(data.GetBaseHash(), data.GetSalt())
	return key == hKey
}

func getHashOpts(opts ...*StoreOpts) ([]byte, *accessController, time.Time) {
	if len(opts) == 0 {
		salt := make([]byte, 8)
		rand.Read(salt)
		return salt, nil, time.Time{}
	}
	if opts[0].Salt == nil {
		opts[0].Salt = make([]byte, 8)
		rand.Read(opts[0].Salt)
	}
	return opts[0].Salt, opts[0].Ac, opts[0].TimeLimit
}

type hashStore struct {
	*logStore
	salt []byte
	ac   *accessController
}

func (cv *crdtVerse) NewHashStore(name string, opts ...*StoreOpts) (IStore, error) {
	st := &logStore{}
	if err := cv.initCRDT(name, newHashValidator(st), st); err != nil {
		return nil, err
	}

	salt, ac, tl := getHashOpts(opts...)
	st.timeLimit = tl
	st.setTimeLimit()
	return &hashStore{st, salt, ac}, nil
}

func (s *hashStore) Close() {
	if s.ac != nil {
		s.ac.Close()
	}
	s.logStore.Close()
}
func (s *hashStore) Cancel() {
	s.logStore.Cancel()
}
func (s *hashStore) Address() string {
	name := s.logStore.Address()
	if s.ac != nil {
		name += "/" + s.ac.Address()
	}
	return name
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

func (s *hashStore) InitPut(bHashStr string) error {
	hKey := MakeHashKey(bHashStr, s.salt)
	val := pv.RandBytes(8)

	hd := &pb.HashData{
		BaseHash: bHashStr,
		Salt:     s.salt,
		Value:    val,
	}
	m, err := proto.Marshal(hd)
	if err != nil {
		return err
	}
	return s.logStore.Put(hKey, m)
}
func (s *hashStore) LoadCheck() bool {
	if !s.isInTime(){return true}

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
