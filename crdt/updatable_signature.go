package crdtverse

import(
	"errors"
	"crypto/rand"
	"strings"

	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	proto "google.golang.org/protobuf/proto"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	query "github.com/ipfs/go-datastore/query"
)

func getUpdatableSignatureOpts(opts ...*StoreOpts) (p2pcrypto.PrivKey, p2pcrypto.PubKey, *accessController, *timeController){
	if len(opts) == 0{
		priv, pub, _ := p2pcrypto.GenerateEd25519Key(rand.Reader)
		return priv, pub, nil, nil
	}
	if opts[0].Priv == nil || opts[0].Pub == nil{
		opts[0].Priv, opts[0].Pub, _ = p2pcrypto.GenerateEd25519Key(rand.Reader)
	}
	return opts[0].Priv, opts[0].Pub, opts[0].Ac, opts[0].Tc
}

type updatableSignatureStore struct{
	*updatableStore
	priv p2pcrypto.PrivKey
	pub p2pcrypto.PubKey
	ac *accessController
	tc *timeController
}
func (cv *crdtVerse) NewUpdatableSignatureStore(name string, opts ...*StoreOpts) (iStore, error){
	priv, pub, ac, tc := getUpdatableSignatureOpts(opts...)

	v := signatureValidator{&updatableValidator{}}
	st, err := cv.newCRDT(name, &v)
	if err != nil{return nil, err}
	s := &updatableSignatureStore{&updatableStore{st}, priv, pub, ac, tc}
	if tc != nil{
		tc.dStore = s
		tc.AutoGrant()
	}
	return s, nil
}
func (cv *crdtVerse) LoadUpdatableSignatureStore(addr string, opts ...*StoreOpts) (iStore, error){
	addrs := strings.Split(strings.TrimPrefix(addr, "/"), "/")
	s, err := cv.NewUpdatableSignatureStore(addrs[0], opts...)
	if err != nil{return nil, err}
	if len(addrs) >= 2{
		ac, err := cv.LoadAccessController(addrs[1])
		if err != nil{return nil, err}
		s.(*updatableSignatureStore).ac = ac
		if len(addrs) >= 3 && len(opts) > 0{
			opts[0].Ac = ac
			tc, err := cv.LoadTimeController(addrs[2], opts...)
			if err != nil{return nil, err}
			s.(*updatableSignatureStore).tc = tc
			tc.dStore = s.(*updatableSignatureStore)
			tc.AutoGrant()
		}
	}
	return s, nil
}
func (s *updatableSignatureStore) Close(){
	if s.tc != nil{s.tc.Close()}
	if s.ac != nil{s.ac.Close()}
	s.updatableStore.Close()
}
func (s *updatableSignatureStore) Address() string{
	name := s.name
	if s.ac != nil{name += "/" + s.ac.Address()}
	if s.tc != nil{name += "/" + s.tc.Address()}
	return name
}
func (s *updatableSignatureStore) verify(key string) error{
	if s.ac != nil{
		ok, err := s.ac.Has(key)
		if !ok || err != nil{
			return errors.New("permission error")
		}
	}
	return nil
}
func (s *updatableSignatureStore) withinTime(key string) error{
	if s.tc != nil{
		ok, err := s.tc.Has(key)
		if !ok || err != nil{
			return errors.New("time limit error")
		}
	}
	return nil
}
func (s *updatableSignatureStore) Put(key string, val []byte) error{
	sign, err := s.priv.Sign(val)
	if err != nil{return err}
	sd := &pb.SignatureData{
		Value: val,
		Sign: sign,
	}
	msd, err := proto.Marshal(sd)
	if err != nil{return err}

	sKey := PubKeyToStr(s.pub)
	if sKey == ""{return errors.New("invalid pubKey")}
	if err := s.verify(sKey); err != nil{return err}

	key = sKey + "/" + key
	return s.updatableStore.Put(key, msd)
}
func (s *updatableSignatureStore) Get(key string) ([]byte, error){
	if err := s.verify(key); err != nil{return nil, err}
	if err := s.withinTime(key); err != nil{return nil, err}

	msd, err := s.updatableStore.Get(key)
	if err != nil{return nil, err}
	sd := &pb.SignatureData{}
	if err := proto.Unmarshal(msd, sd); err != nil{return nil, err}
	return sd.GetValue(), nil
}
func (s *updatableSignatureStore) GetSize(key string) (int, error){
	if err := s.verify(key); err != nil{return -1, err}
	if err := s.withinTime(key); err != nil{return -1, err}

	val, err := s.Get(key)
	if err != nil{return -1, err}
	return len(val), nil
}
func (s *updatableSignatureStore) Has(key string) (bool, error){
	if err := s.verify(key); err != nil{return false, err}
	if err := s.withinTime(key); err != nil{return false, err}

	return s.updatableStore.Has(key)
}
func (s *updatableSignatureStore) baseQuery(q query.Query) (query.Results, error){
	if s.ac != nil{
		q.Filters = append(q.Filters, acFilter{s.ac})
	}

	return s.updatableStore.Query(q)
}
func (s *updatableSignatureStore) Query(qs ...query.Query) (query.Results, error){
	var q query.Query
	if len(qs) == 0{
		q = query.Query{}
	}else{
		q = qs[0]
	}
	if s.tc != nil{
		q.Filters = append(q.Filters, tcFilter{s.tc})
	}

	return s.baseQuery(q)
}