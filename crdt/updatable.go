package crdtverse

import(
	"time"
	"strings"
	"encoding/base64"

	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
)

//a<b:-1, a==b:0, a>b:1
type updatableOrder struct{}
func (o updatableOrder) Compare(a, b query.Entry) int{
	takey := strings.Split(a.Key, "/")[2]
	ma, err := base64.StdEncoding.DecodeString(takey)
	if err != nil{return 1}
	tbkey := strings.Split(b.Key, "/")[2]
	mb, err := base64.StdEncoding.DecodeString(tbkey)
	if err != nil{return -1}

	ta := time.Time{}
	if err := ta.UnmarshalJSON(ma); err != nil{return 1}
	tb := time.Time{}
	if err := tb.UnmarshalJSON(mb); err != nil{return -1}

	if ok := ta.UTC().Equal(tb.UTC()); ok{return 0}
	if ok := ta.UTC().After(tb.UTC()); ok{return -1}
	return 1
}


type updatableValidator struct{}
func (v *updatableValidator) Validate(key string, val []byte) bool{
	tKey := strings.Split(key, "/")[2]
	tb, err := base64.StdEncoding.DecodeString(tKey)
	if err != nil{return false}

	t := time.Time{}
	if err := t.UnmarshalJSON(tb); err != nil{return false}
	return t.Location().String() == time.UTC.String()
}
func (v *updatableValidator) Select(key string, vals [][]byte) bool{
	return len(vals) == 1
}


type updatableStore struct{
	*logStore
}
func (cv *crdtVerse) NewUpdatableStore(name string) (*updatableStore, error){
	st, err := cv.newCRDT(name, &updatableValidator{})
	if err != nil{return nil, err}
	return &updatableStore{st}, nil
}
func (s *updatableStore) Put(key string, val []byte) error{
	tb, err := time.Now().UTC().MarshalJSON()
	if err != nil{return err}
	tKey := base64.StdEncoding.EncodeToString(tb)

	//ds.NewKey(key) => "/"+key
	dsKey := ds.NewKey(key).ChildString("/"+tKey)
	return s.dt.Put(s.ctx, dsKey, val)
}
func (s *updatableStore) Get(key string) ([]byte, error){
	rs, err := s.dt.Query(s.ctx, query.Query{
		Prefix: "/"+key,
		Orders: []query.Order{updatableOrder{}},
	})
	if err != nil{return nil, err}
	r := <-rs.Next()
	rs.Close()
	return r.Value, nil
}
func (s *updatableStore) GetSize(key string) (int, error){
	rs, err := s.dt.Query(s.ctx, query.Query{
		Prefix: "/"+key,
		Orders: []query.Order{updatableOrder{}},
		ReturnsSizes: true,
	})
	if err != nil{return -1, err}
	r := <-rs.Next()
	rs.Close()
	return r.Size, nil
}
func (s *updatableStore) Has(key string) (bool, error){
	rs, err := s.dt.Query(s.ctx, query.Query{
		Prefix: "/"+key,
		Orders: []query.Order{updatableOrder{}},
		KeysOnly: true,
	})
	if err != nil{return false, err}
	r := <-rs.Next()
	rs.Close()
	return r.Value != nil, nil
}
func (s *updatableStore) Query(qs ...query.Query) (query.Results, error){
	var q query.Query
	if len(qs) == 0{
		q = query.Query{}
	}else{
		q = qs[0]
	}
	q.Orders = append(q.Orders, updatableOrder{})
	return s.dt.Query(s.ctx, q)
}