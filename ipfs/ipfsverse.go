package ipfsverse

import(
	"context"
	"bytes"
	"io"
	"os"

	pv "github.com/pilinsin/p2p-verse"

	peer "github.com/libp2p/go-libp2p-core/peer"
	host "github.com/libp2p/go-libp2p-core/host"
	uio "github.com/ipfs/go-unixfs/io"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	ipfslt "github.com/hsanjuan/ipfs-lite"
)
type hostGenerator func(...io.Reader) (host.Host, error)

type ipfsStore struct{
	ctx context.Context
	cancel func()
	dsCancel func()
	dhtKW string
	dht *pv.DiscoveryDHT
	dStore ds.Datastore
	ipfs *ipfslt.Peer
}
func NewIpfsStore(hGen hostGenerator, dirPath, keyword string, save, useMemory bool, bootstraps ...peer.AddrInfo) (*ipfsStore, error){
	h, err := hGen()
	if err != nil{return nil, err}

	if err := os.MkdirAll(dirPath, 0700); err != nil{return nil, err}
	dsCancel := func(){}
	if !save{dsCancel = func(){os.RemoveAll(dirPath)}}

	stOpts := badger.DefaultOptions
	stOpts.InMemory = useMemory
	store, err := badger.NewDatastore(dirPath, &stOpts)
	if err != nil{return nil, err}

	dht, err := pv.NewDHT(h)
	if err != nil{return nil, err}

	ctx, cancel := context.WithCancel(context.Background())
	ipfs, err := ipfslt.New(ctx, store, h, dht.DHT(), nil)
	if err != nil{return nil, err}

	if err := dht.Bootstrap(keyword, bootstraps); err != nil{
		return nil, err
	}

	return &ipfsStore{ctx, cancel, dsCancel, keyword, dht, store, ipfs}, nil
}
func (s *ipfsStore) Close(){
	s.cancel()
	s.dStore.Close()
	s.dht.Close()
	s.dsCancel()
}

func (s *ipfsStore) AddReader(r io.Reader) (string, error){
	ap := ipfslt.AddParams{HashFun: "keccak-256"}
	nd, err := s.ipfs.AddFile(s.ctx, r, &ap)
	if err != nil{return "", err}
	return nd.Cid().String(), nil
}
func (s *ipfsStore) Add(data []byte) (string, error){
	buf := bytes.NewBuffer(data)
	return s.AddReader(buf)
}
func (s *ipfsStore) GetReader(cidStr string) (uio.ReadSeekCloser, error){
	c, err := cid.Decode(cidStr)
	if err != nil{return nil, err}
	return s.ipfs.GetFile(s.ctx, c)
}
func (s *ipfsStore) Get(cidStr string) ([]byte, error){
	r, err := s.GetReader(cidStr)
	if err != nil{return nil, err}

	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}
func (s *ipfsStore) Has(cidStr string) (bool, error){
	c, err := cid.Decode(cidStr)
	if err != nil{return false, err}
	return s.ipfs.HasBlock(s.ctx, c)
}