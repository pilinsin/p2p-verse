package ipfsverse

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"time"

	pv "github.com/pilinsin/p2p-verse"
	"github.com/pilinsin/p2p-verse/ipfs/pb"
	"google.golang.org/protobuf/proto"

	ipfslt "github.com/hsanjuan/ipfs-lite"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	uio "github.com/ipfs/go-unixfs/io"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

var defaultDuration = time.Minute * 5

func getDurationFromDurations(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return defaultDuration
	}
	return durations[0]
}

func splitToBlocks(r io.Reader) ([]io.Reader, error) {
	rs := make([]io.Reader, 0)
	for {
		//256KiB
		bs := make([]byte, 1<<18)
		n, err := r.Read(bs)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if n > 0 {
			buf := bytes.NewBuffer(bs[:n])
			rs = append(rs, buf)
		}

		if err == io.EOF {
			return rs, nil
		}
	}
}
func mergeFromBlocks(rs []uio.ReadSeekCloser) (io.Reader, error) {
	buf := &bytes.Buffer{}
	for _, r := range rs {
		if _, err := r.WriteTo(buf); err != nil {
			return nil, err
		}
	}
	return bytes.NewReader(buf.Bytes()), nil
}

type blockCids struct {
	cids []string
}

func (bc *blockCids) Marshal() ([]byte, error) {
	pbBlockCids := &pb.BlockCids{
		Cids: bc.cids,
	}
	return proto.Marshal(pbBlockCids)
}
func (bc *blockCids) Unmarshal(m []byte) error {
	pbBlockCids := &pb.BlockCids{}
	if err := proto.Unmarshal(m, pbBlockCids); err != nil {
		return err
	}
	bc.cids = pbBlockCids.GetCids()
	return nil
}

type Ipfs interface {
	Close()
	AddrInfo() peer.AddrInfo
	AddReader(io.Reader, ...time.Duration) (string, error)
	Add([]byte, ...time.Duration) (string, error)
	GetReader(string, ...time.Duration) (io.Reader, error)
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
func (s *ipfsStore) addReader(ctx context.Context, ap *ipfslt.AddParams, r io.Reader) (string, error) {
	nd, err := s.ipfs.AddFile(ctx, r, ap)
	if err != nil {
		return "", err
	}
	return nd.Cid().String(), nil
}
func (s *ipfsStore) newBlockCids(ctx context.Context, ap *ipfslt.AddParams, rs []io.Reader) (*blockCids, error) {
	var err error
	cids := make([]string, len(rs))
	for idx, r := range rs {
		cids[idx], err = s.addReader(ctx, ap, r)
		if err != nil {
			return nil, err
		}
	}
	return &blockCids{cids: cids}, nil
}
func (s *ipfsStore) AddReader(r io.Reader, timeouts ...time.Duration) (string, error) {
	timeout := getDurationFromDurations(timeouts)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ap := &ipfslt.AddParams{
		Shard:   true,
		HashFun: "sha3-256",
	}

	rs, err := splitToBlocks(r)
	if err != nil {
		return "", err
	}
	bc, err := s.newBlockCids(ctx, ap, rs)
	if err != nil {
		return "", err
	}
	m, err := bc.Marshal()
	if err != nil {
		return "", err
	}
	return s.addReader(ctx, ap, bytes.NewBuffer(m))
}
func (s *ipfsStore) Add(data []byte, timeouts ...time.Duration) (string, error) {
	buf := bytes.NewBuffer(data)
	return s.AddReader(buf, timeouts...)
}

func (s *ipfsStore) getReader(ctx context.Context, cidStr string) (uio.ReadSeekCloser, error) {
	c, err := cid.Decode(cidStr)
	if err != nil {
		return nil, err
	}
	return s.ipfs.GetFile(ctx, c)
}
func (s *ipfsStore) getReadersFromBlockCids(ctx context.Context, bc *blockCids) ([]uio.ReadSeekCloser, error) {
	var err error
	rs := make([]uio.ReadSeekCloser, len(bc.cids))
	for idx, cid := range bc.cids {
		rs[idx], err = s.getReader(ctx, cid)
		if err != nil {
			return nil, err
		}
	}
	return rs, nil
}
func (s *ipfsStore) GetReader(cidStr string, timeouts ...time.Duration) (io.Reader, error) {
	timeout := getDurationFromDurations(timeouts)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	bcr, err := s.getReader(ctx, cidStr)
	if err != nil {
		return nil, err
	}
	m, err := io.ReadAll(bcr)
	if err != nil {
		return nil, err
	}
	bc := &blockCids{}
	if err := bc.Unmarshal(m); err != nil {
		return nil, err
	}
	rs, err := s.getReadersFromBlockCids(ctx, bc)
	if err != nil {
		return nil, err
	}
	return mergeFromBlocks(rs)
}
func (s *ipfsStore) Get(cidStr string, timeouts ...time.Duration) ([]byte, error) {
	r, err := s.GetReader(cidStr, timeouts...)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(r)
}
func (s *ipfsStore) Has(cidStr string, timeouts ...time.Duration) (bool, error) {
	c, err := cid.Decode(cidStr)
	if err != nil {
		return false, err
	}

	timeout := getDurationFromDurations(timeouts)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	has, err := s.ipfs.HasBlock(ctx, c)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return false, err
	}
	return has, nil
}
