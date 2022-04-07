package p2pverse

import (
	"crypto/rand"
	"encoding/base64"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"

	pb "github.com/pilinsin/p2p-verse/pb"
	proto "google.golang.org/protobuf/proto"
)

func RandBytes(bSize int) []byte {
	bs := make([]byte, bSize)
	rand.Read(bs)
	return bs
}
func RandString(bSize int) string {
	bs := make([]byte, bSize)
	rand.Read(bs)
	return base64.URLEncoding.EncodeToString(bs)
}

func AddrInfo(pid peer.ID, maddrs ...ma.Multiaddr) peer.AddrInfo {
	return peer.AddrInfo{
		ID:    pid,
		Addrs: maddrs,
	}
}
func HostToAddrInfo(h host.Host) peer.AddrInfo {
	return peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
}

func peerToPb(ai peer.AddrInfo) *pb.AddrInfo {
	addrs := make([][]byte, len(ai.Addrs))
	for idx, addr := range ai.Addrs {
		addrs[idx] = addr.Bytes()
	}
	return &pb.AddrInfo{
		ID:    ai.ID.String(),
		Addrs: addrs,
	}
}
func pbToPeer(mai *pb.AddrInfo) (peer.AddrInfo, error) {
	id, err := peer.Decode(mai.GetID())
	if err != nil {
		return peer.AddrInfo{}, err
	}
	addrs := make([]ma.Multiaddr, len(mai.GetAddrs()))
	for idx, m := range mai.GetAddrs() {
		addr, err := ma.NewMultiaddrBytes(m)
		if err != nil {
			return peer.AddrInfo{}, err
		}
		addrs[idx] = addr
	}

	return peer.AddrInfo{
		ID:    id,
		Addrs: addrs,
	}, nil
}

func AddrInfoToString(ai peer.AddrInfo) string {
	mai := peerToPb(ai)
	m, err := proto.Marshal(mai)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(m)
}
func AddrInfoFromString(aiStr string) peer.AddrInfo {
	m, err := base64.URLEncoding.DecodeString(aiStr)
	if err != nil {
		return peer.AddrInfo{}
	}

	mai := &pb.AddrInfo{}
	if err := proto.Unmarshal(m, mai); err != nil {
		return peer.AddrInfo{}
	}

	ai, _ := pbToPeer(mai)
	return ai
}

func AddrInfosToString(ais ...peer.AddrInfo) string {
	mais := make([]*pb.AddrInfo, len(ais))
	for idx, ai := range ais {
		mais[idx] = peerToPb(ai)
	}
	mais2 := &pb.AddrInfos{AddrInfos: mais}
	m, err := proto.Marshal(mais2)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(m)
}
func AddrInfosFromString(sais string) []peer.AddrInfo {
	m, err := base64.URLEncoding.DecodeString(sais)
	if err != nil {
		return nil
	}

	mais := &pb.AddrInfos{}
	if err := proto.Unmarshal(m, mais); err != nil {
		return nil
	}

	ais := make([]peer.AddrInfo, 0)
	for _, mai := range mais.AddrInfos {
		ai, err := pbToPeer(mai)
		if err != nil {
			continue
		}
		ais = append(ais, ai)
	}
	return ais
}
