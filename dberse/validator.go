package dberse

import(
	"errors"
	"encoding/json"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
	peer "github.com/libp2p/go-libp2p-core/peer"
	record "github.com/libp2p/go-libp2p-record"
)

type valType int
const(
	Simple valType = iota
	Const
	Signature
	Hash
)

type simpleValidator struct{}
func (v simpleValidator) Validate(key string, val []byte) error{
	return nil
}
func (v simpleValidator) Select(key string, vals [][]byte) (int, error){
	return 0, nil
}

type constValidator struct{}
func (v constValidator) Validate(key string, val []byte) error{
	return nil
}
func (v constValidator) Select(key string, vals [][]byte) (int, error){
	if len(vals) > 1{
		return -1, nil
	}
	return 0, nil
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
id, err := peer.IDFromPublicKey(signKey)
if err != nil{return err}
key := peer.Encode(id)
psStore.Put(key, msd)
*/
type signatureValidator struct{}
func (v signatureValidator) Validate(key string, val []byte) error{
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

/*
hd := struct{
	Data []byte
	Hash []byte
	Salt []byte
}{data, userHash}
mhd, err := json.Marshal(hd)
if err != nil{return err}
uhHash := argon2.IDKey(hd.Hash, hd.Salt, 1, 64*1024, 4, 128)
key := base64.StdEncoding.EncodeToString(uhHash)
psStore.Put(key, mhd)
*/
type hashValidator struct{}
func (v hashValidator) Validate(key string, val []byte) error{
	hd := struct{
		Data []byte
		Hash []byte
		Salt []byte
	}{}
	if err := json.Unmarshal(val, &hd); err != nil{return err}

	hHash := argon2.IDKey(hd.Hash, hd.Salt, 1, 64*1024, 4, 128)
	s := base64.StdEncoding.EncodeToString(hHash)
	if key != s{return errors.New("validation error")}

	return nil
}
func (v hashValidator) Select(key string, vals [][]byte) (int, error){
	if len(vals) > 1{
		return -1, nil
	}
	return 0, nil
}


func newValidator(validatorTypes ...valType) (record.Validator, error){
	if len(validatorTypes) == 0{
		return &simpleValidator{}, nil
	}
	switch validatorTypes[0] {
	case Signature:
		return &signatureValidator{}, nil
	case Hash:
		return &hashValidator{}, nil
	case Const:
		return &constValidator{}, nil
	default:
		return &simpleValidator{}, nil
	}
}