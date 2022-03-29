package crdtverse

import(
	"time"
	"strings"
	"encoding/base64"

	query "github.com/ipfs/go-datastore/query"
)

//data key: <category>/<tKey>
//a<b:-1, a==b:0, a>b:1
type categoryOrder struct{}
func (o categoryOrder) Compare(a, b query.Entry) int{
	//extract a key except tKey
	keys := strings.Split(strings.TrimPrefix(a.Key, "/"), "/")
	if len(keys) < 2{return 1}
	aKey := strings.Join(keys[:len(keys)-1], "/")
	
	keys = strings.Split(strings.TrimPrefix(b.Key, "/"), "/")
	if len(keys) < 2{return -1}
	bKey := strings.Join(keys[:len(keys)-1], "/")

	return strings.Compare(aKey, bKey)
}

//a<b:-1, a==b:0, a>b:1
type updatableOrder struct{}
func (o updatableOrder) Compare(a, b query.Entry) int{
	keys := strings.Split(strings.TrimPrefix(a.Key, "/"), "/")
	taKey := keys[len(keys)-1]
	ma, err := base64.URLEncoding.DecodeString(taKey)
	if err != nil{return 1}
	keys = strings.Split(strings.TrimPrefix(b.Key, "/"), "/")
	tbKey := keys[len(keys)-1]
	mb, err := base64.URLEncoding.DecodeString(tbKey)
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
	keys := strings.Split(strings.TrimPrefix(key, "/"), "/")
	tKey := keys[len(keys)-1]
	tb, err := base64.URLEncoding.DecodeString(tKey)
	if err != nil{return false}

	t := time.Time{}
	if err := t.UnmarshalJSON(tb); err != nil{return false}
	isUTC := t.Location().String() == time.UTC.String()
	isBefore := t.Before(time.Now().UTC())
	return isUTC && isBefore
}
func (v *updatableValidator) Select(key string, vals [][]byte) bool{
	return len(vals) == 1
}


type updatableStore struct{
	*logStore
}
func (cv *crdtVerse) NewUpdatableStore(name string, _ ...*StoreOpts) (iStore, error){
	st, err := cv.newCRDT(name, &updatableValidator{})
	if err != nil{return nil, err}
	return &updatableStore{st}, nil
}
func (cv *crdtVerse) LoadUpdatableStore(addr string, _ ...*StoreOpts) (iStore, error){
	addr = strings.Split(strings.TrimPrefix(addr, "/"), "/")[0]
	return cv.NewUpdatableStore(addr)
}
func (s *updatableStore) Put(key string, val []byte) error{	
	tb, err := time.Now().UTC().MarshalJSON()
	if err != nil{return err}
	tKey := base64.URLEncoding.EncodeToString(tb)

	key += "/"+tKey
	return s.logStore.Put(key, val)
}
func (s *updatableStore) Get(key string) ([]byte, error){
	rs, err := s.logStore.Query(query.Query{
		Prefix: "/"+key,
		Orders: []query.Order{updatableOrder{}},
		Limit: 1,
	})
	if err != nil{return nil, err}
	r := <-rs.Next()
	rs.Close()
	return r.Value, nil
}
func (s *updatableStore) GetSize(key string) (int, error){
	rs, err := s.logStore.Query(query.Query{
		Prefix: "/"+key,
		Orders: []query.Order{updatableOrder{}},
		ReturnsSizes: true,
		Limit: 1,
	})
	if err != nil{return -1, err}
	r := <-rs.Next()
	rs.Close()
	return r.Size, nil
}
func (s *updatableStore) Has(key string) (bool, error){
	rs, err := s.logStore.Query(query.Query{
		Prefix: "/"+key,
		Orders: []query.Order{updatableOrder{}},
		KeysOnly: true,
		Limit: 1,
	})
	if err != nil{return false, err}
	resList, err := rs.Rest()
	rs.Close()
	return len(resList) > 0, err
}
func (s *updatableStore) Query(qs ...query.Query) (query.Results, error){
	var q query.Query
	if len(qs) == 0{
		q = query.Query{}
	}else{
		q = qs[0]
	}
	q.Orders = append(q.Orders, categoryOrder{}, updatableOrder{})
	rs, err := s.logStore.Query(q)
	if err != nil{return nil, err}

	cKey := ""
	ch := make(chan query.Result)
	go func(){
		defer close(ch)
		for r := range rs.Next(){
			keys := strings.Split(strings.TrimPrefix(r.Key, "/"), "/")
			if len(keys) < 2{continue}
			cKey2 := strings.Join(keys[:len(keys)-1], "/")
			if cKey != cKey2{
				ch <- r
				cKey = cKey2
			}
		}
	}()
	return query.ResultsWithChan(query.Query{}, ch), nil
}