package dberse

import(
	"errors"
)

type constValidator struct{}
func (v constValidator) Validate(key0 string, val []byte) error{
	ns, _, _, err := splitKey(key0)
	if err != nil{return err}
	if ns != v.Type(){return errors.New("invalid key")}
	return nil
}
func (v constValidator) Select(key string, vals [][]byte) (int, error){
	if len(vals) > 1{
		return -1, nil
	}
	return 0, nil
}
func (v constValidator) Type() string{
	return "/const"
}


func (d *dBerse) NewConstStore(name string) (iStore, error){
	return d.newStore(name, Const)
}