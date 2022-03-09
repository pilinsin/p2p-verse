package p2pverse

import(
	"fmt"
	"time"
	"context"
	"sync"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	kad "github.com/libp2p/go-libp2p-kad-dht"
	p2pdiscovery "github.com/libp2p/go-libp2p-discovery"
)

func Discovery(h host.Host, keyword string, bootstraps []peer.AddrInfo) error{
	ctx := context.Background()
	d, err := kad.New(ctx, h)
	if err != nil{return err}

	connectBootstraps(ctx, h, bootstraps)

	<-time.Tick(5*time.Second)
	
	if err := d.Bootstrap(ctx); err != nil{
		return err
	}

	routingDiscovery := p2pdiscovery.NewRoutingDiscovery(d)
	p2pdiscovery.Advertise(ctx, routingDiscovery, keyword)
	peersCh, err := routingDiscovery.FindPeers(ctx, keyword)
	if err != nil{return err}
	for peer := range peersCh{
		if peer.ID == h.ID(){
			continue
		}
		if len(peer.Addrs) <= 0{
			continue
		}
		if err := h.Connect(ctx, peer); err != nil{
			fmt.Println("connection err:", err)
		}
	}

	//fmt.Println("dht listPeers:", d.RoutingTable().ListPeers())
	<-time.Tick(5*time.Second)
	return nil
}

func connectBootstraps(ctx context.Context, self host.Host, others []peer.AddrInfo){
	var wg sync.WaitGroup
	for _, other := range others{
		wg.Add(1)
		go func(){
			defer wg.Done()
			if err := self.Connect(ctx, other); err != nil{
				fmt.Println("connection err:", err)
			}
		}()
	}
	wg.Wait()
}
