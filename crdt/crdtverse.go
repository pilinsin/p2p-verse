package crdtverse

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"

	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
	crdt "github.com/ipfs/go-ds-crdt"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	pv "github.com/pilinsin/p2p-verse"
)

const (
	timeout string = "load error: sync timeout"
	dirLock string = "Cannot acquire directory lock on"
)

type crdtVerse struct {
	hGenerator pv.HostGenerator
	dirPath    string
	save       bool
	bootstraps []peer.AddrInfo
}

func NewVerse(hGen pv.HostGenerator, dir string, save bool, bootstraps ...peer.AddrInfo) *crdtVerse {
	return &crdtVerse{hGen, dir, save, bootstraps}
}
func (cv *crdtVerse) newCRDT(name string, v iValidator) (*logStore, error) {
	h, err := cv.hGenerator()
	if err != nil {
		return nil, err
	}

	dirAddr := filepath.Join(cv.dirPath, name)
	if err := os.MkdirAll(dirAddr, 0700); err != nil {
		return nil, err
	}
	dsCancel := func() {}
	if !cv.save {
		dsCancel = func() { os.RemoveAll(dirAddr) }
	}

	ctx, cancel := context.WithCancel(context.Background())
	sp, err := cv.setupStore(ctx, h, name, v)
	if err != nil {
		return nil, err
	}
	st := &logStore{ctx, cancel, dsCancel, name, h, sp.dht, sp.dStore, sp.dt}
	return st, nil
}

func (cv *crdtVerse) NewStore(name, mode string, opts ...*StoreOpts) (IStore, error) {
	exmpl := pv.RandString(32)
	hash := argon2.IDKey([]byte(name), []byte(exmpl), 1, 64*1024, 4, 32)
	name = base64.URLEncoding.EncodeToString(hash)

	s, err := cv.selectNewStore(name, mode, opts...)
	if err != nil {
		return nil, err
	}

	if err := s.InitPut(exmpl); err != nil {
		s.Close()
		return nil, err
	}

	return s, nil
}
func (cv *crdtVerse) selectNewStore(name, mode string, opts ...*StoreOpts) (IStore, error) {
	switch mode {
	case "updatable":
		return cv.NewUpdatableStore(name, opts...)
	case "signature":
		return cv.NewSignatureStore(name, opts...)
	case "updatableSignature":
		return cv.NewUpdatableSignatureStore(name, opts...)
	case "hash":
		return cv.NewHashStore(name, opts...)
	default:
		return cv.NewLogStore(name, opts...)
	}
}

func (cv *crdtVerse) LoadStore(ctx context.Context, addr, mode string, opts ...*StoreOpts) (IStore, error) {
	addrs := strings.Split(strings.TrimPrefix(addr, "/"), "/")
	if len(addrs) == 0 {
		return nil, errors.New("invalid addr")
	}

	opt := &StoreOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if len(addrs) > 1 {
		ac, err := cv.loadAccessController(ctx, addrs[1])
		if err != nil {
			return nil, err
		}
		opt.Ac = ac
	}
	if len(addrs) > 2 {
		tc, err := cv.loadTimeController(ctx, addrs[2])
		if err != nil && opt.Ac != nil {
			opt.Ac.Close()
			return nil, err
		}
		opt.Tc = tc
	}

	s, err := cv.baseLoadStore(ctx, addrs[0], mode, opt)
	if err != nil {
		if opt.Ac != nil {
			opt.Ac.Close()
		}
		if opt.Tc != nil {
			opt.Tc.Close()
		}
		return nil, err
	}
	s.autoSync()
	return s, nil
}

