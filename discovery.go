package p2pverse

import (
	"context"
	"errors"
	"fmt"
	"sync"

	discovery "github.com/libp2p/go-libp2p-core/discovery"
	host "github.com/libp2p/go-libp2p-core/host"
	network "github.com/libp2p/go-libp2p-core/network"
	peer "github.com/libp2p/go-libp2p-core/peer"
	kad "github.com/libp2p/go-libp2p-kad-dht"
	routing "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	discutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

func Discovery(h host.Host, keyword string, bootstraps []peer.AddrInfo) error {
	ctx := context.Background()
	d, err := kad.New(ctx, h)
	if err != nil {
		return err
	}

	if err := connectBootstraps(ctx, h, bootstraps); err != nil {
		return err
	}
	if err := d.Bootstrap(ctx); err != nil {
		return err
	}

	routingDiscovery := routing.NewRoutingDiscovery(d)
	discutil.Advertise(ctx, routingDiscovery, keyword)
	peersCh, err := routingDiscovery.FindPeers(ctx, keyword)
	if err != nil {
		return err
	}
	nSuccess := 0
	maxSuccess := 5
	for peer := range peersCh {
		if nSuccess >= maxSuccess {
			return nil
		}

		if peer.ID == h.ID() {
			continue
		}
		if len(peer.Addrs) <= 0 {
			continue
		}
		if h.Network().Connectedness(peer.ID) == network.Connected {
			nSuccess++
			continue
		}
		if err := h.Connect(ctx, peer); err != nil {
			fmt.Println("connection err:", err)
		}
		nSuccess++
	}

	return nil
}
func connectBootstraps(ctx context.Context, self host.Host, others []peer.AddrInfo) error {
	var cbErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		isSuccess := false
		for _, other := range others {
			if self.Network().Connectedness(other.ID) == network.Connected {
				isSuccess = true
				continue
			}
			if err := self.Connect(ctx, other); err == nil {
				isSuccess = true
			} else {
				fmt.Println("connection err:", err)
			}
		}
		if !isSuccess {
			cbErr = errors.New("no bootstraps are connected")
		}
	}()
	wg.Wait()

	return cbErr
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
	d.h = nil
}
func (d *DiscoveryDHT) DHT() *kad.IpfsDHT {
	return d.d
}
func (d *DiscoveryDHT) Discovery() discovery.Discovery {
	return routing.NewRoutingDiscovery(d.d)
}
func (d *DiscoveryDHT) Bootstrap(keyword string, bootstraps []peer.AddrInfo) error {
	if err := connectBootstraps(d.ctx, d.h, bootstraps); err != nil {
		return err
	}
	if err := d.d.Bootstrap(d.ctx); err != nil {
		return err
	}

	routingDiscovery := routing.NewRoutingDiscovery(d.d)
	discutil.Advertise(d.ctx, routingDiscovery, keyword)
	peersCh, err := routingDiscovery.FindPeers(d.ctx, keyword)
	if err != nil {
		return err
	}

	nSuccess := 0
	maxSuccess := 5
	for peer := range peersCh {
		if nSuccess >= maxSuccess {
			return nil
		}

		if peer.ID == d.h.ID() {
			continue
		}
		if len(peer.Addrs) <= 0 {
			continue
		}
		if d.h.Network().Connectedness(peer.ID) == network.Connected {
			nSuccess++
			continue
		}
		if err := d.h.Connect(d.ctx, peer); err != nil {
			fmt.Println("connection err:", err)
		}
		nSuccess++
	}

	return nil
}
