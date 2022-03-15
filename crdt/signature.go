package crdtverse

import(
	"strings"

	pv "github.com/pilinsin/p2p-verse"
	query "github.com/ipfs/go-datastore/query"
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
	validator
}
func (v *signatureValidator) Validate(key string, val []byte) bool{
	if ok := v.validator.Validate(key, val); !ok{return false}

	keys := strings.Split(strings.TrimPrefix(key, "/"), "/")
	vKey := keys[0]
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
	*logStore
	priv p2pcrypto.PrivKey
	pub p2pcrypto.PubKey
}
func (cv *crdtVerse) NewSignatureStore(name string, priv p2pcrypto.PrivKey, pub p2pcrypto.PubKey) (*signatureStore, error){
	v := signatureValidator{&logValidator{}}
	st, err := cv.newCRDT(name, &v)
	if err != nil{return nil, err}
	return &signatureStore{st, priv, pub}, nil
}
func (s *signatureStore) Put(key string, val []byte) error{
	sign, err := s.priv.Sign(val)
	if err != nil{return err}
	sd := &signedData{val, sign}
	msd, err := pv.Marshal(sd)
	if err != nil{return err}

	sKey, err := makeSignatureKey(s.pub)
	if err != nil{return err}

	key = sKey + "/" + key
	return s.logStore.Put(key, msd)
}
func (s *signatureStore) GetRaw(key string) ([]byte, error){
	rs, err := s.logStore.Query(query.Query{
		Prefix: "/"+key,
	})
	if err != nil{return nil, err}
	r := <-rs.Next()
	rs.Close()
	return r.Value, nil
}
func (s *signatureStore) Get(key string) ([]byte, error){
	msd, err := s.GetRaw(key)
	sd, err := UnmarshalSignedData(msd)
	if err != nil{return nil, err}
	return sd.Value, nil
}
func (s *signatureStore) GetRawSize(key string) (int, error){
	rs, err := s.logStore.Query(query.Query{
		Prefix: "/"+key,
		ReturnsSizes: true,
	})
	if err != nil{return -1, err}
	r := <-rs.Next()
	rs.Close()
	return r.Size, nil
}
func (s *signatureStore) GetSize(key string) (int, error){
	val, err := s.Get(key)
	if err != nil{return -1, err}
	return len(val), nil
}
func (s *signatureStore) Has(key string) (bool, error){
	rs, err := s.logStore.Query(query.Query{
		Prefix: "/"+key,
		KeysOnly: true,
		Limit: 1,
	})
	if err != nil{return false, err}
	resList, err := rs.Rest()
	rs.Close()
	return len(resList) > 0, err
}


