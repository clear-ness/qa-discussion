package testlib

import (
	"github.com/clear-ness/qa-discussion/store"
)

type TestStore struct {
	store.Store
}

func (s TestStore) Close() {
}
