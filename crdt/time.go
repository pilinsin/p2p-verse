package crdtverse

import(
	"context"
	"errors"
	"time"
	"encoding/base64"
	"strings"
	"strconv"

	"golang.org/x/crypto/argon2"

	query "github.com/ipfs/go-datastore/query"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	pv "github.com/pilinsin/p2p-verse"
)

type tcFilter struct{
	tc *timeController
}
func (f tcFilter) Filter(e query.Entry) bool{
	ok, err := f.tc.Has(e.Key)
	return err == nil && ok
}


const proofKey = "time-proof"

//proof key: <pid>/<proofKey>/<data key hash>/<tKey>
type proofFilter struct{
	keyHash string
}
func (f proofFilter) Filter(e query.Entry) bool{
	keys := strings.Split(strings.TrimPrefix(e.Key, "/"), "/")
	if len(keys) < 4{return false}
	if keys[1] != proofKey{return false}

	if f.keyHash == ""{return true}
	return keys[2] == f.keyHash
}

type flagFilter struct{flag bool}
func (f flagFilter) Filter(e query.Entry) bool{
	ok, err := strconv.ParseBool(string(e.Value))
	return err == nil && ok == f.flag
}



type timeController struct{
	ctx context.Context
	cancel func()
	dStore *updatableSignatureStore
	pStore *updatableSignatureStore
	name string
	begin time.Time
	end time.Time
	eps time.Duration
	cooldown time.Duration
	n int
}
func (cv *crdtVerse) NewTimeController(name string, begin, end time.Time, eps, cooldown time.Duration, n int) (*timeController, error){
	exmpl := pv.RandString(32)
	hash := argon2.IDKey([]byte(name), []byte(exmpl), 1, 64*1024, 4, 32)
	name = base64.URLEncoding.EncodeToString(hash)

	st, err := cv.NewUpdatableSignatureStore(name)
	if err != nil{return nil, err}
	usst := st.(*updatableSignatureStore)

	ctx, cancel := context.WithCancel(context.Background())
	tc := &timeController{ctx, cancel, nil, usst, name, begin, end, eps, cooldown, n}
	if err := tc.pStore.InitPut(exmpl); err != nil{
		tc.Close()
		return nil, err
	}
	return tc, nil
}
func (cv *crdtVerse) LoadTimeController(tAddr string)(*timeController, error){
	m, err := base64.URLEncoding.DecodeString(tAddr)
	if err != nil{return nil, err}
	tp := &pb.TimeParams{}
	if err := tp.Unmarshal(m); err != nil{return nil, err}

	s, err := cv.NewUpdatableSignatureStore(tp.Name)
	if err != nil{return nil, err}
	usst := s.(*updatableSignatureStore)
	ctx, cancel := context.WithCancel(context.Background())
	tc := &timeController{ctx, cancel, nil, usst, tp.Name, tp.Begin, tp.End, tp.Eps, tp.Cool, tp.N}
	if err := tc.initLoad(); err != nil{return nil, err}
	return tc, nil
}
func (tc *timeController) initLoad() error{
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	ticker := time.NewTicker(time.Second*2)
	for{
		select{
		case <-ctx.Done():
			tc.Close()
			return errors.New("load error: sync timeout")
		case <-ticker.C:
			if err := tc.pStore.Sync(); err != nil{
				tc.Close()
				return err
			}

			if ok := tc.pStore.LoadCheck(); ok{return nil}
		}
	}
}
func (tc *timeController) Address() string{
	tp := &pb.TimeParams{tc.name, tc.begin, tc.end, tc.eps, tc.cooldown, tc.n}
	m, err := tp.Marshal()
	if err != nil{return ""}
	return base64.URLEncoding.EncodeToString(m)
}
func (tc *timeController) Close(){
	tc.cancel()
}

