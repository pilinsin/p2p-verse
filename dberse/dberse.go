package dberse

import(
	"fmt"
	"context"
	"sync"
	"errors"
	"encoding/base64"

	"golang.org/x/crypto/argon2"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	host "github.com/libp2p/go-libp2p-core/protocol"
	kad "github.com/libp2p/go-libp2p-kad-dht"
	p2pdiscovery "github.com/libp2p/go-libp2p-discovery"
	record "github.com/libp2p/go-libp2p-record"
)

type valType int
const(
	Simple valType = iota
)
type typdValidator interface{
	record.Validator
	Type() string
}
type SimpleValidator struct{}
func (v SimpleValidator) Validate(key string, val []byte) error{
	return nil
}
func (v SimpleValidator) Select(key string, vals [][]byte) (int, error){
	return 0, nil
}
func (v SimpleValidator) Type() string{
	return "simple"
}


func newExtVal(ext string, validatorTypes ...valType) (protocol.ID, record.Validator, error){
	var tVal typedValidator
	vt := Simple
	if len(validatorTypes) > 0{vt = validatorTypes[0]}
	switch vt {
	case Simple:
		tVal = &SimpleValidator{}
	default:
		return nil, nil, errors.New(fmt.Sprintln("invalid validator type", vt))
	}

	hb := argon2.IDKey(ext, tVal.Type(), 1, 64*1024, 4, 128)
	s := base64.StdEncoding.EncodeToString(hb)
	return protocol.ID{s}, tVal, nil
}


type api struct{
	ctx context.Context
	dht *kad.IpfsDHT
}
func NewDB(ctx context.Context, self host.Host, ext string, val valType, bootstraps ...peer.AddrInfo) (*api, error){
	ext2, val2, err := newExtVal(ext, val)
	if err != nil{return nil, err}

	prefixOpt := kad.ProtocolPrefix("/dberse")
	extOpt := kad.ProtocolExtension(ext2)
	valOpt := kad.Validator(val2)
	d, err := kad.New(ctx, self, prefixOpt, extOpt, valOpt)
	if err != nil{return nil, err}
	if err := d.Bootstrap(ctx); err != nil{return nil, err}

	if err := discovery(self, "i2p-dberse:,lm;eLvVfjtgoEhgg___eIeo;gje", bootstraps); err != nil{
		return nil, err
	}else{
		return &api{ctx, d}, nil
	}
}
func (a *api) Close(){
	a.d.Close()
}

func (a *api) Put(key string, val []byte) error{
	return a.d.PutValue(a.ctx, key, val)
}
func (a *api) Get(key string) ([]byte, error){
	return a.d.GetValue(a.ctx, key)
}