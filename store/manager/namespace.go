package manager

import (
	"fmt"

	"github.com/zero-os/0-stor/store/db"
)

type NamespaceManager struct {
	db db.DB
}

func NewNamespaceManager(db db.DB) *NamespaceManager {
	return &NamespaceManager{
		db: db,
	}
}

func nsKey(label string) []byte {
	return []byte(fmt.Sprintf("%s:%s", NAMESPACE_PREFIX, label))
}

func (mgr *NamespaceManager) Get(label string) (*db.Namespace, error) {
	b, err := mgr.db.Get(nsKey(label))
	if err != nil {
		return nil, err
	}

	ns := db.NewNamespace()
	err = ns.Decode(b)
	return ns, err
}
