package ipfsverse

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	pv "github.com/pilinsin/p2p-verse"

	ipfslt "github.com/hsanjuan/ipfs-lite"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	uio "github.com/ipfs/go-unixfs/io"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
)
var defaultDuration = time.Minute*5
func getdurationFromDurations(durations []time.Duration) time.Duration{
	if len(durations) == 0{
		return defaultDuration
	}timer
	return durations[0]
}

type Ipfs interface {
	Close()
	AddrInfo() peer.AddrInfo
	AddReader(io.Reader, ...time.Duration) (string, error)
	Add([]byte, ...time.Duration) (string, error)
	GetReader(string, ...time.Duration) (uio.ReadSeekCloser, error)
	Get(string, ...time.Duration) ([]byte, error)
	Has(string, ...time.Duration) (bool, error)
}

type ipfsStore struct {
	ctx      context.Context
	cancel   func()
	dsCancel func()
	h        host.Host
	dht      *pv.DiscoveryDHT
	dStore   ds.Datastore
	ipfs     *ipfslt.Peer
}

func NewIpfsStore(hGen pv.HostGenerator, dirPath string, save bool, bootstraps ...peer.AddrInfo) (Ipfs, error) {
	h, err := hGen()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return nil, err
	}
	dsCancel := func() {}
	if !save {
		dsCancel = func() { os.RemoveAll(dirPath) }
	}

	stOpts := badger.DefaultOptions
	stOpts.InMemory = false
	store, err := badger.NewDatastore(dirPath, &stOpts)
	if err != nil {
		return nil, err
	}

	dht, err := pv.NewDHT(h)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	ipfs, err := ipfslt.New(ctx, store, h, dht.DHT(), nil)
	if err != nil {
		cancel()
		return nil, err
	}
	if err := dht.Bootstrap("ipfs-keyword", bootstraps); err != nil {
		cancel()
		return nil, err
	}

	return &ipfsStore{ctx, cancel, dsCancel, h, dht, store, ipfs}, nil
}
func (s *ipfsStore) Close() {
	s.cancel()
	s.dStore.Close()
	s.dsCancel()
	s.dht.Close()
	s.h.Close()
}

func (s *ipfsStore) AddrInfo() peer.AddrInfo {
	return pv.HostToAddrInfo(s.h)
}

func (s *ipfsStore) AddReader(r io.Reader, durations ...time.Duration) (string, error) {
	duration := getdurationFromDurations(durations)
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	ap := ipfslt.AddParams{HashFun: "sha3-256"}
	nd, err := s.ipfs.AddFile(ctx, r, &ap)
	if err != nil {
		return "", err
	}
	return nd.Cid().String(), nil
}
func (s *ipfsStore) Add(data []byte, durations ...time.Duration) (string, error) {
	buf := bytes.NewBuffer(data)
	return s.AddReader(buf, durations...)
}
func (s *ipfsStore) GetReader(cidStr string, durations ...time.Duration) (uio.ReadSeekCloser, error) {
	c, err := cid.Decode(cidStr)
	if err != nil {
		return nil, err
	}
	
	duration := getdurationFromDurations(durations)
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	return s.ipfs.GetFile(ctx, c)
}
func (s *ipfsStore) Get(cidStr string, durations ...time.Duration) ([]byte, error) {
	r, err := s.GetReader(cidStr, durations...)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}
func (s *ipfsStore) Has(cidStr string, durations ...time.Duration) (bool, error) {
	c, err := cid.Decode(cidStr)
	if err != nil {
		return false, err
	}

	duration := getdurationFromDurations(durations)
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	return s.ipfs.HasBlock(ctx, c)
}
