package p2pverse

import(
	"context"
	"io"
	"crypto/rand"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	kad "github.com/libp2p/go-libp2p-kad-dht"
	libp2p "github.com/libp2p/go-libp2p"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

func getSeed(seeds ...io.Reader) io.Reader{
	if len(seeds) == 0{
		return rand.Reader
	}else{
		return seeds[0]
	}
}
func SampleHost(seeds ...io.Reader) (host.Host, error){
	seed := getSeed(seeds...)
	priv, _, _ := p2pcrypto.GenerateEd25519Key(seed)

	return libp2p.New(
		libp2p.Identity(priv),
		libp2p.DefaultSecurity,
		libp2p.ForceReachabilityPublic(),
		libp2p.EnableRelay(),
	)
}

type bootstrap struct{
	ctx context.Context
	h host.Host
	dht *kad.IpfsDHT
}
func NewBootstrap(h host.Host) (*bootstrap, error){
	ctx := context.Background()
	d, err := kad.New(ctx, h)
	if err != nil{return nil, err}

	return &bootstrap{ctx, h, d}, nil
}
func (b *bootstrap) Close(){
	b.dht.Close()
}
func (b *bootstrap) AddrInfo() peer.AddrInfo{
	return HostToAddrInfo(b.h)
}
