package crdtverse

import(
	"fmt"
	"strings"
	"encoding/base64"
	"crypto/rand"
	"time"

	pv "github.com/pilinsin/p2p-verse"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	query "github.com/ipfs/go-datastore/query"
	"golang.org/x/crypto/argon2"
)

type acFilter struct{
	ac *accessController
}
func (f acFilter) Filter(e query.Entry) bool{
	ok, err := f.ac.Has(e.Key)
	return err == nil && ok
}



func getAccessOpts(opts ...*StoreOpts) []byte{
	if len(opts) == 0{
		salt := make([]byte, 8)
		rand.Read(salt)
		return salt
	}
	return opts[0].Salt
}

type accessController struct{
	store *signatureStore
	name string
	salt []byte
}
func (cv *crdtVerse) NewAccessController(name string, accesses <-chan string, opts ...*StoreOpts) (*accessController, error){
	salt := getAccessOpts(opts...)

	st, err := cv.NewSignatureStore(name, opts...)
	if err != nil{return nil, err}
	sgst := st.(*signatureStore)
	ac := &accessController{sgst, name, salt}
	if err := ac.init(accesses); err != nil{
		ac.Close()
		return nil, err
	}
	return ac, nil
}
func (s *accessController) init(accesses <-chan string) error{
	if s.store.priv == nil || s.store.pub == nil{
		s.store.priv, s.store.pub, _ = p2pcrypto.GenerateEd25519Key(rand.Reader)
	}

	if accesses == nil{
		b := make([]byte, 32)
		rand.Read(b)
		if err := s.put("*", b); err != nil{return err}
	}else{
		for access := range accesses{

			b := make([]byte, 32)
			rand.Read(b)
			if err := s.put(access, b); err != nil{return err}
		}
	}
	fmt.Println("wait for accessController broadcasting (30s)")
	<-time.Tick(time.Second*30)

	s.store.priv = nil
	return nil
}
func (s *accessController) put(key string, val []byte) error{
	sKey := strings.Split(strings.TrimPrefix(key, "/"), "/")[0]
	hash := argon2.IDKey([]byte(sKey), s.salt, 1, 64*1024, 4, 64)
	hashKey := base64.URLEncoding.EncodeToString(hash)
	return s.store.Put(hashKey, val)
}
func (cv *crdtVerse) LoadAccessController(acAddr string) (*accessController, error){
	m, err := base64.URLEncoding.DecodeString(acAddr)
	if err != nil{return nil, err}
	tmp := struct{
		P string
		N string
		S []byte
	}{}
	if err := pv.Unmarshal(m, &tmp); err != nil{return nil, err}

	pub, err := StrToPubKey(tmp.P)
	if err != nil{
		pub = nil
	}
	st, err := cv.NewSignatureStore(tmp.N, &StoreOpts{Priv: nil, Pub: pub})
	if err != nil{return nil, err}
	sgst := st.(*signatureStore)
	acst := &accessController{sgst, tmp.N, tmp.S}
	if err := acst.Sync(); err != nil{
		acst.Close()
		return nil, err
	}
	return acst, nil
}
func (s *accessController) Close(){
	s.store.Close()
}
func (s *accessController) Address() string{
	pid := PubKeyToStr(s.store.pub)
	m, err := pv.Marshal(struct{
		P, N string
		S []byte
	}{pid, s.name, s.salt})
	if err != nil{return ""}
	return base64.URLEncoding.EncodeToString(m)
}
func (s *accessController) Sync() error{
	return s.store.Sync()
}
func (s *accessController) Repair() error{
	return s.store.Repair()
}
func (s *accessController) Has(key string) (bool, error){
	if ok, err := s.has("*"); ok && err == nil{return true, nil}
	sKey := strings.Split(strings.TrimPrefix(key, "/"), "/")[0]
	return s.has(sKey)
}
func (s *accessController) has(key string) (bool, error){
	hash := argon2.IDKey([]byte(key), s.salt, 1, 64*1024, 4, 64)
	hashKey := base64.URLEncoding.EncodeToString(hash)

	pid := PubKeyToStr(s.store.pub)
	return s.store.Has(pid + "/" + hashKey)
}

