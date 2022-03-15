package crdtverse

import(
	"strings"

	pv "github.com/pilinsin/p2p-verse"
	peer "github.com/libp2p/go-libp2p-core/peer"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type signedData struct{
	Value []byte
	Sign []byte
}
func UnmarshalSignedData(m []byte) (*signedData, error){
	sd := &signedData{}
	if err := pv.Unmarshal(m, sd); err != nil{
		return nil, err
	}
	return sd, nil
}

func makeSignatureKey(vk p2pcrypto.PubKey) (string, error){
	id, err := peer.IDFromPublicKey(vk)
	if err != nil{return "", err}
	return id.Pretty(), nil
}
type signatureValidator struct{
	updatableValidator
}
func (v *signatureValidator) Validate(key string, val []byte) bool{
	if ok := v.updatableValidator.Validate(key, val); !ok{return false}

	vKey := strings.Split(key, "/")[1]
	id, err := peer.Decode(vKey)
	if err != nil{return false}
	vk, err := id.ExtractPublicKey()
	if err != nil{return false}

	sd, err := UnmarshalSignedData(val)
	if err != nil{return false}
	ok, err := vk.Verify(sd.Value, sd.Sign)
	return err == nil && ok
}



type signatureStore struct{
	*updatableStore
	priv p2pcrypto.PrivKey
	pub p2pcrypto.PubKey
}
func (cv *crdtVerse) NewSignatureStore(name string, priv p2pcrypto.PrivKey) (*signatureStore, error){
	v := signatureValidator{updatableValidator{}}
	st, err := cv.newCRDT(name, &v)
	if err != nil{return nil, err}
	return &signatureStore{&updatableStore{st}, priv, priv.GetPublic()}, nil
}
func (s *signatureStore) Put(val []byte) error{
	sign, err := s.priv.Sign(val)
	if err != nil{return err}
	sd := &signedData{val, sign}
	msd, err := pv.Marshal(sd)
	if err != nil{return err}

	key, err := makeSignatureKey(s.pub)
	if err != nil{return err}

	return s.updatableStore.Put(key, msd)
}
func (s *signatureStore) GetRaw(key string) ([]byte, error){
	return s.updatableStore.Get(key)
}
func (s *signatureStore) Get(key string) ([]byte, error){
	msd, err := s.updatableStore.Get(key)
	sd, err := UnmarshalSignedData(msd)
	if err != nil{return nil, err}
	return sd.Value, nil
}
func (s *signatureStore) GetRawSize(key string) (int, error){
	return s.updatableStore.GetSize(key)
}
func (s *signatureStore) GetSize(key string) (int, error){
	val, err := s.Get(key)
	if err != nil{return -1, err}
	return len(val), nil
}
