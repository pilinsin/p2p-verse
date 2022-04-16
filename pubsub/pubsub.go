package pubsub

import (
	"context"
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


type api struct {
	ctx context.Context
	h host.Host
	ps  *p2ppubsub.PubSub
}

func NewPubSub(hGen pv.HostGenerator, bootstraps ...peer.AddrInfo) (*api, error) {
	self, err := hGen()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	gossip, err := p2ppubsub.NewGossipSub(ctx, self)
	if err != nil {
		return nil, err
	}
	if err := pv.Discovery(self, "pubsub:ejvoaenvaeo;vn;aeo", bootstraps); err != nil {
		return nil, err
	} else {
		return &api{ctx, self, gossip}, nil
	}
}
func (a *api) Close(){
	a.ps = nil
	a.h = nil
}
func (a *api) Topics() []string {
	return a.ps.GetTopics()
}

type room struct {
	ctx       context.Context
	topic     *p2ppubsub.Topic
	topicName string
	sub       *p2ppubsub.Subscription
}

func (a *api) JoinTopic(topicName string) (*room, error) {
	topic, err := a.ps.Join(encodeTopicName(topicName))
	if err != nil {
		return nil, err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	return &room{a.ctx, topic, topicName, sub}, nil
}
func (r *room) Close() {
	r.sub.Cancel()
	r.topic.Close()
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
	return r.topic.Publish(r.ctx, mm)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mes, err := r.sub.Next(ctx)
	if err != nil {
		return nil, err
	}

	return convertMessage(mes)
}
func (r *room) GetAll() ([]*recievedMessage, error) {
	mess := make([]*recievedMessage, 0)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		mes, err := r.sub.Next(ctx)
		if err != nil {
			if len(mess) > 0 {
				messSort(mess)
				return mess, nil
			} else {
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
