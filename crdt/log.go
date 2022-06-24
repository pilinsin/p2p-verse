package crdtverse

import (
	"time"
)

func getLogOpts(opts ...*StoreOpts) time.Time {
	if len(opts) == 0 {
		return time.Time{}
	}
	return opts[0].TimeLimit
}

type logStore struct {
	*baseStore
}

func (cv *crdtVerse) NewLogStore(name string, opts ...*StoreOpts) (IStore, error) {
	st := &baseStore{}
	if err := cv.initCRDT(name, newBaseValidator(st), st); err != nil {
		return nil, err
	}

	tl := getLogOpts(opts...)
	st.timeLimit = tl
	st.setTimeLimit()
	return &logStore{st}, nil
}
