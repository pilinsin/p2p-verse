package crdtverse

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	query "github.com/ipfs/go-datastore/query"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	proto "google.golang.org/protobuf/proto"
)

type acFilter struct{}

func (f acFilter) Filter(e query.Entry) bool {
	//sig key is /pid/acKey/...
	//hash key is /hash(acKey, salt)/...
	pub := f.extractPubKey(e.Key)
	if pub == nil {
		return false
	}

	sd := &pb.SignatureData{}
	if err := proto.Unmarshal(e.Value, sd); err != nil {
		return false
	}
	ok, err := pub.Verify(sd.GetValue(), sd.GetSign())
	return ok && err == nil
}
func (f acFilter) extractPubKey(key string) IPubKey {
	keys := strings.Split(strings.TrimPrefix(key, "/"), "/")
	if len(keys) == 0 {
		return nil
	}
	for _, maybeAcKey := range keys {
		mak, err := base64.URLEncoding.DecodeString(maybeAcKey)
		if err != nil {
			continue
		}
		pak := &pb.AccessKey{}
		if err := proto.Unmarshal(mak, pak); err != nil {
			continue
		}
		if pub, err := StrToPubKey(pak.GetMasterKey()); err == nil {
			return pub
		}
	}
	return nil
}

type acVerifyFilter struct {
	ac IAccessStore
}

func (f acVerifyFilter) Filter(e query.Entry) bool {
	err := f.ac.Verify(e.Key)
	return err == nil
}

func getAccessOpts(opts ...*StoreOpts) (IPrivKey, IPubKey) {
	if len(opts) == 0 {
		priv, pub, _ := generateKeyPair()
		return priv, pub
	}
	if opts[0].Pub == nil {
		opts[0].Priv, opts[0].Pub, _ = generateKeyPair()
	}
	return opts[0].Priv, opts[0].Pub
}

type IAccessBaseStore interface {
	IStore
	putKey(string) string
	acQuery(string) (query.Results, error)
	accessFromKey(string) string
}
type IAccessStore interface {
	IAccessBaseStore
	IUpdatableStore
	ISignatureStore
	Grant(string) error
	Verify(string) error
}
type accessStore struct {
	IAccessBaseStore
	priv IPrivKey
	pub  IPubKey
}

func (cv *crdtVerse) NewAccessStore(st IStore, accesses <-chan string, opts ...*StoreOpts) (IAccessStore, error) {
	base, ok := st.(IAccessBaseStore)
	if !ok {
		return nil, errors.New("invalid base store")
	}

	priv, pub := getAccessOpts(opts...)
	ac := &accessStore{
		IAccessBaseStore: base,
		priv:             priv,
		pub:              pub,
	}

	for access := range accesses {
		if err := ac.Grant(access); err != nil {
			ac.Close()
			return nil, err
		}
	}
	ac.priv = nil
	return ac, nil
}
func (s *accessStore) Grant(access string) error {
	if s.priv == nil {
		return errors.New("no valid privKey")
	}

	val := make([]byte, 32)
	rand.Read(val)
	sign, err := s.priv.Sign(val)
	if err != nil {
		return err
	}
	sd := &pb.SignatureData{
		Value: val,
		Sign:  sign,
	}
	msd, err := proto.Marshal(sd)
	if err != nil {
		return err
	}

	acKey := s.accessKey(access)
	if acKey == "" {
		return errors.New("accessKey generation error")
	}
	return s.IAccessBaseStore.Put(acKey, msd)
}
func (s *accessStore) accessKey(access string) string {
	sKey := PubKeyToStr(s.pub)
	if sKey == "" {
		return ""
	}
	key := &pb.AccessKey{
		MasterKey: sKey,
		Access:    access,
	}
	m, err := proto.Marshal(key)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(m)
}

func (cv *crdtVerse) loadAccessStore(st IStore, pid string, opts ...*StoreOpts) (IAccessStore, error) {
	base, ok := st.(IAccessBaseStore)
	if !ok {
		return nil, errors.New("invalid base store")
	}

	priv, _ := getAccessOpts(opts...)
	pub, err := StrToPubKey(pid)
	if err != nil {
		return nil, err
	}
	ac := &accessStore{
		IAccessBaseStore: base,
		priv:             priv,
		pub:              pub,
	}
	return ac, nil
}

func (s *accessStore) Address() string {
	name, salt, _, time, err := parseAddress(s.IAccessBaseStore.Address())
	if err != nil {
		return ""
	}

	pid := PubKeyToStr(s.pub)
	return MakeAddress(name, pid, salt, time)
}

func (s *accessStore) ResetKeyPair(priv IPrivKey, pub IPubKey) {
	if sig, ok := s.IAccessBaseStore.(ISignatureStore); ok {
		sig.ResetKeyPair(priv, pub)
	}
}

func (s *accessStore) Verify(key string) error {
	ok, err := s.verify(key)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no access permission")
	}
	return nil
}
func (s *accessStore) verify(key string) (bool, error) {
	access := s.accessFromKey(key)
	if access == "" {
		return false, errors.New("invalid access error")
	}
	acKey := s.accessKey(access)
	if acKey == "" {
		return false, errors.New("accessKey generation error")
	}

	rs, err := s.acQuery(acKey)
	if err != nil {
		return false, err
	}
	rs = query.NaiveFilter(rs, acFilter{})
	resList, err := rs.Rest()
	return len(resList) > 0, err
}

func (s *accessStore) Put(key string, val []byte) error {
	if err := s.Verify(s.putKey(key)); err != nil {
		return err
	}

	return s.IAccessBaseStore.Put(key, val)
}
func (s *accessStore) Get(key string) ([]byte, error) {
	if err := s.Verify(key); err != nil {
		if _, ok := s.IAccessBaseStore.(*hashStore); !ok {
			return nil, err
		}
		if err := s.Verify(s.putKey(key)); err != nil {
			return nil, err
		}
	}

	return s.IAccessBaseStore.Get(key)
}
func (s *accessStore) GetSize(key string) (int, error) {
	if err := s.Verify(key); err != nil {
		if _, ok := s.IAccessBaseStore.(*hashStore); !ok {
			return -1, err
		}
		if err := s.Verify(s.putKey(key)); err != nil {
			return -1, err
		}
	}

	return s.IAccessBaseStore.GetSize(key)
}
func (s *accessStore) Has(key string) (bool, error) {
	if err := s.Verify(key); err != nil {
		if _, ok := s.IAccessBaseStore.(*hashStore); !ok {
			return false, err
		}
		if err := s.Verify(s.putKey(key)); err != nil {
			return false, err
		}
	}

	return s.IAccessBaseStore.Has(key)
}
func (s *accessStore) Query(qs ...query.Query) (query.Results, error) {
	rs, err := s.IAccessBaseStore.Query()
	if err != nil {
		return nil, err
	}

	var q query.Query
	if len(qs) == 0 {
		q = query.Query{}
	} else {
		q = qs[0]
	}
	q.Filters = append(q.Filters, acVerifyFilter{ac: s})
	return query.NaiveQueryApply(q, rs), nil
}
func (s *accessStore) QueryAll(qs ...query.Query) (query.Results, error) {
	if us, ok := s.IAccessBaseStore.(IUpdatableSignatureStore); ok {
		rs, err := us.QueryAll()
		if err != nil {
			return nil, err
		}

		var q query.Query
		if len(qs) == 0 {
			q = query.Query{}
		} else {
			q = qs[0]
		}
		q.Filters = append(q.Filters, acVerifyFilter{ac: s})
		return query.NaiveQueryApply(q, rs), nil
	}

	return nil, errors.New("not implemented error")
}
