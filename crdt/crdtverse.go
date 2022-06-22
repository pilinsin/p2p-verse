package crdtverse

import (
	"fmt"

	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	proto "google.golang.org/protobuf/proto"

	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
	crdt "github.com/ipfs/go-ds-crdt"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	pv "github.com/pilinsin/p2p-verse"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
)

const (
	timeout string = "load error: sync timeout"
	dirLock string = "Cannot acquire directory lock on"
)

func MakeAddress(name string, timeLimits ...time.Time) string {
	tl := time.Time{}
	if len(timeLimits) > 0 {
		tl = timeLimits[0]
	}

	mt, err := tl.MarshalBinary()
	if err != nil {
		return ""
	}
	baseAddress := &pb.BaseAddress{
		Name: name,
		Time: mt,
	}
	m, err := proto.Marshal(baseAddress)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(m)
}

type crdtVerse struct {
	hGenerator pv.HostGenerator
	dirPath    string
	save       bool
	bootstraps []peer.AddrInfo
}

func NewVerse(hGen pv.HostGenerator, dir string, save bool, bootstraps ...peer.AddrInfo) *crdtVerse {
	return &crdtVerse{hGen, dir, save, bootstraps}
}

type baseStore struct {
	ctx       context.Context
	cancel    func()
	dsCancel  func()
	name      string
	timeLimit time.Time
	inTime    bool
	h         host.Host
	dht       *pv.DiscoveryDHT
	dStore    ds.Datastore
	dt        *crdt.Datastore

	cv *crdtVerse
}

func (cv *crdtVerse) initCRDT(name string, v iValidator, st *baseStore) error {
	h, err := cv.hGenerator()
	if err != nil {
		return err
	}

	dirAddr := filepath.Join(cv.dirPath, name)
	if err := os.MkdirAll(dirAddr, 0700); err != nil {
		return err
	}
	dsCancel := func() {}
	if !cv.save {
		dsCancel = func() { os.RemoveAll(dirAddr) }
	}

	ctx, cancel := context.WithCancel(context.Background())
	sp, err := cv.setupStore(ctx, h, name, v)
	if err != nil {
		cancel()
		return err
	}

	st.ctx = ctx
	st.cancel = cancel
	st.dsCancel = dsCancel
	st.name = name
	st.inTime = true
	st.h = h
	st.dht = sp.dht
	st.dStore = sp.dStore
	st.dt = sp.dt
	st.cv = cv
	return nil
}

func (cv *crdtVerse) NewStore(name, mode string, opts ...*StoreOpts) (IStore, error) {
	addrs := strings.Split(strings.TrimPrefix(name, "/"), "/")
	if len(addrs) == 0 {
		return nil, errors.New("invalid addr")
	}

	opt := &StoreOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if len(addrs) > 1 {
		ac, err := cv.loadAccessController(context.Background(), addrs[1])
		if err != nil {
			return nil, err
		}
		opt.Ac = ac
	}

	var s IStore
	stName, tl, err := parseAddress(addrs[0])
	if err != nil {
		s, err = cv.newStore(name, mode, opt)
	} else {
		opt.TimeLimit = tl
		s, err = cv.loadStore(stName, mode, opt)
	}
	if err != nil && s == nil {
		if opt.Ac != nil {
			opt.Ac.Close()
		}
		return nil, err
	}

	s.autoSync()
	return s, err
}
func parseAddress(addr string) (string, time.Time, error) {
	if addr == "" {
		return "", time.Time{}, errors.New("invalid address")
	}
	m, err := base64.URLEncoding.DecodeString(addr)
	if err != nil {
		return "", time.Time{}, err
	}

	baseAddress := &pb.BaseAddress{}
	if err := proto.Unmarshal(m, baseAddress); err != nil {
		return "", time.Time{}, err
	}

	tl := time.Time{}
	if err := tl.UnmarshalBinary(baseAddress.GetTime()); err != nil {
		return "", time.Time{}, err
	}
	return baseAddress.GetName(), tl, nil
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
func (cv *crdtVerse) newStore(name, mode string, opt *StoreOpts) (IStore, error) {
	s, err := cv.selectNewStore(name, mode, opt)
	if err != nil {
		return nil, err
	}
	if err := s.initPut(); err != nil {
		s.Close()
		return nil, err
	}

	return s, nil
}
func (cv *crdtVerse) loadStore(name, mode string, opt *StoreOpts) (IStore, error) {
	N := 3
	for i := 0; i < N; i++ {
		s, err := cv.selectNewStore(name, mode, opt)
		if err != nil {
			if strings.HasPrefix(err.Error(), dirLock) {
				fmt.Println("dirLock error, now reloading...")
				continue
			}
			return nil, err
		}

		if err := cv.loadCheck(s); err != nil {
			if strings.HasPrefix(err.Error(), timeout) {
				fmt.Println("timeout error, now reloading...")
				continue
			}
			return nil, err
		}

		return s, nil
	}

	s, err := cv.selectNewStore(name, mode, opt)
	if err != nil {
		return nil, err
	}
	if err := s.initPut(); err != nil {
		s.Close()
		return nil, err
	}
	return s, errors.New("load failed")
}
func (cv *crdtVerse) loadCheck(s IStore) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer cancel()
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
			if ok := s.loadCheck(); ok {
				return nil
			}
		}
	}
}

