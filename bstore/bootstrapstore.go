package bootstrapstore

import(
	"fmt"
	"context"
	"time"
	"errors"
	"strings"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
	kad "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p-core/peer"
	query "github.com/ipfs/go-datastore/query"

	pv "github.com/pilinsin/p2p-verse"
	crdt "github.com/pilinsin/p2p-verse/crdt"
)

const (
	timeout string = "load error: sync timeout"
	dirLock string = "Cannot acquire directory lock on"
)

var bStoreName string
func init(){
	name := "bootstrap store"
	salt := "{storeAddress: boottrap AddrInfo}"
	hash := argon2.IDKey([]byte(name), []byte(salt), 1, 64*1024, 4, 32)
	bStoreName = base64.URLEncoding.EncodeToString(hash)
}

func stAddrToKey(stAddr string) string{
	hash := argon2.IDKey([]byte(stAddr), []byte(bStoreName), 1, 64*1024, 4, 55)
	return base64.URLEncoding.EncodeToString(hash)
}


type IBootstrapStore interface{
	Close()
	Put(string, string) error
	Get(string) ([]peer.AddrInfo, error)
}
type bootstrapStore struct{
	store crdt.IUpdatableSignatureStore
}
func NewBootstrapStore(dir string) (IBootstrapStore, error){
	p2pBsAddrInfos := kad.GetDefaultBootstrapPeerAddrInfos()

	cv := crdt.NewVerse(pv.SampleHost, dir, false, p2pBsAddrInfos...)
	st, err := cv.NewUpdatableSignatureStore(bStoreName)
	if err != nil{return nil, err}

	exmpl := pv.RandString(32)
	if err := st.InitPut(exmpl); err != nil {
		st.Close()
		return nil, err
	}

	bs := &bootstrapStore{st}
	bs.store.AutoSync()
	return bs, nil
}
func LoadBootstrapStore(dir string) (IBootstrapStore, error){
	st, err := baseLoadBootstrapStore(dir)
	if err != nil{return nil, err}

	bs := &bootstrapStore{st}
	bs.store.AutoSync()
	return bs, nil
}
func baseLoadBootstrapStore(dir string) (crdt.IUpdatableSignatureStore, error){
	p2pBsAddrInfos := kad.GetDefaultBootstrapPeerAddrInfos()
	for{
		cv := crdt.NewVerse(pv.SampleHost, dir, false, p2pBsAddrInfos...)
		st, err := cv.NewUpdatableSignatureStore(bStoreName)
		if err != nil {
			if strings.HasPrefix(err.Error(), dirLock) {
				fmt.Println(err, ", now reloading...")
				continue
			}
			return nil, err
		}

		err = loadCheck(st)
		if err == nil {
			return st, nil
		}
		if strings.HasPrefix(err.Error(), timeout) {
			fmt.Println(err, ", now reloading...")
			continue
		}
		return nil, err
	}
}
func loadCheck(st crdt.IUpdatableSignatureStore) error{
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			st.Cancel()
			return errors.New("load error: sync timeout (bStore)")
		case <-ticker.C:
			if err := st.Sync(); err != nil {
				st.Close()
				return err
			}

			if ok := st.LoadCheck(); ok {
				return nil
			}
		}
	}
}
func (bs *bootstrapStore) Close(){
	bs.store.Close()
}

func (bs *bootstrapStore) Put(stAddr, bAddr string) error{
	ai := pv.AddrInfoFromString(bAddr)
	if ai.ID == "" || len(ai.Addrs) == 0 {
		ais := pv.AddrInfosFromString(bAddr)
		if len(ais) == 0{
			return errors.New("invalid bootstrap address")
		}
	}

	key := stAddrToKey(stAddr)
	val, err := base64.URLEncoding.DecodeString(bAddr)
	if err != nil{return err}
	return bs.store.Put(key, val)
}
func (bs *bootstrapStore) Get(stAddr string) ([]peer.AddrInfo, error){
	kef := crdt.KeyExistFilter{stAddrToKey(stAddr)}
	rs, err := bs.store.Query(query.Query{
		Filters: []query.Filter{kef},
	})
	if err != nil{return nil, err}

	ais := make([]peer.AddrInfo, 0)
	for res := range rs.Next(){
		bAddr := base64.URLEncoding.EncodeToString(res.Value)
		
		ai := pv.AddrInfoFromString(bAddr)
		if ai.ID != "" && len(ai.Addrs) > 0{
			ais = append(ais, ai)
			continue
		}

		tmp := pv.AddrInfosFromString(bAddr)
		if len(tmp) == 0{
			ais = append(ais, tmp...)
			continue
		}
	}

	return ais, nil
}
