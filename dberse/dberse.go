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




type api struct{
	ctx context.Context
	psStore *psstore.PubsubValueStore
}
func NewDB(ctx context.Context, self host.Host, valt valType, bootstraps ...peer.AddrInfo) (*api, error){
	val, err := newValidator(valt)
	if err != nil{return nil, err}

	gossip, err := p2ppubsub.NewGossipSub(ctx, self)
	if err != nil{return nil, err}
	store, err := psstore.NewPubsubValueStore(ctx, self, gossip, val)
	if err != nil{return nil, err}

	if err := pv.Discovery(self, "dberse:,lm;eLvVfjtgoEhgg___eIeo;gje", bootstraps); err != nil{
		fmt.Println("discovery err:", err)
		return nil, err
	}else{
		return &api{ctx, store}, nil
	}
}
func (a *api) Close(){
	for _, name := range a.psStore.GetSubscriptions(){
		a.psStore.Cancel(name)
	}
}

func (a *api) Put(key string, val []byte) error{
	return a.psStore.PutValue(a.ctx, key, val)
}
func (a *api) Get(key string) ([]byte, error){
	return a.psStore.GetValue(a.ctx, key)
}
func (a *api) GetWait(ctx context.Context, key string) ([]byte, error){
	ticker := time.NewTicker(time.Second)
	for{
		select{
		case <-ticker.C:
			val, err := a.Get(key)
			if err == nil{return val, nil}
			if ok := errors.Is(err, routing.ErrNotFound); !ok{
				return nil, err
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}