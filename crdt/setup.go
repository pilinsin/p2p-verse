package crdtverse

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	proto "google.golang.org/protobuf/proto"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	p2ppubsub "github.com/libp2p/go-libp2p-pubsub"

	ipfslt "github.com/hsanjuan/ipfs-lite"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	crdt "github.com/ipfs/go-ds-crdt"
	crdtpb "github.com/ipfs/go-ds-crdt/pb"
	dag "github.com/ipfs/go-merkledag"

	pv "github.com/pilinsin/p2p-verse"
)

type storeParams struct {
	dht    *pv.DiscoveryDHT
	dStore ds.Datastore
	dt     *crdt.Datastore
}

func (cv *crdtVerse) setupStore(ctx context.Context, h host.Host, name string, v iValidator) (*storeParams, error) {
	dht, err := pv.NewDHT(h)
	if err != nil {
		return nil, err
	}

	dirAddr := filepath.Join(cv.dirPath, name)
	stOpts := badger.DefaultOptions
	stOpts.InMemory = false
	store, err := badger.NewDatastore(dirAddr, &stOpts)
	if err != nil {
		return nil, err
	}

	ipfs, err := ipfslt.New(ctx, store, h, dht.DHT(), nil)
	if err != nil {
		return nil, err
	}

	gossip, err := p2ppubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}
	valid := validatorFunc(h.ID(), v, store, ds.NewKey(name), ipfs)
	if err := gossip.RegisterTopicValidator(name, valid); err != nil {
		return nil, err
	}
	psbc, err := crdt.NewPubSubBroadcaster(ctx, gossip, name)
	if err != nil {
		return nil, err
	}

	opts := crdt.DefaultOptions()
	opts.RebroadcastInterval = 5 * time.Second
	dt, err := crdt.New(store, ds.NewKey(name), ipfs, psbc, opts)
	if err != nil {
		return nil, err
	}

	if err := dht.Bootstrap("crdt-keyword", cv.bootstraps); err != nil {
		return nil, err
	}
	return &storeParams{dht, store, dt}, nil
}

func validatorFunc(hid peer.ID, v iValidator, dstore ds.Datastore, ns ds.Key, dg crdt.SessionDAGService) p2ppubsub.Validator {
	return func(ctx context.Context, pid peer.ID, msg *p2ppubsub.Message) bool {
		if hid.String() == pid.String() {
			return true
		}

		deltas, err := msgToDeltas(ctx, msg, dg)
		if err != nil {
			return false
		}

		for _, delta := range deltas {
			for _, elem := range delta.Elements {
				ok := validate(elem.Key, elem.Value, v, dstore, ns)
				if !ok {
					return false
				}
			}
			//append-only
			if len(delta.Tombstones) > 0 {
				return false
			}
		}
		return true
	}
}

func validate(key string, val []byte, v iValidator, d ds.Datastore, ns ds.Key) bool {
	if ok := v.Validate(key, val); !ok {
		return false
	}

	vkey := ns.ChildString("s").ChildString("k").ChildString(key).ChildString("v")
	exist, err := d.Has(context.Background(), vkey)
	return !exist || err != nil
}

func msgToDeltas(ctx context.Context, msg *p2ppubsub.Message, dg crdt.SessionDAGService) ([]*crdtpb.Delta, error) {
	heads, err := msgToCRDTHeads(msg)
	if err != nil {
		return nil, err
	}

	ng := dg.Session(ctx)
	deltas := make([]*crdtpb.Delta, 0, len(heads))
	for _, c := range heads {
		nd, err := ng.Get(ctx, c)
		if err != nil {
			continue
		}
		prnd, ok := nd.(*dag.ProtoNode)
		if !ok {
			continue
		}
		d := &crdtpb.Delta{}
		if err := proto.Unmarshal(prnd.Data(), d); err != nil {
			continue
		}
		deltas = append(deltas, d)
	}
	return deltas, nil
}
func msgToCRDTHeads(msg *p2ppubsub.Message) ([]cid.Cid, error) {
	bcastData := crdtpb.CRDTBroadcast{}
	if err := proto.Unmarshal(msg.GetData(), &bcastData); err != nil {
		return nil, err
	}

	msgReflect := bcastData.ProtoReflect()
	if len(msgReflect.GetUnknown()) > 0 {
		c, err := cid.Cast(msgReflect.GetUnknown())
		if err != nil {
			return nil, err
		}
		return []cid.Cid{c}, nil
	}

	bcastHeads := make([]cid.Cid, 0, len(bcastData.Heads))
	for _, head := range bcastData.Heads {
		c, err := cid.Cast(head.Cid)
		if err != nil {
			continue
		}
		bcastHeads = append(bcastHeads, c)
	}
	if len(bcastHeads) == 0 {
		return nil, errors.New("invalid msg")
	}
	return bcastHeads, nil
}
