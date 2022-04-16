package crdtverse

import (
	"errors"

	query "github.com/ipfs/go-datastore/query"
	pv "github.com/pilinsin/p2p-verse"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	proto "google.golang.org/protobuf/proto"
)

func getUpdatableSignatureOpts(opts ...*StoreOpts) (IPrivKey, IPubKey, *accessController, *timeController) {
	if len(opts) == 0 {
		priv, pub, _ := generateKeyPair()
		return priv, pub, nil, nil
	}
	if opts[0].Pub == nil {
		opts[0].Priv, opts[0].Pub, _ = generateKeyPair()
	}
	return opts[0].Priv, opts[0].Pub, opts[0].Ac, opts[0].Tc
}

type IUpdatableSignatureStore interface {
	IUpdatableStore
	QueryWithoutTc(...query.Query) (query.Results, error)
}

type updatableSignatureStore struct {
	*updatableStore
	priv IPrivKey
	pub  IPubKey
	ac   *accessController
	tc   *timeController
}

func (cv *crdtVerse) NewUpdatableSignatureStore(name string, opts ...*StoreOpts) (IUpdatableStore, error) {
	priv, pub, ac, tc := getUpdatableSignatureOpts(opts...)

	v := signatureValidator{&updatableValidator{}}
	st, err := cv.newCRDT(name, &v)
	if err != nil {
		return nil, err
	}
	s := &updatableSignatureStore{&updatableStore{st}, priv, pub, ac, tc}
	if tc != nil {
		tc.dStore = s
		tc.AutoGrant()
	}
	return s, nil
}

func (s *updatableSignatureStore) Close() {
	if s.tc != nil {
		s.tc.Close()
	}
	if s.ac != nil {
		s.ac.Close()
	}
	s.updatableStore.Close()
}
func (s *updatableSignatureStore) Address() string {
	name := s.name
	if s.ac != nil {
		name += "/" + s.ac.Address()
	}
	if s.tc != nil {
		name += "/" + s.tc.Address()
	}
	return name
}
func (s *updatableSignatureStore) verify(key string) error {
	if s.ac != nil {
		ok, err := s.ac.Has(key)
		if !ok || err != nil {
			return errors.New("permission error")
		}
	}
	return nil
}
func (s *updatableSignatureStore) withinTime(key string) error {
	if s.tc != nil {
		//key: <pid>/<category>
		rs, err := s.baseQuery(query.Query{KeysOnly: true, Limit: 1})
		if err != nil {
			return errors.New("invalid key")
		}
		r := <-rs.Next()
		rs.Close()

		//r.Key: <pid>/<category>/<tKey>
		ok, err := s.tc.Has(r.Key)
		if !ok || err != nil {
			return errors.New("time limit error")
		}
	}
	return nil
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
	if err := s.verify(sKey); err != nil {
		return err
	}

	key = sKey + "/" + key
	return s.updatableStore.Put(key, msd)
}
func (s *updatableSignatureStore) Get(key string) ([]byte, error) {
	if err := s.verify(key); err != nil {
		return nil, err
	}
	if err := s.withinTime(key); err != nil {
		return nil, err
	}

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
	if err := s.verify(key); err != nil {
		return -1, err
	}
	if err := s.withinTime(key); err != nil {
		return -1, err
	}

	val, err := s.Get(key)
	if err != nil {
		return -1, err
	}
	return len(val), nil
}
func (s *updatableSignatureStore) Has(key string) (bool, error) {
	if err := s.verify(key); err != nil {
		return false, err
	}
	if err := s.withinTime(key); err != nil {
		return false, err
	}

	return s.updatableStore.Has(key)
}
func (s *updatableSignatureStore) baseQuery(q query.Query) (query.Results, error) {
	if s.ac != nil {
		q.Filters = append(q.Filters, acFilter{s.ac})
	}

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
	if s.tc != nil {
		q.Filters = append(q.Filters, tcFilter{s.tc})
	}

	return s.baseQuery(q)
}
func (s *updatableSignatureStore) QueryWithoutTc(qs ...query.Query) (query.Results, error) {
	var q query.Query
	if len(qs) == 0 {
		q = query.Query{}
	} else {
		q = qs[0]
	}

	return s.baseQuery(q)
}

func (s *updatableSignatureStore) baseQueryAll(q query.Query) (query.Results, error) {
	if s.ac != nil {
		q.Filters = append(q.Filters, acFilter{s.ac})
	}

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
	if s.tc != nil {
		q.Filters = append(q.Filters, tcFilter{s.tc})
	}

	return s.baseQueryAll(q)
}

func (s *updatableSignatureStore) InitPut(key string) error {
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
	return s.updatableStore.Put(key, msd)
}
func (s *updatableSignatureStore) LoadCheck() bool {
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
