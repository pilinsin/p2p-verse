package crdtverse

import (
	"bytes"
	query "github.com/ipfs/go-datastore/query"
	"strings"
)

//data key: (<pid>)/<category>/(<tKey>)
type KeyMatchFilter struct {
	Key string
}

func (f KeyMatchFilter) Filter(e query.Entry) bool {
	keys := strings.Split(strings.TrimPrefix(e.Key, "/"), "/")
	fKeys := strings.Split(strings.TrimPrefix(f.Key, "/"), "/")
	if len(keys) < len(fKeys) {
		return false
	}

	for idx := range fKeys {
		if fKeys[idx] != "*" && fKeys[idx] != keys[idx] {
			return false
		}
	}
	return true
}

type KeyExistFilter struct {
	Key string
}

func (f KeyExistFilter) Filter(e query.Entry) bool {
	keys := strings.Split(strings.TrimPrefix(e.Key, "/"), "/")
	for _, eKey := range keys {
		if eKey == f.Key {
			return true
		}
	}
	return false
}

type ValueMatchFilter struct {
	Val []byte
}

func (f ValueMatchFilter) Filter(e query.Entry) bool {
	return bytes.Equal(e.Value, f.Val)
}