type iValidator interface {
	Validate(string, []byte) bool
}
type baseValidator struct {
	s IStore
}

func newBaseValidator(s IStore) iValidator {
	return &baseValidator{s}
}
func (v *baseValidator) Validate(key string, val []byte) bool {
	return v.s.isInTime()
}

type IStore interface {
	Cancel()
	Close()
	Address() string
	AddrInfo() peer.AddrInfo
	isInTime() bool
	setTimeLimit()
	Sync() error
	autoSync()
	Put(string, []byte) error
	Get(string) ([]byte, error)
	GetSize(string) (int, error)
	Has(string) (bool, error)
	Query(...query.Query) (query.Results, error)
	initPut() error
	loadCheck() bool
}

type StoreOpts struct {
	Salt      []byte
	Priv      IPrivKey
	Pub       IPubKey
	Ac        *accessController
	TimeLimit time.Time
}

func (s *baseStore) Cancel() {
	s.cancel()
	s.dt.Close()
	s.dStore.Close()
	s.dht.Close()
	s.h = nil
}
func (s *baseStore) Close() {
	if s == nil {
		return
	}
	s.Cancel()
	s.dsCancel()
}
func (s *baseStore) Address() string {
	return MakeAddress(s.name, s.timeLimit)
}
func (s *baseStore) AddrInfo() peer.AddrInfo {
	return pv.HostToAddrInfo(s.h)
}
func (s *baseStore) isInTime() bool { return s.inTime }
func (s *baseStore) setTimeLimit() {
	if !s.inTime {
		return
	}
	if s.timeLimit.Equal(time.Time{}) {
		return
	}
	if s.timeLimit.Before(time.Now()) {
		s.inTime = false
		return
	}
	go func() {
		ticker := time.NewTicker(time.Until(s.timeLimit))
		defer ticker.Stop()
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.dt.Sync(s.ctx, ds.NewKey("/"))
			s.inTime = false
		}
	}()
}

func (s *baseStore) Sync() error {
	if !s.inTime {
		return nil
	}
	return s.dt.Sync(s.ctx, ds.NewKey("/"))
}
func (s *baseStore) autoSync() {
	if !s.inTime {
		return
	}
	if err := s.dt.Sync(s.ctx, ds.NewKey("/")); err != nil {
		return
	}

	go func() {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				if !s.inTime {
					return
				}
				if err := s.dt.Sync(s.ctx, ds.NewKey("/")); err != nil {
					return
				}
			}
		}
	}()
}
func (s *baseStore) Put(key string, val []byte) error {
	exist, err := s.Has(key)
	if exist && err == nil {
		return nil
	}
	return s.dt.Put(s.ctx, ds.NewKey(key), val)
}
func (s *baseStore) Get(key string) ([]byte, error) {
	b, err := s.dt.Get(s.ctx, ds.NewKey(key))
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errors.New("no valid data")
	}
	return b, nil
}
func (s *baseStore) GetSize(key string) (int, error) {
	return s.dt.GetSize(s.ctx, ds.NewKey(key))
}
func (s *baseStore) Has(key string) (bool, error) {
	return s.dt.Has(s.ctx, ds.NewKey(key))
}
func (s *baseStore) Query(qs ...query.Query) (query.Results, error) {
	var q query.Query
	if len(qs) == 0 {
		q = query.Query{}
	} else {
		q = qs[0]
	}
	return s.dt.Query(s.ctx, q)
}

func (s *baseStore) initPut() error {
	return s.Put(s.name, []byte(s.name))
}
func (s *baseStore) loadCheck() bool {
	if !s.inTime {
		return true
	}

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