func (tc *timeController) getAllNewData() (<-chan string, error){
	rs, err := tc.dStore.baseQuery(query.Query{
		KeysOnly: true,
	})
	if err != nil{return nil, err}

	ch := make(chan string)
	go func(){
		defer close(ch)
		for r := range rs.Next(){
			ch <- r.Key
		}
	}()
	return ch, nil
}
func (tc *timeController) withinTime(t time.Time) bool{
	return t.After(tc.begin) && t.Before(tc.end)
}
func (tc *timeController) extractTime(key string) (time.Time, error){
	keys := strings.Split(strings.TrimPrefix(key, "/"), "/")
	tKey := keys[len(keys)-1]
	tb, err := base64.URLEncoding.DecodeString(tKey)
	if err != nil{return time.Time{}, err}

	t := time.Time{}
	if err := t.UnmarshalBinary(tb); err != nil{return time.Time{}, err}
	return t, nil
}
func (tc *timeController) makedataKeyHash(key string) string{
	keys := strings.Split(strings.TrimPrefix(key, "/"), "/")
	if len(keys) < 3{return ""}

	tKey := keys[len(keys)-1]
	cKey := strings.Join(keys[:len(keys)-1], "/")
	hash := argon2.IDKey([]byte(cKey), []byte(tKey), 1, 64*1024, 4, 64)
	return base64.URLEncoding.EncodeToString(hash)
}
func (tc *timeController) put(key string, flag bool) error{
	keyHash := tc.makedataKeyHash(key)
	if keyHash == ""{return errors.New("invalid key")}
	keyHash = proofKey + "/" + keyHash

	b := []byte(strconv.FormatBool(flag))
	return tc.pStore.Put(keyHash, b)
}
func (tc *timeController) getNewProofs(key string) (query.Results, error){
	keyHash := tc.makedataKeyHash(key)
	if keyHash == ""{return nil, errors.New("invalid key")}

	return tc.pStore.Query(query.Query{
		Filters: []query.Filter{proofFilter{keyHash}},
		Limit: tc.n,
	})
}

func (tc *timeController) grant(){
	keys, err := tc.getAllNewData()
	if err != nil{return}
	for key := range keys{
		t, err := tc.extractTime(key)
		if err != nil{
			tc.put(key, false)
			continue
		}
		if ok := tc.withinTime(t); !ok{
			tc.put(key, false)
			continue
		}

		rs, err := tc.getNewProofs(key)
		if err != nil{
			tc.put(key, false)
			continue
		}
		ts, err := query.NaiveFilter(rs, flagFilter{true}).Rest()
		if err != nil{
			tc.put(key, false)
			continue
		}
		nT := len(ts)
		fs, err := query.NaiveFilter(rs, flagFilter{false}).Rest()
		if err != nil{
			tc.put(key, false)
			continue
		}
		nF := len(fs)

		if nT + nF >= tc.n{
			tc.put(key, nT > nF)
		}else{
			now := time.Now()
			ok := now.After(t) && now.Before(t.Add(tc.eps))
			tc.put(key, ok)
		}
	}
}
func (tc *timeController) AutoGrant(){
	go func(){
		ticker := time.NewTicker(tc.cooldown)
		for{
			select{
			case <-ticker.C:
				tc.dStore.Sync()
				tc.pStore.Sync()
				tc.grant()
			case <-tc.ctx.Done():
				return
			}
		}
	}()
}
//key: <pid>/<category>/<tKey>
func (tc *timeController) Has(key string) (bool, error){
	t, err := tc.extractTime(key)
	if err != nil{return false, err}
	if ok := tc.withinTime(t); !ok{return false, nil}

	rs, err := tc.getNewProofs(key)
	if err != nil{return false, err}
	ts, err := query.NaiveFilter(rs, flagFilter{true}).Rest()
	if err != nil{return false, err}
	nT := len(ts)
	fs, err := query.NaiveFilter(rs, flagFilter{false}).Rest()
	if err != nil{return false, err}
	nF := len(fs)

	if nT + nF >= tc.n{
		return nT > nF, nil
	}else{
		now := time.Now()
		return now.After(t) && now.Before(t.Add(tc.eps)), nil
	}
}


