package p2pverse

import(
	ma "github.com/multiformats/go-multiaddr"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

func AddrInfo(pid peer.ID, maddrs ...ma.Multiaddr) peer.AddrInfo{
	return peer.AddrInfo{
		ID: pid,
		Addrs: maddrs,
	}
}
func HostToAddrInfo(h host.Host) peer.AddrInfo{
	return peer.AddrInfo{
		ID: h.ID(),
		Addrs: h.Addrs(),
	}
}