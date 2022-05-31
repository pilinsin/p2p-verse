package p2pverse

import (
	"context"
	"crypto/rand"
	"io"

	libp2p "github.com/libp2p/go-libp2p"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	kad "github.com/libp2p/go-libp2p-kad-dht"
)

func sliceToMap(slc []peer.AddrInfo) map[peer.ID]peer.AddrInfo {
	m := make(map[peer.ID]peer.AddrInfo)
	for _, elem := range slc {
		m[elem.ID] = elem
	}
	return m
}

type HostGenerator func(seeds ...io.Reader) (host.Host, error)

func getSeed(seeds ...io.Reader) io.Reader {
	if len(seeds) == 0 {
		return rand.Reader
	} else {
		return seeds[0]
	}
}
func SampleHost(seeds ...io.Reader) (host.Host, error) {
	seed := getSeed(seeds...)
	priv, _, _ := p2pcrypto.GenerateEd25519Key(seed)

	return libp2p.New(
		libp2p.Identity(priv),
		libp2p.DefaultSecurity,
		libp2p.ForceReachabilityPublic(),
		libp2p.EnableRelay(),
	)
}

type IBootstrap interface {
	Close()
	AddrInfo() peer.AddrInfo
	ConnectedPeers() []peer.AddrInfo
}
type bootstrap struct {
	ctx   context.Context
	h     host.Host
	dht   *kad.IpfsDHT
	peers []peer.AddrInfo
}

func NewBootstrap(hGen HostGenerator, others ...peer.AddrInfo) (IBootstrap, error) {
	h, err := hGen()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	d, err := kad.New(ctx, h)
	if err != nil {
		return nil, err
	}

	peers := make([]peer.AddrInfo, 0)
	if len(others) > 0 {
		if err := connectBootstraps(ctx, h, others); err != nil {
			d.Close()
			return nil, err
		}
		othersMap := sliceToMap(others)
		for _, pid := range h.Network().Peers() {
			if ai, ok := othersMap[pid]; ok {
				peers = append(peers, ai)
			}
		}
	}

	return &bootstrap{ctx, h, d, peers}, nil
}
func (b *bootstrap) Close() {
	b.dht.Close()
	b.h = nil
}
func (b *bootstrap) AddrInfo() peer.AddrInfo {
	return HostToAddrInfo(b.h)
}
func (b *bootstrap) ConnectedPeers() []peer.AddrInfo {
	return b.peers
}
