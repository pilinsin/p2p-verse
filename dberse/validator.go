package dberse

import(
	"strings"
	"errors"

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