package crdtverse

import (
	"errors"
	"strings"

	query "github.com/ipfs/go-datastore/query"
	peer "github.com/libp2p/go-libp2p-core/peer"
	pv "github.com/pilinsin/p2p-verse"
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

func getSignatureOpts(opts ...*StoreOpts) (IPrivKey, IPubKey, *accessController) {
	if len(opts) == 0 {
		priv, pub, _ := generateKeyPair()
		return priv, pub, nil
	}
	if opts[0].Pub == nil {
		opts[0].Priv, opts[0].Pub, _ = generateKeyPair()
	}
	return opts[0].Priv, opts[0].Pub, opts[0].Ac
}

type ISignatureStore interface{
	IStore
	ResetKeyPair(IPrivKey, IPubKey)
}

type signatureStore struct {
	*logStore
	priv IPrivKey
	pub  IPubKey
	ac   *accessController
}

func (cv *crdtVerse) NewSignatureStore(name string, opts ...*StoreOpts) (ISignatureStore, error) {
	priv, pub, ac := getSignatureOpts(opts...)

	v := signatureValidator{&logValidator{}}
	st, err := cv.newCRDT(name, &v)
	if err != nil {
		return nil, err
	}
	return &signatureStore{st, priv, pub, ac}, nil
}

func (s *signatureStore) Close() {
	if s.ac != nil {
		s.ac.Close()
	}
	s.logStore.Close()
}
func (s *signatureStore) Cancel() {
	s.logStore.Cancel()
}
func (s *signatureStore) Address() string {
	name := s.name
	if s.ac != nil {
		name += "/" + s.ac.Address()
	}
	return name
}
func (s *signatureStore) ResetKeyPair(priv IPrivKey, pub IPubKey){
	if priv == nil || pub == nil{
		priv, pub, _ = generateKeyPair()
	}
	s.priv = priv
	s.pub = pub
}
func (s *signatureStore) verify(key string) error {
	if s.ac != nil {
		ok, err := s.ac.Has(key)
		if !ok || err != nil {
			return errors.New("permission error")
		}
	}
	return nil
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
	if err := s.verify(sKey); err != nil {
		return err
	}

	key = sKey + "/" + key
	return s.logStore.Put(key, msd)
}
func (s *signatureStore) Get(key string) ([]byte, error) {
	if err := s.verify(key); err != nil {
		return nil, err
	}

	msd, err := s.logStore.Get(key)
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
func (s *signatureStore) Has(key string) (bool, error) {
	if err := s.verify(key); err != nil {
		return false, err
	}
	return s.logStore.Has(key)
}
func (s *signatureStore) Query(qs ...query.Query) (query.Results, error) {
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

func (s *signatureStore) InitPut(key string) error {
	if s.priv == nil {
		return errors.New("no valid privKey")
	}

	val := pv.RandBytes(8)
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
	return s.logStore.Put(key, msd)
}
func (s *signatureStore) LoadCheck() bool {
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
