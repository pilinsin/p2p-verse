package crdtverse

import (
	"errors"
	"time"

	query "github.com/ipfs/go-datastore/query"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	proto "google.golang.org/protobuf/proto"
)

func newUpdatableSignatureValidator(s IStore) iValidator {
	return &updatableValidator{newSignatureValidator(s)}
}

func getUpdatableSignatureOpts(opts ...*StoreOpts) (IPrivKey, IPubKey, time.Time) {
	if len(opts) == 0 {
		priv, pub, _ := generateKeyPair()
		return priv, pub, time.Time{}
	}
	if opts[0].Pub == nil {
		opts[0].Priv, opts[0].Pub, _ = generateKeyPair()
	}
	return opts[0].Priv, opts[0].Pub, opts[0].TimeLimit
}

type IUpdatableSignatureStore interface {
	IUpdatableStore
	ISignatureStore
}

type updatableSignatureStore struct {
	*updatableStore
	priv IPrivKey
	pub  IPubKey
}

func (cv *crdtVerse) NewUpdatableSignatureStore(name string, opts ...*StoreOpts) (IUpdatableSignatureStore, error) {
	st := &baseStore{}
	if err := cv.initCRDT(name, newUpdatableSignatureValidator(st), st); err != nil {
		return nil, err
	}

	priv, pub, tl := getUpdatableSignatureOpts(opts...)
	st.timeLimit = tl
	st.setTimeLimit()
	return &updatableSignatureStore{&updatableStore{st}, priv, pub}, nil
}

func (s *updatableSignatureStore) ResetKeyPair(priv IPrivKey, pub IPubKey) {
	if priv == nil || pub == nil {
		priv, pub, _ = generateKeyPair()
	}
	s.priv = priv
	s.pub = pub
}

func (s *updatableSignatureStore) putKey(key string) string {
	sKey := PubKeyToStr(s.pub)
	if sKey == "" {
		return ""
	}
	return sKey + "/" + key
}

func (s *updatableSignatureStore) Put(key string, val []byte) error {
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

	sKey := PubKeyToStr(s.pub)
	if sKey == "" {
		return errors.New("invalid pubKey")
	}

	key = sKey + "/" + key
	return s.updatableStore.Put(key, msd)
}
func (s *updatableSignatureStore) Get(key string) ([]byte, error) {
	msd, err := s.updatableStore.Get(key)
	if err != nil {
		return nil, err
	}
	sd := &pb.SignatureData{}
	if err := proto.Unmarshal(msd, sd); err != nil {
		return nil, err
	}
	return sd.GetValue(), nil
}
func (s *updatableSignatureStore) GetSize(key string) (int, error) {
	val, err := s.Get(key)
	if err != nil {
		return -1, err
	}
	return len(val), nil
}

func (s *updatableSignatureStore) baseQuery(q query.Query) (query.Results, error) {
	rs, err := s.updatableStore.Query(q)
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
			sd := &pb.SignatureData{}
			if err := proto.Unmarshal(r.Value, sd); err != nil {
				continue
			}

			r.Value = sd.GetValue()
			ch <- r
		}
	}()
	return query.ResultsWithChan(query.Query{}, ch), nil
}
func (s *updatableSignatureStore) Query(qs ...query.Query) (query.Results, error) {
	var q query.Query
	if len(qs) == 0 {
		q = query.Query{}
	} else {
		q = qs[0]
	}

	return s.baseQuery(q)
}

func (s *updatableSignatureStore) baseQueryAll(q query.Query) (query.Results, error) {
	rs, err := s.updatableStore.QueryAll(q)
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
			sd := &pb.SignatureData{}
			if err := proto.Unmarshal(r.Value, sd); err != nil {
				continue
			}

			r.Value = sd.GetValue()
			ch <- r
		}
	}()
	return query.ResultsWithChan(query.Query{}, ch), nil
}
func (s *updatableSignatureStore) QueryAll(qs ...query.Query) (query.Results, error) {
	var q query.Query
	if len(qs) == 0 {
		q = query.Query{}
	} else {
		q = qs[0]
	}

	return s.baseQueryAll(q)
}

func (s *updatableSignatureStore) initPut() error {
	if s.priv == nil {
		return errors.New("no valid privKey")
	}

	val := []byte(s.name)
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

	sKey := PubKeyToStr(s.pub)
	if sKey == "" {
		return errors.New("invalid pubKey")
	}

	key := sKey + "/" + s.name
	return s.updatableStore.Put(key, msd)
}
func (s *updatableSignatureStore) loadCheck() bool {
	if !s.inTime {
		return true
	}

	rs, err := s.updatableStore.Query(query.Query{
		KeysOnly: true,
		Limit:    1,
	})
	if err != nil {
		return false
	}

	resList, err := rs.Rest()
	return len(resList) > 0 && err == nil
}
