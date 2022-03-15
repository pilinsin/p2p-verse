package dberse

import(
	"errors"
	"context"
	"encoding/json"

	peer "github.com/libp2p/go-libp2p-core/peer"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)


func makeSignatureKey(vk p2pcrypto.PubKey) (string, error){
	id, err := peer.IDFromPublicKey(vk)
	if err != nil{return "", err}
	return id.Pretty(), nil
}
type signatureValidator struct{}
func (v signatureValidator) Validate(key0 string, val []byte) error{
	ns, _, key, err := splitKey(key0)
	if err != nil{return err}
	if ns != v.Type(){return errors.New("invalid key")}

	id, err := peer.Decode(key)
	if err != nil{return err}
	vk, err := id.ExtractPublicKey()
	if err != nil{return err}
	
	sd := struct{
		Data []byte
		Sign []byte
	}{}
	if err := json.Unmarshal(val, &sd); err != nil{return err}

	ok, err := vk.Verify(sd.Data, sd.Sign)
	if err != nil{return err}
	if !ok{return errors.New("validation error")}

	return nil
}
func (v signatureValidator) Select(key string, vals [][]byte) (int, error){
	return 0, nil
}
func (v signatureValidator) Type() string{
	return "/signature"
}


func (d *dBerse) NewSignatureStore(name string, priv p2pcrypto.PrivKey, pub p2pcrypto.PubKey) (iStore, error){
	db, err := d.newStore(name, Signature)
	if err != nil{return nil, err}
	return &signatureStore{db, priv, pub}, nil
}

type signedData struct{
	Data, Sign []byte
}
func (sd signedData) Marshal() ([]byte, error){
	return json.Marshal(sd)
}
func UnmarshalSignedData(m []byte) (*signedData, error){
	sd := &signedData{}
	err := json.Unmarshal(m, sd)
	return sd, err
}
type signatureStore struct{
	*store
	priv p2pcrypto.PrivKey
	pub p2pcrypto.PubKey
}
func (d *signatureStore) Put(val []byte, keys ...string) error{
	sign, err := d.priv.Sign(val)
	if err != nil{return err}
	sd := &signedData{val, sign}
	msd, err := sd.Marshal()
	if err != nil{return err}

	key, err := makeSignatureKey(d.pub)
	if err != nil{return err}
	return d.store.Put(msd, key)
}
func (d *signatureStore) GetRaw(key string) ([]byte, error){
	return d.store.Get(key)
}
func (d *signatureStore) Get(key string) ([]byte, error){
	msd, err := d.store.Get(key)
	if err != nil{return nil, err}
	sd, err := UnmarshalSignedData(msd)
	if err != nil{return nil, err}

	return sd.Data, nil
}
func (d *signatureStore) GetRawWait(ctx context.Context, key string) ([]byte, error){
	return d.store.GetWait(ctx, key)
}
func (d *signatureStore) GetWait(ctx context.Context, key string) ([]byte, error){
	msd, err := d.store.GetWait(ctx, key)
	if err != nil{return nil, err}
	sd, err := UnmarshalSignedData(msd)
	if err != nil{return nil, err}

	return sd.Data, nil
}
