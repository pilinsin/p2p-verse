package crdtverse

import(
	pv "github.com/pilinsin/p2p-verse"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)


type updatableSignatureStore struct{
	*updatableStore
	priv p2pcrypto.PrivKey
	pub p2pcrypto.PubKey
}
func (cv *crdtVerse) NewUpdatableSignatureStore(name string, priv p2pcrypto.PrivKey, pub p2pcrypto.PubKey) (*updatableSignatureStore, error){
	v := signatureValidator{&updatableValidator{}}
	st, err := cv.newCRDT(name, &v)
	if err != nil{return nil, err}
	return &updatableSignatureStore{&updatableStore{st}, priv, pub}, nil
}
func (s *updatableSignatureStore) Put(key string, val []byte) error{
	sign, err := s.priv.Sign(val)
	if err != nil{return err}
	sd := &signedData{val, sign}
	msd, err := pv.Marshal(sd)
	if err != nil{return err}

	sKey, err := makeSignatureKey(s.pub)
	if err != nil{return err}

	key = sKey + "/" + key
	return s.updatableStore.Put(key, msd)
}
func (s *updatableSignatureStore) Get(key string) ([]byte, error){
	msd, err := s.updatableStore.Get(key)
	sd, err := UnmarshalSignedData(msd)
	if err != nil{return nil, err}
	return sd.Value, nil
}
func (s *updatableSignatureStore) GetSize(key string) (int, error){
	val, err := s.Get(key)
	if err != nil{return -1, err}
	return len(val), nil
}
