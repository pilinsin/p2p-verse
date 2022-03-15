package p2pverse

import(
	"bytes"
	"encoding/json"

	ma "github.com/multiformats/go-multiaddr"
	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

func Marshal(objWithPublicMembers interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	err := enc.Encode(objWithPublicMembers)
	return buf.Bytes(), err
}
func Unmarshal(b []byte, objWithPublicMembers interface{}) error {
	dec := json.NewDecoder(bytes.NewBuffer(b))
	dec.DisallowUnknownFields()
	err := dec.Decode(objWithPublicMembers)
	return err
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