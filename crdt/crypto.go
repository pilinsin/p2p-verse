package crdtverse

import(
	"errors"
	peer "github.com/libp2p/go-libp2p-core/peer"
	mh "github.com/multiformats/go-multihash"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type GenerateKeyPair func() (IPrivKey, IPubKey, error)
type MarshalPubKey func(IPubKey) ([]byte, error)
type UnmarshalPubKey func([]byte) (IPubKey, error)

var generateKeyPair GenerateKeyPair
var marshalPubKey MarshalPubKey
var unmarshalPubKey UnmarshalPubKey

func init(){
	generateKeyPair = func()(IPrivKey, IPubKey, error){
		return p2pcrypto.GenerateEd25519Key(nil)
	}
	marshalPubKey = func(pub IPubKey)([]byte, error){
		p2ppub, ok := pub.(p2pcrypto.PubKey)
		if !ok{return nil, errors.New("invalid pubKey")}
		return p2pcrypto.MarshalPublicKey(p2ppub)
	}
	unmarshalPubKey = func(m []byte)(IPubKey, error){
		return p2pcrypto.UnmarshalPublicKey(m)
	}
}
func InitCryptoFuncs(gkp GenerateKeyPair, mpu MarshalPubKey, upu UnmarshalPubKey){
	generateKeyPair = gkp
	marshalPubKey = mpu
	unmarshalPubKey = upu
}

type IKey interface{
	Raw() ([]byte, error)
}
type IPubKey interface{
	IKey
	Verify(data, sig []byte) (bool, error)
}
type IPrivKey interface{
	IKey
	Sign([]byte) ([]byte, error)
}

func idFromPubKey(pub IPubKey) (peer.ID, error){
	b, err := marshalPubKey(pub)
	if err != nil{return "", err}

	hash, _ := mh.Sum(b, mh.IDENTITY, -1)
	return peer.ID(hash), nil
}

func extractPubKey(pid peer.ID) (IPubKey, error){
	decoded, err := mh.Decode([]byte(pid))
	if err != nil{return nil, err}
	if decoded.Code != mh.IDENTITY{
		return nil, peer.ErrNoPublicKey
	}

	return unmarshalPubKey(decoded.Digest)
}
