package crdtverse

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
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

type HostGenerator func(...io.Reader) (host.Host, error)

type crdtVerse struct {
	hGenerator HostGenerator
	dirPath    string
	save       bool
	useMemory  bool
	bootstraps []peer.AddrInfo
}

func NewVerse(hGen HostGenerator, dir string, save, useMemory bool, bootstraps ...peer.AddrInfo) *crdtVerse {
	return &crdtVerse{hGen, dir, save, useMemory, bootstraps}
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
	st := &logStore{ctx, cancel, dsCancel, name, sp.dht, sp.dStore, sp.dt}
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

func (cv *crdtVerse) LoadStore(addr, mode string, opts ...*StoreOpts) (IStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	for {
		select{
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			db, err := cv.baseLoadStore(addr, mode, opts...)
			if err == nil {
				return db, nil
			}
			errS := err.Error()
			fmt.Println("has prefix:", strings.HasPrefix(errS, dirLock))
			if errS == timeout || strings.HasPrefix(errS, dirLock) {
				fmt.Println(err, ", now reloading...")
				time.Sleep(time.Second * 10)

				os.RemoveAll(cv.dirPath)
				continue
			}
			if err != nil {
				return nil, err
			}
		}
	}
}
func (cv *crdtVerse) baseLoadStore(addr, mode string, opts ...*StoreOpts) (IStore, error) {
	s, err := cv.selectLoadStore(addr, mode, opts...)
	if err != nil {
		if s != nil{s.Close()}
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	ticker := time.NewTicker(time.Second * 3)
	for {
		select {
		case <-ctx.Done():
			s.Close()
			return nil, errors.New("load error: sync timeout (store)")
		case <-ticker.C:
			if err := s.Sync(); err != nil {
				s.Close()
				return nil, err
			}

			if ok := s.LoadCheck(); ok {
				return s, nil
			}
		}
	}
}
func (cv *crdtVerse) selectLoadStore(addr, mode string, opts ...*StoreOpts) (IStore, error) {
	switch mode {
	case "updatable":
		return cv.LoadUpdatableStore(addr, opts...)
	case "signature":
		return cv.LoadSignatureStore(addr, opts...)
	case "updatableSignature":
		return cv.LoadUpdatableSignatureStore(addr, opts...)
	case "hash":
		return cv.LoadHashStore(addr, opts...)
	default:
		return cv.LoadLogStore(addr, opts...)
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
	Close()
	Address() string
	Sync() error
	Repair() error
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
	dht      *pv.DiscoveryDHT
	dStore   ds.Datastore
	dt       *crdt.Datastore
}

func (cv *crdtVerse) NewLogStore(name string, _ ...*StoreOpts) (IStore, error) {
	return cv.newCRDT(name, &logValidator{})
}
func (cv *crdtVerse) LoadLogStore(addr string, _ ...*StoreOpts) (IStore, error) {
	addr = strings.Split(strings.TrimPrefix(addr, "/"), "/")[0]
	return cv.NewLogStore(addr)
}
func (s *logStore) Close() {
	s.cancel()
	s.dt.Close()
	s.dStore.Close()
	s.dht.Close()
	s.dsCancel()
}
func (s *logStore) Address() string {
	return s.name
}
func (s *logStore) Sync() error {
	return s.dt.Sync(s.ctx, ds.NewKey("/"))
}
func (s *logStore) Repair() error {
	return s.dt.Repair()
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
