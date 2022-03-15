package dberse

import(
	"fmt"
	"time"
	"errors"
	"context"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	routing "github.com/libp2p/go-libp2p-core/routing"
	pv "github.com/pilinsin/p2p-verse"
	p2ppubsub "github.com/libp2p/go-libp2p-pubsub"
	psstore "github.com/libp2p/go-libp2p-pubsub-router"
)


type dBerse struct{
	ctx context.Context
	h host.Host
	bootstraps []peer.AddrInfo
}
func NewDBerse(ctx context.Context, self host.Host, bootstraps ...peer.AddrInfo) *dBerse{
	return &dBerse{ctx, self, bootstraps}
}
func (d *dBerse) newStore(name string, valt ...valType) (*store, error){
	gossip, err := p2ppubsub.NewGossipSub(ctx, d.h)
	if err != nil{return nil, err}
	keyword := "dberse:,lm;eLvVfjtgoEhgg___eIeo;gje"
	if err := pv.Discovery(self, keyword, d.bootstraps); err != nil{
		fmt.Println("discovery err:", err)
		return nil, err
	}
	
	val, err := newValidator(valt...)
	if err != nil{return nil, err}
	pss, err := psstore.NewPubsubValueStore(d.ctx, d.h, d.ps, val)
	if err != nil{return nil, err}

	myKeys := make(map[string]struct{})

	return &store{d.ctx, pss, val, name, myKeys}, nil
}
func (d *dBerse) NewStore(name string) (iStore, error){
	return d.newStore(name, Simple)
}



type iStore interface{
	Close()
	MyKeys() map[string]struct{}
	Prefix() string
	Put([]byte, ...string) error
	GetRaw(string) ([]byte, error)
	Get(string) ([]byte, error)
	GetRawWait(context.Context, string) ([]byte, error)
	GetWait(context.Context, string) ([]byte, error)
}

type store struct{
	ctx context.Context
	psStore *psstore.PubsubValueStore
	validator typedValidator
	storeName string
	myKeys map[string]struct{}
}
func (d *store) Close(){
	for _, name := range d.psStore.GetSubscriptions(){
		if _, _, key, err := splitKey(name); err == nil{
			if _, ok := d.myKeys[key]; ok{
				delete(d.myKeys, key)
			}
		}
		d.psStore.Cancel(name)
	}
}
func (d *store) MyKeys() map[string]struct{}{
	return d.myKeys
}
func (d *store) Prefix() string{
	return d.validator.Type() + "/" + d.storeName
}
func (d *store) Put(val []byte, keys ...string) error{
	if len(keys) == 0{return errors.New("invalid key")}

	key := d.Prefix() + "/" + keys[0]
	err := d.psStore.PutValue(d.ctx, key, val)
	if err == nil{
		d.myKeys[keys[0]] = struct{}{}
	}
	return err
}
func (d *store) GetRaw(key string) ([]byte, error){
	key = d.Prefix() + "/" + key
	return d.psStore.GetValue(d.ctx, key)
}
func (d *store) Get(key string) ([]byte, error){
	return d.GetRaw(key)
}
func (d *store) GetRawWait(ctx context.Context, key string) ([]byte, error){
	ticker := time.NewTicker(time.Second)
	for{
		select{
		case <-ticker.C:
			val, err := d.Get(key)
			if err == nil{return val, nil}
			if ok := errors.Is(err, routing.ErrNotFound); !ok{
				return nil, err
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
func (d *store) GetWait(ctx context.Context, key string) ([]byte, error){
	return d.GetRawWait(ctx, key)
}
