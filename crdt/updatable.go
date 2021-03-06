package crdtverse

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	query "github.com/ipfs/go-datastore/query"
)

//data key: <category>/<tKey>
//a<b:-1, a==b:0, a>b:1
type categoryOrder struct{}

func (o categoryOrder) Compare(a, b query.Entry) int {
	//extract a key except tKey
	keys := strings.Split(strings.TrimPrefix(a.Key, "/"), "/")
	if len(keys) < 2 {
		return 1
	}
	aKey := strings.Join(keys[:len(keys)-1], "/")

	keys = strings.Split(strings.TrimPrefix(b.Key, "/"), "/")
	if len(keys) < 2 {
		return -1
	}
	bKey := strings.Join(keys[:len(keys)-1], "/")

	return strings.Compare(aKey, bKey)
}

//a<b:-1, a==b:0, a>b:1
type updatableOrder struct{}

func (o updatableOrder) Compare(a, b query.Entry) int {
	keys := strings.Split(strings.TrimPrefix(a.Key, "/"), "/")
	taKey := keys[len(keys)-1]
	ma, err := base64.URLEncoding.DecodeString(taKey)
	if err != nil {
		return 1
	}
	keys = strings.Split(strings.TrimPrefix(b.Key, "/"), "/")
	tbKey := keys[len(keys)-1]
	mb, err := base64.URLEncoding.DecodeString(tbKey)
	if err != nil {
		return -1
	}

	ta := time.Time{}
	if err := ta.UnmarshalBinary(ma); err != nil {
		return 1
	}
	tb := time.Time{}
	if err := tb.UnmarshalBinary(mb); err != nil {
		return -1
	}

	if ok := ta.UTC().Equal(tb.UTC()); ok {
		return 0
	}
	if ok := ta.UTC().After(tb.UTC()); ok {
		return -1
	}
	return 1
}

type updatableValidator struct {
	iValidator
}

func newUpdatableValidator(s IStore) iValidator {
	return &updatableValidator{newBaseValidator(s)}
}
func (v *updatableValidator) Validate(key string, val []byte) bool {
	if ok := v.iValidator.Validate(key, val); !ok {
		return false
	}

	keys := strings.Split(strings.TrimPrefix(key, "/"), "/")
	tKey := keys[len(keys)-1]
	tb, err := base64.URLEncoding.DecodeString(tKey)
	if err != nil {
		return false
	}

	t := time.Time{}
	if err := t.UnmarshalBinary(tb); err != nil {
		return false
	}
	isUTC := t.Location().String() == time.UTC.String()
	isBefore := t.Before(time.Now().UTC())
	return isUTC && isBefore
}

type IUpdatableStore interface {
	IStore
	QueryAll(...query.Query) (query.Results, error)
}

type updatableStore struct {
	*baseStore
}

func (cv *crdtVerse) NewUpdatableStore(name string, opts ...*StoreOpts) (IUpdatableStore, error) {
	st := &baseStore{}
	if err := cv.initCRDT(name, newUpdatableValidator(st), st); err != nil {
		return nil, err
	}

	tl := getLogOpts(opts...)
	st.timeLimit = tl
	st.setTimeLimit()
	return &updatableStore{st}, nil
}

func (s *updatableStore) Put(key string, val []byte) error {
	tb, err := time.Now().UTC().MarshalBinary()
	if err != nil {
		return err
	}
	tKey := base64.URLEncoding.EncodeToString(tb)

	key += "/" + tKey
	return s.baseStore.Put(key, val)
}
func (s *updatableStore) Get(key string) ([]byte, error) {
	rs, err := s.baseStore.Query(query.Query{
		Prefix: "/" + key,
		Orders: []query.Order{updatableOrder{}},
		Limit:  1,
	})
	if err != nil {
		return nil, err
	}
	r := <-rs.Next()
	rs.Close()
	if r.Value == nil {
		return nil, errors.New("no valid data")
	}
	return r.Value, nil
}
func (s *updatableStore) GetSize(key string) (int, error) {
	rs, err := s.baseStore.Query(query.Query{
		Prefix:       "/" + key,
		Orders:       []query.Order{updatableOrder{}},
		ReturnsSizes: true,
		Limit:        1,
	})
	if err != nil {
		return -1, err
	}
	r := <-rs.Next()
	rs.Close()
	if r.Value == nil {
		return -1, errors.New("no valid data")
	}
	return r.Size, nil
}
func (s *updatableStore) Has(key string) (bool, error) {
	rs, err := s.baseStore.Query(query.Query{
		Prefix:   "/" + key,
		Orders:   []query.Order{updatableOrder{}},
		KeysOnly: true,
		Limit:    1,
	})
	if err != nil {
		return false, err
	}
	resList, err := rs.Rest()
	rs.Close()
	return len(resList) > 0, err
}
func (s *updatableStore) baseQuery(qs ...query.Query) (query.Results, error) {
	var q query.Query
	if len(qs) == 0 {
		q = query.Query{}
	} else {
		q = qs[0]
	}
	// example: [Am, ..., A1, Bn, ..., B1, ...]
	q.Orders = append(q.Orders, categoryOrder{}, updatableOrder{})
	return s.baseStore.Query(q)
}
func (s *updatableStore) Query(qs ...query.Query) (query.Results, error) {
	rs, err := s.baseQuery(qs...)
	if err != nil {
		return nil, err
	}

	cKey := ""
	ch := make(chan query.Result)
	go func() {
		defer close(ch)
		for r := range rs.Next() {
			keys := strings.Split(strings.TrimPrefix(r.Key, "/"), "/")
			if len(keys) < 2 {
				continue
			}
			cKey2 := strings.Join(keys[:len(keys)-1], "/")
			if cKey != cKey2 {
				ch <- r
				cKey = cKey2
			}
		}
	}()
	return query.ResultsWithChan(query.Query{}, ch), nil
}
func (s *updatableStore) QueryAll(qs ...query.Query) (query.Results, error) {
	rs, err := s.baseQuery(qs...)
	if err != nil {
		return nil, err
	}

	ch := make(chan query.Result)
	go func() {
		defer close(ch)
		for r := range rs.Next() {
			keys := strings.Split(strings.TrimPrefix(r.Key, "/"), "/")
			if len(keys) < 2 {
				continue
			}
			ch <- r
		}
	}()
	return query.ResultsWithChan(query.Query{}, ch), nil
}

func (s *updatableStore) initPut() error {
	return s.Put(s.name, []byte(s.name))
}
func (s *updatableStore) loadCheck() bool {
	if !s.inTime {
		return true
	}

	rs, err := s.Query(query.Query{
		KeysOnly: true,
		Limit:    1,
	})
	if err != nil {
		return false
	}

	resList, err := rs.Rest()
	return len(resList) > 0 && err == nil
}
