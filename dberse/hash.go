package dberse

import(
	"errors"
	"context"
	"encoding/json"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)


func makeHashKey(hash, salt []byte) (string, error){
	uhHash := argon2.IDKey(hash, salt, 1, 64*1024, 4, 128)
	key := base64.StdEncoding.EncodeToString(uhHash)
	return key, nil
}
type hashValidator struct{}
func (v hashValidator) Validate(key0 string, val []byte) error{
	ns, _, key, err := splitKey(key0)
	if err != nil{return err}
	if ns != v.Type(){return errors.New("invalid key")}

	hd := struct{
		Data []byte
		Hash []byte
		Salt []byte
	}{}
	if err := json.Unmarshal(val, &hd); err != nil{return err}

	s, _ := makeHashKey(hd.Hash, hd.Salt)
	if key != s{return errors.New("validation error")}

	return nil
}
func (v hashValidator) Select(key string, vals [][]byte) (int, error){
	if len(vals) > 1{
		return -1, nil
	}
	return 0, nil
}
func (v hashValidator) Type() string{
	return "/hash"
}


func (d *dBerse) NewHashStore(name string, hash, salt []byte) (iStore, error){
	db, err := d.newStore(name, Hash)
	if err != nil{return nil, err}
	return &hashStore{db, hash, salt}, nil
}

type hashedData struct{
	Data, Hash, Salt []byte
}
func (hd hashedData) Marshal() ([]byte, error){
	return json.Marshal(hd)
}
func UnmarshalHashedData(m []byte) (*hashedData, error){
	hd := &hashedData{}
	err := json.Unmarshal(m, hd)
	return hd, err
}
type hashStore struct{
	*store
	hash []byte
	salt []byte
}
func (d *hashStore) Put(val []byte, keys ...string) error{
	hd := &hashedData{val, d.hash, d.salt}
	mhd, err := hd.Marshal()
	if err != nil{return err}

	key, _ := makeHashKey(d.hash, d.salt)
	return d.store.Put(mhd, key)
}
func (d *hashStore) GetRaw(key string) ([]byte, error){
	return d.store.Get(key)
}
func (d *hashStore) Get(key string) ([]byte, error){
	mhd, err := d.store.Get(key)
	if err != nil{return nil, err}
	hd, err := UnmarshalHashedData(mhd)
	if err != nil{return nil, err}

	return hd.Data, nil
}
func (d *hashStore) GetRawWait(ctx context.Context, key string) ([]byte, error){
	return d.store.GetWait(ctx, key)
}
func (d *hashStore) GetWait(ctx context.Context, key string) ([]byte, error){
	mhd, err := d.store.GetWait(ctx, key)
	if err != nil{return nil, err}
	hd, err := UnmarshalHashedData(mhd)
	if err != nil{return nil, err}

	return hd.Data, nil
}

