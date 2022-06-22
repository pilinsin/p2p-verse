package pubsub

import (
	"context"
	"errors"
	"sort"
	"time"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	p2ppubsub "github.com/libp2p/go-libp2p-pubsub"
	pv "github.com/pilinsin/p2p-verse"
	pb "github.com/pilinsin/p2p-verse/pubsub/pb"
	proto "google.golang.org/protobuf/proto"
)

//PubSub Setup Note!!!!
//(NewMyBootstrap) -> NewHost -> NewPubSub -> Discovery ->
// -> JoinTopic -> Subscribe -> Publish/Get

const pubsubKeyword = "pubsub:ejvoaenvaeo;vn;aeo"

type IPubSub interface {
	Close()
	AddrInfo() peer.AddrInfo
	Topics() []string
	JoinTopic(string) (IRoom, error)
}
type pubSub struct {
	hGen pv.HostGenerator
	bs   []peer.AddrInfo
	h    host.Host
	dht  *pv.DiscoveryDHT
	ps   *p2ppubsub.PubSub
}

func NewPubSub(hGen pv.HostGenerator, bootstraps ...peer.AddrInfo) (IPubSub, error) {
	self, err := hGen()
	if err != nil {
		return nil, err
	}
	dht, err := pv.NewDHT(self)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	withDiscovery := p2ppubsub.WithDiscovery(dht.Discovery())
	gossip, err := p2ppubsub.NewGossipSub(ctx, self, withDiscovery)
	if err != nil {
		return nil, err
	}

	if err := dht.Bootstrap(pubsubKeyword, bootstraps); err != nil {
		return nil, err
	}
	return &pubSub{hGen, bootstraps, self, dht, gossip}, nil
}
func (ps *pubSub) Close() {
	ps.h.Close()
	ps.dht.Close()
	ps.ps = nil
}
func (ps *pubSub) AddrInfo() peer.AddrInfo {
	return pv.HostToAddrInfo(ps.h)
}
func (ps *pubSub) Topics() []string {
	return ps.ps.GetTopics()
}

type IRoom interface {
	Close()
	ListPeers() []peer.ID
	Publish([]byte) error
	Get() (*recievedMessage, error)
	GetAll() ([]*recievedMessage, error)
}
type room struct {
	ctx       context.Context
	cancel    func()
	ps        *pubSub
	topicName string
	topic     *p2ppubsub.Topic
	sub       *p2ppubsub.Subscription
}

func (ps *pubSub) JoinTopic(topicName string) (IRoom, error) {
	topic, err := ps.ps.Join(encodeTopicName(topicName))
	if err != nil {
		return nil, err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &room{ctx, cancel, ps, topicName, topic, sub}, nil
}
func (r *room) reset() error {
	N := 50
	for i := 0; i < N; i++ {
		if len(r.ListPeers()) > 0 {
			return nil
		}
		if err := r.baseReset(); err != nil {
			return err
		}
	}
	return errors.New("connection reset timeout")
}
func (r *room) baseReset() error {
	r.cancel()
	r.sub.Cancel()
	r.topic.Close()
	r.ps.Close()

	ps, err := NewPubSub(r.ps.hGen, r.ps.bs...)
	if err != nil {
		return err
	}
	r2, err := ps.JoinTopic(r.topicName)
	if err != nil {
		return err
	}
	room2 := r2.(*room)

	r.ctx = room2.ctx
	r.cancel = room2.cancel
	r.ps = room2.ps
	r.topicName = room2.topicName
	r.topic = room2.topic
	r.sub = room2.sub
	return nil
}
func (r *room) Close() {
	r.cancel()
	r.sub.Cancel()
	r.topic.Close()
	r.ps = nil
}
func (r *room) ListPeers() []peer.ID {
	return r.topic.ListPeers()
}
func (r *room) Publish(data []byte) error {
	t, _ := time.Now().UTC().MarshalBinary()
	mes := &pb.Message{
		Data: data,
		Time: t,
	}
	mm, err := proto.Marshal(mes)
	if err != nil {
		return err
	}

	if err := r.reset(); err != nil {
		return err
	}

	ready := p2ppubsub.WithReadiness(p2ppubsub.MinTopicSize(1))
	return r.topic.Publish(r.ctx, mm, ready)
}

type recievedMessage struct {
	*p2ppubsub.Message
	Time time.Time
}

func convertMessage(mes *p2ppubsub.Message) (*recievedMessage, error) {
	rawMes := &pb.Message{}
	if err := proto.Unmarshal(mes.GetData(), rawMes); err != nil {
		return nil, err
	}
	t := time.Time{}
	if err := t.UnmarshalBinary(rawMes.GetTime()); err != nil {
		return nil, err
	}
	rMes := &recievedMessage{mes, t}
	rMes.Data = rawMes.GetData()
	return rMes, nil
}
func (r *room) Get() (*recievedMessage, error) {
	if err := r.reset(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mes, err := r.sub.Next(ctx)
	select {
	case <-ctx.Done():
		return convertMessage(mes)
	default:
		if err != nil {
			return nil, err
		}
	}

	return convertMessage(mes)
}
func (r *room) GetAll() ([]*recievedMessage, error) {
	mess := make([]*recievedMessage, 0)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		mes, err := r.sub.Next(ctx)
		select {
		case <-ctx.Done():
			if len(mess) > 0 {
				messSort(mess)
			}
			return mess, nil
		default:
			if err != nil {
				return nil, err
			}
		}
		if rMes, err := convertMessage(mes); err == nil {
			mess = append(mess, rMes)
		}
	}
}

func messSort(mess []*recievedMessage) {
	sort.Slice(mess, func(i, j int) bool {
		return mess[i].Time.Before(mess[j].Time)
	})
}

func encodeTopicName(topicName string) string {
	return "pubsub topic: " + topicName
}
