package crdtverse

import (
	"errors"
	"strings"
	"time"

	query "github.com/ipfs/go-datastore/query"
	peer "github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	proto "google.golang.org/protobuf/proto"
)

func PubKeyToStr(vk IPubKey) string {
	id, err := idFromPubKey(vk)
	if err != nil {
		return ""
	}
	return id.Pretty()
}
func StrToPubKey(s string) (IPubKey, error) {
	id, err := peer.Decode(s)
	if err != nil {
		return nil, err
	}
	vk, err := extractPubKey(id)
	if err != nil {
		return nil, err
	}
	return vk, nil
}

type signatureValidator struct {
	iValidator
}

func newSignatureValidator(s IStore) iValidator {
	return &signatureValidator{newBaseValidator(s)}
}
func (v *signatureValidator) Validate(key string, val []byte) bool {
	if ok := v.iValidator.Validate(key, val); !ok {
		return false
	}

	keys := strings.Split(strings.TrimPrefix(key, "/"), "/")
	vk, err := StrToPubKey(keys[0])
	if err != nil {
		return false
	}

	sd := &pb.SignatureData{}
	if err := proto.Unmarshal(val, sd); err != nil {
		return false
	}
	ok, err := vk.Verify(sd.GetValue(), sd.GetSign())
	return err == nil && ok
}

func getSignatureOpts(opts ...*StoreOpts) (IPrivKey, IPubKey, time.Time) {
	if len(opts) == 0 {
		priv, pub, _ := generateKeyPair()
		return priv, pub, time.Time{}
	}
	if opts[0].Pub == nil {
		opts[0].Priv, opts[0].Pub, _ = generateKeyPair()
	}
	return opts[0].Priv, opts[0].Pub, opts[0].TimeLimit
}

type ISignatureStore interface {
	IStore
	ResetKeyPair(IPrivKey, IPubKey)
}

type signatureStore struct {
	*baseStore
	priv IPrivKey
	pub  IPubKey
}

func (cv *crdtVerse) NewSignatureStore(name string, opts ...*StoreOpts) (ISignatureStore, error) {
	st := &baseStore{}
	if err := cv.initCRDT(name, newSignatureValidator(st), st); err != nil {
		return nil, err
	}

	priv, pub, tl := getSignatureOpts(opts...)
	st.timeLimit = tl
	st.setTimeLimit()
	return &signatureStore{st, priv, pub}, nil
}

func (s *signatureStore) ResetKeyPair(priv IPrivKey, pub IPubKey) {
	if priv == nil || pub == nil {
		priv, pub, _ = generateKeyPair()
	}
	s.priv = priv
	s.pub = pub
}
func (s *signatureStore) putKey(key string) string {
	sKey := PubKeyToStr(s.pub)
	if sKey == "" {
		return ""
	}
	return sKey + "/" + key
}

func (s *signatureStore) Put(key string, val []byte) error {
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
	return s.baseStore.Put(key, msd)
}
func (s *signatureStore) Get(key string) ([]byte, error) {
	msd, err := s.baseStore.Get(key)
	if err != nil {
		return nil, err
	}
	sd := &pb.SignatureData{}
	if err := proto.Unmarshal(msd, sd); err != nil {
		return nil, err
	}
	return sd.GetValue(), nil
}
func (s *signatureStore) GetSize(key string) (int, error) {
	val, err := s.Get(key)
	if err != nil {
		return -1, err
	}
	return len(val), nil
}

func (s *signatureStore) Query(qs ...query.Query) (query.Results, error) {
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

func (s *signatureStore) initPut() error {
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
	return s.baseStore.Put(key, msd)
}
func (s *signatureStore) loadCheck() bool {
	if !s.inTime {
		return true
	}

	rs, err := s.baseStore.Query(query.Query{
		KeysOnly: true,
		Limit:    1,
	})
	if err != nil {
		return false
	}

	resList, err := rs.Rest()
	return len(resList) > 0 && err == nil
}
