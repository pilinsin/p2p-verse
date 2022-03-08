package pubsub

import(
	"context"
	"time"
	"encoding/json"
	"sort"

	host "github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	p2ppubsub "github.com/libp2p/go-libp2p-pubsub"
	pv "github.com/pilinsin/p2p-verse"
)

type filterMap map[string]map[peer.ID]struct{}
func NewFilter() filterMap{
	fm := make(filterMap)
	return fm
}
func (fm filterMap) Append(topic string, peers ...peer.ID) filterMap{
	for _, pid := range peers{
		fm[topic][pid] = struct{}{}
	}
	return fm
}
func filterFunc(filter *filterMap) p2ppubsub.PeerFilter{
	if filter == nil || len(*filter) == 0{
		return func(pid peer.ID, topic string) bool{
			return true
		}
	}
	return func(pid peer.ID, topic string) bool{
		if pm, ok := (*filter)[topic]; !ok{
			return false
		}else{
			_, ok := pm[pid]
			return len(pm) == 0 || ok
		}
	}
}

//PubSub Setup Note!!!!
//(NewMyBootstrap) -> NewHost -> NewPubSub -> Discovery -> 
// -> JoinTopic -> Subscribe -> Publish/Get

type api struct{
	ctx context.Context
	ps *p2ppubsub.PubSub
}
func NewPubSub(ctx context.Context, self host.Host, filter *filterMap, bootstraps ...peer.AddrInfo) (*api, error){
	gossip, err := p2ppubsub.NewGossipSub(ctx, self)
	if err != nil{return nil, err}
	if err := pv.Discovery(self, "pubsub:ejvoaenvaeo;vn;aeo", bootstraps); err != nil{
		return nil, err
	}else{
		return &api{ctx, gossip}, nil
	}
}
func (a *api) Topics() []string{
	return a.ps.GetTopics()
}


type room struct{
	ctx context.Context
	topic *p2ppubsub.Topic
	topicName string
	sub *p2ppubsub.Subscription
}
func (a *api) JoinTopic(topicName string) (*room, error){
	topic, err := a.ps.Join(encodeTopicName(topicName))
	if err != nil{return nil, err}
	sub, err := topic.Subscribe()
	if err != nil{return nil, err}

	return &room{a.ctx, topic, topicName, sub}, nil
}
func (r *room) Close(){
	r.sub.Cancel()
	r.topic.Close()
}
func (r *room) ListPeers() []peer.ID{
	return r.topic.ListPeers()
}

type message struct{
	data []byte
	t time.Time
}
func (m message) Data() []byte{return m.data}
func (m message) Time() time.Time{return m.t}
func (m message) marshal() ([]byte, error){
	mm := struct{
		D []byte
		T time.Time
	}{m.data, m.t}
	return json.Marshal(mm)
}
func (m *message) unmarshal(bs []byte) error{
	mm := struct{
		D []byte
		T time.Time
	}{}
	if err := json.Unmarshal(bs, &mm); err != nil{return err}

	m.data = mm.D
	m.t = mm.T
	return nil
}
func (r *room) Publish(data []byte) error{
	mes := message{data, time.Now().UTC()}
	mm, err := mes.marshal()
	if err != nil{return err}
	return r.topic.Publish(r.ctx, mm)
}

type recievedMessage struct{
	*p2ppubsub.Message
	Time time.Time
}
func convertMessage(mes *p2ppubsub.Message) (*recievedMessage, error){
	rawMes := &message{}
	if err := rawMes.unmarshal(mes.Data); err != nil{return nil, err}
	rMes := &recievedMessage{mes, rawMes.Time()}
	rMes.Data = rawMes.Data()
	return rMes, nil
}
func (r *room) Get() (*recievedMessage, error){
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mes, err := r.sub.Next(ctx)
	if err != nil{return nil, err}

	return convertMessage(mes)
}
func (r *room) GetAll() ([]*recievedMessage, error){
	mess := make([]*recievedMessage, 0)
	for{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		mes, err := r.sub.Next(ctx)
		if err != nil{
			if len(mess) > 0{
				messSort(mess)
				return mess, nil
			}else{
				return nil, err
			}
		}
		if rMes, err := convertMessage(mes); err == nil{
			mess = append(mess, rMes)
		}
	}
}

func messSort(mess []*recievedMessage){
	sort.Slice(mess, func(i, j int) bool {
    	return mess[i].Time.Before(mess[j].Time)
	})
}

func encodeTopicName(topicName string) string{
	return "pubsub topic: " + topicName
}