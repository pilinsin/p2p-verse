package p2pverse

import(
	"crypto/rand"
	"encoding/base64"
	"encoding/json"

	ma "github.com/multiformats/go-multiaddr"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
)
func RandBytes(bSize int) []byte{
	bs := make([]byte, bSize)
	rand.Read(bs)
	return bs
}
func RandString(bSize int) string{
	bs := make([]byte, bSize)
	rand.Read(bs)
	return base64.URLEncoding.EncodeToString(bs)
}

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

func AddrInfoToString(ai peer.AddrInfo) string{
	m, _ := json.Marshal(ai)
	return string(m)
}
func AddrInfoFromString(aiStr string) peer.AddrInfo{
	ai := peer.AddrInfo{}
	if err := json.Unmarshal([]byte(aiStr), &ai); err != nil{
		return peer.AddrInfo{}
	}
	return ai
}