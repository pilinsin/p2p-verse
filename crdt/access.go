package crdtverse

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	query "github.com/ipfs/go-datastore/query"
	pb "github.com/pilinsin/p2p-verse/crdt/pb"
	"golang.org/x/crypto/argon2"
	proto "google.golang.org/protobuf/proto"

	pv "github.com/pilinsin/p2p-verse"
)

type acFilter struct {
	ac *accessController
}

func (f acFilter) Filter(e query.Entry) bool {
	ok, err := f.ac.Has(e.Key)
	return err == nil && ok
}

func getAccessOpts(opts ...*StoreOpts) []byte {
	if len(opts) == 0 {
		salt := make([]byte, 8)
		rand.Read(salt)
		return salt
	}
	return opts[0].Salt
}

type accessController struct {
	store *signatureStore
	name  string
	salt  []byte
}

func (cv *crdtVerse) NewAccessController(name string, accesses <-chan string, opts ...*StoreOpts) (*accessController, error) {
	salt := getAccessOpts(opts...)

	st, err := cv.NewSignatureStore(name, opts...)
	if err != nil {
		return nil, err
	}
	sgst := st.(*signatureStore)
	ac := &accessController{sgst, name, salt}
	if err := ac.init(accesses); err != nil {
		ac.Close()
		return nil, err
	}
	return ac, nil
}
func (s *accessController) init(accesses <-chan string) error {
	if s.store.priv == nil || s.store.pub == nil {
		s.store.priv, s.store.pub, _ = generateKeyPair()
	}

	for access := range accesses {
		b := make([]byte, 32)
		rand.Read(b)
		if err := s.put(access, b); err != nil {
			s.Close()
			return err
		}
	}

	exmpl := pv.RandString(32)
	if err := s.store.InitPut(exmpl); err != nil {
		s.Close()
		return err
	}

	s.store.priv = nil
	s.store.autoSync()
	return nil
}
func (s *accessController) put(key string, val []byte) error {
	hash := argon2.IDKey([]byte(key), s.salt, 1, 64*1024, 4, 64)
	hashKey := base64.URLEncoding.EncodeToString(hash)
	return s.store.Put(hashKey, val)
}
func (cv *crdtVerse) loadAccessController(ctx context.Context, acAddr string) (*accessController, error) {
	m, err := base64.URLEncoding.DecodeString(acAddr)
	if err != nil {
		return nil, err
	}
	ap := &pb.AccessParams{}
	if err := proto.Unmarshal(m, ap); err != nil {
		return nil, err
	}
	pub, err := StrToPubKey(ap.GetPid())
	if err != nil {
		return nil, err
	}

	ac, err := cv.baseLoadAccess(ctx, ap.GetName(), ap.GetSalt(), &StoreOpts{Priv: nil, Pub: pub})
	if err != nil{return nil, err}
	ac.store.autoSync()
	return ac, nil
}
func (cv *crdtVerse) baseLoadAccess(ctx context.Context, addr string, salt []byte, opts ...*StoreOpts) (*accessController, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			st, err := cv.NewSignatureStore(addr, opts...)
			if err != nil {
				if strings.HasPrefix(err.Error(), dirLock) {
					fmt.Println(err, ", now reloading...")
					continue
				}
				return nil, err
			}

			sgst := st.(*signatureStore)
			acst := &accessController{sgst, addr, salt}

			err = acst.loadCheck()
			if err == nil {
				return acst, nil
			}
			if strings.HasPrefix(err.Error(), timeout) {
				fmt.Println(err, ", now reloading...")
				continue
			}
			return nil, err
		}
	}
}
func (s *accessController) loadCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			s.Cancel()
			return errors.New("load error: sync timeout (access)")
		case <-ticker.C:
			if err := s.store.Sync(); err != nil {
				s.Close()
				return err
			}

			if ok := s.store.LoadCheck(); ok {
				return nil
			}
		}
	}
}

func (s *accessController) Close() {
	s.store.Close()
}
func (s *accessController) Cancel() {
	s.store.Cancel()
}
func (s *accessController) Address() string {
	pid := PubKeyToStr(s.store.pub)
	m, err := proto.Marshal(&pb.AccessParams{
		Pid:  pid,
		Name: s.name,
		Salt: s.salt,
	})
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(m)
}
func (s *accessController) Has(key string) (bool, error) {
	sKey := strings.Split(strings.TrimPrefix(key, "/"), "/")[0]
	return s.has(sKey)
}
func (s *accessController) has(key string) (bool, error) {
	hash := argon2.IDKey([]byte(key), s.salt, 1, 64*1024, 4, 64)
	hashKey := base64.URLEncoding.EncodeToString(hash)

	pid := PubKeyToStr(s.store.pub)
	return s.store.Has(pid + "/" + hashKey)
}
