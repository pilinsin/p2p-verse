package crdtverse

import(
	"strings"
	query "github.com/ipfs/go-datastore/query"
)

//data key: (<pid>)/<category>/(<tKey>)
type KeyMatchFilter struct{
	key string
}
func (f KeyMatchFilter) Filter(e query.Entry) bool{
	keys := strings.Split(strings.TrimPrefix(e.Key, "/"), "/")
	fKeys := strings.Split(strings.TrimPrefix(f.key, "/"), "/")
	if len(keys) < len(fKeys){return false}
	
	for idx := range fKeys{
		if fKeys[idx] != "*" && fKeys[idx] != keys[idx]{
			return false
		}
	}
	return true
}

type KeyExistFilter struct{
	key string
}
func (f KeyExistFilter) Filter(e query.Entry) bool{
	keys := strings.Split(strings.TrimPrefix(e.Key, "/"), "/")
	for _, eKey := range keys{
		if eKey == f.key{return true}
	}
	return false
}
