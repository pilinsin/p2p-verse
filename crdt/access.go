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
	ok, err := f.ac.Verify(e.Key)
	return ok && err == nil
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

type accessExtractor func(string) string
type putKey func(string) string

type IAccessStore interface {
	IUpdatableStore
	ISignatureStore
	Grant(string, []byte) error
	Verify(string) (bool, error)
}
type accessStore struct {
	IStore
	priv            IPrivKey
	pub             IPubKey
	accessExtractor accessExtractor
	putKey          putKey
}

func (cv *crdtVerse) NewAccessStore(st IStore, accesses <-chan string, opts ...*StoreOpts) (IAccessStore, error) {
	priv, pub := getAccessOpts(opts...)

	ext, putKey := getAccessExtractor(st)
	ac := &accessStore{
		IStore:          st,
		priv:            priv,
		pub:             pub,
		accessExtractor: ext,
		putKey:          putKey,
	}

	for access := range accesses {
		b := make([]byte, 32)
		rand.Read(b)
		if err := ac.Grant(access, b); err != nil {
			ac.Close()
			return nil, err
		}
	}
	ac.priv = nil
	return ac, nil
}
func (s *accessStore) Grant(access string, val []byte) error {
	if s.priv == nil {
		return errors.New("no valid privKey")
	}

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
	return s.IStore.Put(acKey, msd)
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
	priv, _ := getAccessOpts(opts...)
	pub, err := StrToPubKey(pid)
	if err != nil {
		return nil, err
	}

	ext, putKey := getAccessExtractor(st)
	ac := &accessStore{
		IStore:          st,
		priv:            priv,
		pub:             pub,
		accessExtractor: ext,
		putKey:          putKey,
	}
	return ac, nil
}
func getAccessExtractor(st IStore) (accessExtractor, putKey) {
	if hs, ok := st.(*hashStore); ok {
		f := func(key string) string {
			return key
		}
		return f, hs.putKey
	}

	sigExt := func(key string) string {
		keys := strings.Split(strings.TrimPrefix(key, "/"), "/")
		if len(keys) == 0 {
			return ""
		}
		return keys[0]
	}
	if ss, ok := st.(*signatureStore); ok {
		return sigExt, ss.putKey
	}
	if us, ok := st.(*updatableSignatureStore); ok {
		return sigExt, us.putKey
	}
	return func(_ string) string { return "" }, func(_ string) string { return "" }
}

func (s *accessStore) Address() string {
	name, _, time, err := parseAddress(s.IStore.Address())
	if err != nil {
		return ""
	}

	pid := PubKeyToStr(s.pub)
	return MakeAddress(name, pid, time)
}

func (s *accessStore) ResetKeyPair(priv IPrivKey, pub IPubKey) {
	if sig, ok := s.IStore.(ISignatureStore); ok {
		sig.ResetKeyPair(priv, pub)
	}
}

func (s *accessStore) Verify(key string) (bool, error) {
	access := s.accessExtractor(key)
	if access == "" {
		return false, errors.New("invalid access error")
	}
	acKey := s.accessKey(access)
	if acKey == "" {
		return false, errors.New("accessKey generation error")
	}

	rs, err := s.IStore.Query(query.Query{
		Filters: []query.Filter{KeyExistFilter{Key: acKey}},
	})
	if err != nil {
		return false, err
	}

	rs = query.NaiveFilter(rs, acFilter{})
	resList, err := rs.Rest()
	return len(resList) > 0, err
}

func (s *accessStore) Put(key string, val []byte) error {
	ok, err := s.Verify(s.putKey(key))
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no access permission")
	}

	return s.IStore.Put(key, val)
}
func (s *accessStore) Get(key string) ([]byte, error) {
	ok, err := s.Verify(key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("no access permission")
	}

	return s.IStore.Get(key)
}
func (s *accessStore) GetSize(key string) (int, error) {
	ok, err := s.Verify(key)
	if err != nil {
		return -1, err
	}
	if !ok {
		return -1, errors.New("no access permission")
	}

	return s.IStore.GetSize(key)
}
func (s *accessStore) Has(key string) (bool, error) {
	ok, err := s.Verify(key)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, errors.New("no access permission")
	}

	return s.IStore.Has(key)
}
func (s *accessStore) Query(qs ...query.Query) (query.Results, error) {
	rs, err := s.IStore.Query()
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
	if us, ok := s.IStore.(IUpdatableSignatureStore); ok {
		rs, err := us.QueryAll(qs...)
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
