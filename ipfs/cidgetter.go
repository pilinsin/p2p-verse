package ipfsverse

import (
	"bytes"
	"context"
	"io"
	"time"

	ipfslt "github.com/hsanjuan/ipfs-lite"
	host "github.com/libp2p/go-libp2p-core/host"

	pv "github.com/pilinsin/p2p-verse"
)

type cidGetter struct {
	h    host.Host
	dht  *pv.DiscoveryDHT
	ipfs *ipfslt.Peer
}

func NewCidGetter() (*cidGetter, error) {
	h, err := pv.SampleHost()
	if err != nil {
		return nil, err
	}
	dht, err := pv.NewDHT(h)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	store := ipfslt.NewInMemoryDatastore()
	ipfs, err := ipfslt.New(ctx, store, h, dht.DHT(), nil)
	if err != nil {
		return nil, err
	}

	return &cidGetter{h, dht, ipfs}, nil
}
func (cg *cidGetter) Close() {
	cg.dht.Close()
	cg.h.Close()
}

func (cg *cidGetter) addReader(ctx context.Context, ap *ipfslt.AddParams, r io.Reader) (string, error) {
	nd, err := cg.ipfs.AddFile(ctx, r, ap)
	if err != nil {
		return "", err
	}
	return nd.Cid().String(), nil
}
func (cg *cidGetter) newBlockCids(ctx context.Context, ap *ipfslt.AddParams, rs []io.Reader) (*blockCids, error) {
	var err error
	cids := make([]string, len(rs))
	for idx, r := range rs {
		cids[idx], err = cg.addReader(ctx, ap, r)
		if err != nil {
			return nil, err
		}
	}
	return &blockCids{cids: cids}, nil
}
func (cg *cidGetter) GetReader(r io.Reader) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	ap := &ipfslt.AddParams{
		Shard:   true,
		HashFun: "sha3-256",
	}

	rs, err := splitToBlocks(r)
	if err != nil {
		return "", err
	}
	bc, err := cg.newBlockCids(ctx, ap, rs)
	if err != nil {
		return "", err
	}
	m, err := bc.Marshal()
	if err != nil {
		return "", err
	}
	return cg.addReader(ctx, ap, bytes.NewBuffer(m))
}

func (cg *cidGetter) Get(data []byte) (string, error) {
	buf := bytes.NewBuffer(data)
	return cg.GetReader(buf)
}
