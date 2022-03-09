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

//validatorごとにdataBaseも作成する

type dBerse struct{
	ctx context.Context
	h host.Host
	ps *p2ppubsub.PubSub
}
func NewDBerse(ctx context.Context, self host.Host, bootstraps ...peer.AddrInfo) (*dBerse, error){
	gossip, err := p2ppubsub.NewGossipSub(ctx, self)
	if err != nil{return nil, err}
	if err := pv.Discovery(self, "dberse:,lm;eLvVfjtgoEhgg___eIeo;gje", bootstraps); err != nil{
		fmt.Println("discovery err:", err)
		return nil, err
	}else{
		return &dBerse{ctx, self, gossip}, nil
	}
}

type database struct{
	ctx context.Context
	psStore *psstore.PubsubValueStore
	validator typedValidator
	dbName string
}
func (d *dBerse) NewDB(dbName string, valt ...valType) (*database, error){
	val, err := newValidator(valt...)
	if err != nil{return nil, err}
	store, err := psstore.NewPubsubValueStore(d.ctx, d.h, d.ps, val)
	if err != nil{return nil, err}

	return &database{d.ctx, store, val, dbName}, nil
}
func (d *database) Close(){
	for _, name := range d.psStore.GetSubscriptions(){
		d.psStore.Cancel(name)
	}
}

func (d *database) Put(key string, val []byte) error{
	key = d.validator.Type() + "/" + d.dbName + "/" + key
	return d.psStore.PutValue(d.ctx, key, val)
}
func (d *database) Get(key string) ([]byte, error){
	key = d.validator.Type() + "/" + d.dbName + "/" + key
	return d.psStore.GetValue(d.ctx, key)
}
func (d *database) GetWait(ctx context.Context, key string) ([]byte, error){
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