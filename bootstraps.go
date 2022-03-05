package p2pverse

import(
	//"fmt"
	"context"
	//"sync"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	kad "github.com/libp2p/go-libp2p-kad-dht"
	//p2pdiscovery "github.com/libp2p/go-libp2p-discovery"
	//bootstrap "github.com/pilinsin/go-libp2p-bootstrap"

	"io"
	"crypto/rand"
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
func NewBootstrap(h host.Host, keyword string) (*bootstrap, error){
	ctx := context.Background()
	d, err := kad.New(ctx, h)
	if err != nil{return nil, err}
	//if err := d.Bootstrap(ctx); err != nil{
	//	return nil, err
	//}

	//routingDiscovery := p2pdiscovery.NewRoutingDiscovery(d)
	//p2pdiscovery.Advertise(ctx, routingDiscovery, keyword)

	return &bootstrap{ctx, h, d}, nil
}
func (b *bootstrap) Close(){
	b.dht.Close()
}
func (b *bootstrap) AddrInfo() peer.AddrInfo{
	return HostToAddrInfo(b.h)
}

/*
type bootstrapList struct{
	bstrps []bootstrap.Bootstrap
}
func NewBootstrapList(hs []host.Host) (*bootstrapNodeList, error){
	//cfg := bootstrap.Config{}
	return nil, nil
}
*/