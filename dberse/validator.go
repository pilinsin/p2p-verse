package dberse

import(
	"strings"
	"errors"
	"encoding/json"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	peer "github.com/libp2p/go-libp2p-core/peer"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	record "github.com/libp2p/go-libp2p-record"
)

func splitKey(nsKey string) (string, string, string, error){
	nsKey = strings.TrimPrefix(nsKey, "/")
	names := strings.Split(nsKey, "/")
	if len(names) != 3{return "", "", "", errors.New("invalid key")}
	ns, nm, key := names[0], names[1], names[2]
	return "/" + ns, nm, key, nil
}

type valType int
const(
	Simple valType = iota
	Const
	Signature
	Hash
	Cid
)
var strToValType = map[string]valType{
	"/simple"   : Simple,
	"/const"    : Const,
	"/signature": Signature,
	"/hash"     : Hash,
	"/cid"      : Cid,
}

type typedValidator interface{
	record.Validator
	Type() string
}

type simpleValidator struct{}
func (v simpleValidator) Validate(key0 string, val []byte) error{
	ns, _, _, err := splitKey(key0)
	if err != nil{return err}
	if ns != v.Type(){return errors.New("invalid key")}
	return nil
}
func (v simpleValidator) Select(key string, vals [][]byte) (int, error){
	return 0, nil
}
func (v simpleValidator) Type() string{
	return "/simple"
}

type constValidator struct{}
func (v constValidator) Validate(key0 string, val []byte) error{
	ns, _, _, err := splitKey(key0)
	if err != nil{return err}
	if ns != v.Type(){return errors.New("invalid key")}
	return nil
}
func (v constValidator) Select(key string, vals [][]byte) (int, error){
	if len(vals) > 1{
		return -1, nil
	}
	return 0, nil
}
func (v constValidator) Type() string{
	return "/const"
}

/*
sign, err := signKey.Sign(data)
if err != nil{return err}
sd := struct{
	Data []byte
	Sign []byte
}{data, sign}
msd, err := json.Marshal(sd)
if err != nil{return err}
id, err := peer.IDFromPublicKey(verfKey)
if err != nil{return err}
key := peer.Encode(id)
Put(key, msd)
*/
func MakeSignatureKey(vk p2pcrypto.PubKey) (string, error){
	id, err := peer.IDFromPublicKey(vk)
	if err != nil{return "", err}
	key := peer.Encode(id)
	return key, nil
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

/*
hd := struct{
	Data []byte
	Hash []byte
	Salt []byte
}{data, hash. salt}
mhd, err := json.Marshal(hd)
if err != nil{return err}
key, _ := MakeHashKey(hash, salt)
Put(key, mhd)
*/
func MakeHashKey(hash, salt []byte) (string, error){
	uhHash := argon2.IDKey(hash, salt, 1, 64*1024, 4, 128)
	key := base64.StdEncoding.EncodeToString(uhHash)
	return key, nil
}
type hashValidator struct{}
func (v hashValidator) Validate(key0 string, val []byte) error{
	ns, _, key, err := splitKey(key0)
	if err != nil{return err}
	if ns != v.Type(){return errors.New("invalid key")}

	hd := struct{
		Data []byte
		Hash []byte
		Salt []byte
	}{}
	if err := json.Unmarshal(val, &hd); err != nil{return err}

	s, _ := MakeHashKey(hd.Hash, hd.Salt)
	if key != s{return errors.New("validation error")}

	return nil
}
func (v hashValidator) Select(key string, vals [][]byte) (int, error){
	if len(vals) > 1{
		return -1, nil
	}
	return 0, nil
}
func (v hashValidator) Type() string{
	return "/hash"
}

/*
cid, err := MakeCidKey(data)
if err != nil{return err}
Put(cid, data)
*/
func MakeCidKey(val []byte) (string, error){
	format := cid.V1Builder{Codec: cid.Codecs["cbor"], MhType: mh.SHA3}
	c, err := format.Sum(val)
	if err != nil{return "", err}
	return c.String(), nil
}
type cidValidator struct{}
func (v cidValidator) Validate(key0 string, val []byte) error{
	ns, _, key, err := splitKey(key0)
	if err != nil{return err}
	if ns != v.Type(){return errors.New("invalid key")}

	c, err := MakeCidKey(val)
	if err != nil{return err}
	
	if key != c{return errors.New("invalid key")}
	return nil	
}
func (v cidValidator) Select(key string, vals [][]byte) (int, error){
	if len(vals) > 1{
		return -1, nil
	}
	return 0, nil
}
func (v cidValidator) Type() string{
	return "/cid"
}


func newValidator(validatorTypes ...valType) (typedValidator, error){
	if len(validatorTypes) == 0{
		return &simpleValidator{}, nil
	}
	switch validatorTypes[0] {
	case Signature:
		return &signatureValidator{}, nil
	case Hash:
		return &hashValidator{}, nil
	case Cid:
		return &cidValidator{}, nil
	case Const:
		return &constValidator{}, nil
	default:
		return &simpleValidator{}, nil
	}
}