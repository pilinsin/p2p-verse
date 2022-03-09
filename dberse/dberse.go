package dberse

import(
	"fmt"
	"time"
	"errors"
	"context"
	"encoding/json"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
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
func (d *dBerse) newDB(dbName string, valt ...valType) (*database, error){
	val, err := newValidator(valt...)
	if err != nil{return nil, err}
	store, err := psstore.NewPubsubValueStore(d.ctx, d.h, d.ps, val)
	if err != nil{return nil, err}

	return &database{d.ctx, store, val, dbName}, nil
}
func (d *dBerse) NewDB(dbName string) (iDatabase, error){
	return d.newDB(dbName, Simple)
}
func (d *dBerse) NewConstDB(dbName string) (iDatabase, error){
	return d.newDB(dbName, Const)
}
func (d *dBerse) NewSignatureDB(dbName string, priv p2pcrypto.PrivKey, pub p2pcrypto.PubKey) (iDatabase, error){
	db, err := d.newDB(dbName, Signature)
	if err != nil{return nil, err}
	return &signatureDatabase{db, priv, pub}, nil
}
func (d *dBerse) NewHashDB(dbName string, hash, salt []byte) (iDatabase, error){
	db, err := d.newDB(dbName, Hash)
	if err != nil{return nil, err}
	return &hashDatabase{db, hash, salt}, nil
}
func (d *dBerse) NewCidDB(dbName string) (iDatabase, error){
	db, err := d.newDB(dbName, Cid)
	if err != nil{return nil, err}
	return &cidDatabase{db}, nil
}

type iDatabase interface{
	Close()
	SelfKey() (string, error)
	Put([]byte, ...string) error
	GetRaw(string) ([]byte, error)
	Get(string) ([]byte, error)
	GetRawWait(context.Context, string) ([]byte, error)
	GetWait(context.Context, string) ([]byte, error)
}

type database struct{
	ctx context.Context
	psStore *psstore.PubsubValueStore
	validator typedValidator
	dbName string
}
func (d *database) Close(){
	for _, name := range d.psStore.GetSubscriptions(){
		d.psStore.Cancel(name)
	}
}
func (d *database) SelfKey() (string, error){
	return "", errors.New("self key does not exist")
}
func (d *database) Put(val []byte, keys ...string) error{
	if len(keys) == 0{return errors.New("invalid key")}

	key := d.validator.Type() + "/" + d.dbName + "/" + keys[0]
	return d.psStore.PutValue(d.ctx, key, val)
}
func (d *database) GetRaw(key string) ([]byte, error){
	key = d.validator.Type() + "/" + d.dbName + "/" + key
	return d.psStore.GetValue(d.ctx, key)
}
func (d *database) Get(key string) ([]byte, error){
	return d.GetRaw(key)
}
func (d *database) GetRawWait(ctx context.Context, key string) ([]byte, error){
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
func (d *database) GetWait(ctx context.Context, key string) ([]byte, error){
	return d.GetRawWait(ctx, key)
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
type signatureDatabase struct{
	*database
	priv p2pcrypto.PrivKey
	pub p2pcrypto.PubKey
}
func (d *signatureDatabase) SelfKey() (string, error){
	return makeSignatureKey(d.pub)
}
func (d *signatureDatabase) Put(val []byte, keys ...string) error{
	sign, err := d.priv.Sign(val)
	if err != nil{return err}
	sd := &signedData{val, sign}
	msd, err := sd.Marshal()
	if err != nil{return err}

	key, err := makeSignatureKey(d.pub)
	if err != nil{return err}
	return d.database.Put(msd, key)
}
func (d *signatureDatabase) GetRaw(key string) ([]byte, error){
	return d.database.Get(key)
}
func (d *signatureDatabase) Get(key string) ([]byte, error){
	msd, err := d.database.Get(key)
	if err != nil{return nil, err}
	sd, err := UnmarshalSignedData(msd)
	if err != nil{return nil, err}

	return sd.Data, nil
}
func (d *signatureDatabase) GetRawWait(ctx context.Context, key string) ([]byte, error){
	return d.database.GetWait(ctx, key)
}
func (d *signatureDatabase) GetWait(ctx context.Context, key string) ([]byte, error){
	msd, err := d.database.GetWait(ctx, key)
	if err != nil{return nil, err}
	sd, err := UnmarshalSignedData(msd)
	if err != nil{return nil, err}

	return sd.Data, nil
}


type hashedData struct{
	Data, Hash, Salt []byte
}
func (hd hashedData) Marshal() ([]byte, error){
	return json.Marshal(hd)
}
func UnmarshalHashedData(m []byte) (*hashedData, error){
	hd := &hashedData{}
	err := json.Unmarshal(m, hd)
	return hd, err
}
type hashDatabase struct{
	*database
	hash []byte
	salt []byte
}
func (d *hashDatabase) SelfKey() (string, error){
	return makeHashKey(d.hash, d.salt)
}
func (d *hashDatabase) Put(val []byte, keys ...string) error{
	hd := &hashedData{val, d.hash, d.salt}
	mhd, err := hd.Marshal()
	if err != nil{return err}

	key, _ := makeHashKey(d.hash, d.salt)
	return d.database.Put(mhd, key)
}
func (d *hashDatabase) GetRaw(key string) ([]byte, error){
	return d.database.Get(key)
}
func (d *hashDatabase) Get(key string) ([]byte, error){
	mhd, err := d.database.Get(key)
	if err != nil{return nil, err}
	hd, err := UnmarshalHashedData(mhd)
	if err != nil{return nil, err}

	return hd.Data, nil
}
func (d *hashDatabase) GetRawWait(ctx context.Context, key string) ([]byte, error){
	return d.database.GetWait(ctx, key)
}
func (d *hashDatabase) GetWait(ctx context.Context, key string) ([]byte, error){
	mhd, err := d.database.GetWait(ctx, key)
	if err != nil{return nil, err}
	hd, err := UnmarshalHashedData(mhd)
	if err != nil{return nil, err}

	return hd.Data, nil
}



type cidDatabase struct{
	*database
}
func (d *cidDatabase) Put(val []byte, keys ...string) error{
	key, err := MakeCidKey(val)
	if err != nil{return err}
	return d.database.Put(val, key)
}