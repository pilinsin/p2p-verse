package bootstrapstore

import (
	//"context"
	"encoding/base64"
	"errors"

	//"fmt"
	//"strings"
	//"time"

	query "github.com/ipfs/go-datastore/query"
	peer "github.com/libp2p/go-libp2p-core/peer"
	kad "github.com/libp2p/go-libp2p-kad-dht"
	"golang.org/x/crypto/argon2"

	pv "github.com/pilinsin/p2p-verse"
	crdt "github.com/pilinsin/p2p-verse/crdt"
)

var bStoreName string

func init() {
	name := "bootstrap store"
	salt := "{storeAddress: boottrap AddrInfo}"
	hash := argon2.IDKey([]byte(name), []byte(salt), 1, 64*1024, 4, 32)
	bStoreName = base64.URLEncoding.EncodeToString(hash)
}

func stAddrToKey(stAddr string) string {
	hash := argon2.IDKey([]byte(stAddr), []byte(bStoreName), 1, 64*1024, 4, 55)
	return base64.URLEncoding.EncodeToString(hash)
}

type IBootstrapStore interface {
	Close()
	Address() string
	Put(string, string) error
	Get(string) ([]peer.AddrInfo, error)
}
type bootstrapStore struct {
	store crdt.IUpdatableSignatureStore
}

func NewBootstrapStore(dir string) (IBootstrapStore, error) {
	p2pBsAddrInfos := kad.GetDefaultBootstrapPeerAddrInfos()
	addr := crdt.MakeAddress(bStoreName, "", nil)

	cv := crdt.NewVerse(pv.SampleHost, dir, false, p2pBsAddrInfos...)
	tmp, err := cv.NewStore(addr, "updatableSignature")
	if err != nil && tmp == nil {
		return nil, err
	}
	st := tmp.(crdt.IUpdatableSignatureStore)
	return &bootstrapStore{st}, nil
}

func (bs *bootstrapStore) Close() {
	bs.store.Close()
}
func (bs *bootstrapStore) Address() string {
	return bs.store.Address()
}
func (bs *bootstrapStore) Put(stAddr, bAddr string) error {
	ai := pv.AddrInfoFromString(bAddr)
	if ai.ID == "" || len(ai.Addrs) == 0 {
		ais := pv.AddrInfosFromString(bAddr)
		if len(ais) == 0 {
			return errors.New("invalid bootstrap address")
		}
	}

	key := stAddrToKey(stAddr)
	val, err := base64.URLEncoding.DecodeString(bAddr)
	if err != nil {
		return err
	}
	return bs.store.Put(key, val)
}
func (bs *bootstrapStore) Get(stAddr string) ([]peer.AddrInfo, error) {
	kef := crdt.KeyExistFilter{Key: stAddrToKey(stAddr)}
	rs, err := bs.store.Query(query.Query{
		Filters: []query.Filter{kef},
	})
	if err != nil {
		return nil, err
	}

	ais := make([]peer.AddrInfo, 0)
	for res := range rs.Next() {
		bAddr := base64.URLEncoding.EncodeToString(res.Value)

		ai := pv.AddrInfoFromString(bAddr)
		if ai.ID != "" && len(ai.Addrs) > 0 {
			ais = append(ais, ai)
			continue
		}

		tmp := pv.AddrInfosFromString(bAddr)
		if len(tmp) == 0 {
			ais = append(ais, tmp...)
			continue
		}
	}

	return ais, nil
}