func (cv *crdtVerse) baseLoadStore(ctx context.Context, addr, mode string, opts ...*StoreOpts) (IStore, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			db, err := cv.selectNewStore(addr, mode, opts...)
			if err != nil {
				if strings.HasPrefix(err.Error(), dirLock) {
					fmt.Println(err, ", now reloading...")
					continue
				}
				return nil, err
			}

			err = cv.loadCheck(db)
			if err == nil {
				return db, nil
			}
			if strings.HasPrefix(err.Error(), timeout) {
				fmt.Println(err, ", now reloading...")
				continue
			}
			return nil, err
		}
	}
}
func (cv *crdtVerse) loadCheck(s IStore) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			s.Cancel()
			return errors.New("load error: sync timeout (store)")
		case <-ticker.C:
			if err := s.Sync(); err != nil {
				s.Close()
				return err
			}

			if ok := s.LoadCheck(); ok {
				return nil
			}
		}
	}
}

type iValidator interface {
	Validate(string, []byte) bool
}
type logValidator struct{}

func (v *logValidator) Validate(key string, val []byte) bool {
	return true
}

type IStore interface {
	Cancel()
	Close()
	Address() string
	AddrInfo() peer.AddrInfo
	autoSync()
	Sync() error
	Put(string, []byte) error
	Get(string) ([]byte, error)
	GetSize(string) (int, error)
	Has(string) (bool, error)
	Query(...query.Query) (query.Results, error)
	InitPut(string) error
	LoadCheck() bool
}

type StoreOpts struct {
	Salt []byte
	Priv IPrivKey
	Pub  IPubKey
	Ac   *accessController
	Tc   *timeController
}

type logStore struct {
	ctx      context.Context
	cancel   func()
	dsCancel func()
	name     string
	h        host.Host
	dht      *pv.DiscoveryDHT
	dStore   ds.Datastore
	dt       *crdt.Datastore
}

func (cv *crdtVerse) NewLogStore(name string, _ ...*StoreOpts) (IStore, error) {
	return cv.newCRDT(name, &logValidator{})
}

func (s *logStore) Cancel() {
	s.cancel()
	s.dt.Close()
	s.dStore.Close()
	s.dht.Close()
	s.h = nil
}
func (s *logStore) Close() {
	s.Cancel()
	s.dsCancel()
}
func (s *logStore) Address() string {
	return s.name
}
func (s *logStore) AddrInfo() peer.AddrInfo {
	return pv.HostToAddrInfo(s.h)
}
func (s *logStore) Sync() error{
	return s.dt.Sync(s.ctx, ds.NewKey("/"))	
}
func (s *logStore) autoSync() {
	//AutoSync interval is 5s (= Rebroadcast interval)
	ticker := time.NewTicker(time.Second*5)
	go func(){
		defer ticker.Stop()
		select{
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.dt.Sync(s.ctx, ds.NewKey("/")); err != nil{return}
		}
	}()
}
func (s *logStore) Put(key string, val []byte) error {
	exist, err := s.Has(key)
	if exist && err == nil {
		return nil
	}
	return s.dt.Put(s.ctx, ds.NewKey(key), val)
}
func (s *logStore) Get(key string) ([]byte, error) {
	return s.dt.Get(s.ctx, ds.NewKey(key))
}
func (s *logStore) GetSize(key string) (int, error) {
	return s.dt.GetSize(s.ctx, ds.NewKey(key))
}
func (s *logStore) Has(key string) (bool, error) {
	return s.dt.Has(s.ctx, ds.NewKey(key))
}
func (s *logStore) Query(qs ...query.Query) (query.Results, error) {
	var q query.Query
	if len(qs) == 0 {
		q = query.Query{}
	} else {
		q = qs[0]
	}
	return s.dt.Query(s.ctx, q)
}

func (s *logStore) InitPut(key string) error {
	return s.Put(key, pv.RandBytes(8))
}
func (s *logStore) LoadCheck() bool {
	rs, err := s.Query(query.Query{
		KeysOnly: true,
		Limit:    1,
	})
	if err != nil {
		return false
	}
	resList, err := rs.Rest()
	return len(resList) > 0 && err == nil
}
