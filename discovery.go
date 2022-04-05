package p2pverse

import (
	"context"
	"fmt"
	"sync"
	"time"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	p2pdiscovery "github.com/libp2p/go-libp2p-discovery"
	kad "github.com/libp2p/go-libp2p-kad-dht"
)

func Discovery(h host.Host, keyword string, bootstraps []peer.AddrInfo) error {
	ctx := context.Background()
	d, err := kad.New(ctx, h)
	if err != nil {
		return err
	}

	connectBootstraps(ctx, h, bootstraps)

	time.Sleep(5 * time.Second)

	if err := d.Bootstrap(ctx); err != nil {
		return err
	}

	routingDiscovery := p2pdiscovery.NewRoutingDiscovery(d)
	p2pdiscovery.Advertise(ctx, routingDiscovery, keyword)
	peersCh, err := routingDiscovery.FindPeers(ctx, keyword)
	if err != nil {
		return err
	}
	for peer := range peersCh {
		if peer.ID == h.ID() {
			continue
		}
		if len(peer.Addrs) <= 0 {
			continue
		}
		if err := h.Connect(ctx, peer); err != nil {
			fmt.Println("connection err:", err)
		}
	}

	time.Sleep(5 * time.Second)
	return nil
}
func connectBootstraps(ctx context.Context, self host.Host, others []peer.AddrInfo) {
	var wg sync.WaitGroup
	for _, other := range others {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := self.Connect(ctx, other); err != nil {
				fmt.Println("connection err:", err)
			}
		}()
	}
	wg.Wait()
}

type DiscoveryDHT struct {
	ctx context.Context
	h   host.Host
	d   *kad.IpfsDHT
}

func NewDHT(h host.Host) (*DiscoveryDHT, error) {
	ctx := context.Background()
	d, err := kad.New(ctx, h)
	if err != nil {
		return nil, err
	}
	return &DiscoveryDHT{ctx, h, d}, nil
}
func (d *DiscoveryDHT) Close() {
	d.d.Close()
}
func (d *DiscoveryDHT) DHT() *kad.IpfsDHT {
	return d.d
}
func (d *DiscoveryDHT) Bootstrap(keyword string, bootstraps []peer.AddrInfo) error {
	connectBootstraps(d.ctx, d.h, bootstraps)

	time.Sleep(5 * time.Second)

	if err := d.d.Bootstrap(d.ctx); err != nil {
		return err
	}

	routingDiscovery := p2pdiscovery.NewRoutingDiscovery(d.d)
	p2pdiscovery.Advertise(d.ctx, routingDiscovery, keyword)
	peersCh, err := routingDiscovery.FindPeers(d.ctx, keyword)
	if err != nil {
		return err
	}
	for peer := range peersCh {
		if peer.ID == d.h.ID() {
			continue
		}
		if len(peer.Addrs) <= 0 {
			continue
		}
		if err := d.h.Connect(d.ctx, peer); err != nil {
			fmt.Println("connection err:", err)
		}
	}

	time.Sleep(5 * time.Second)
	return nil
}
