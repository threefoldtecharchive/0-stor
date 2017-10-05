package manager

import (
	"fmt"

	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/errors"
)

type ObjectManager struct {
	db        db.DB
	namespace string
}

const (
	_ = iota
	RefListOpSet
	RefListOpAppend
	RefListOpRemove
)

func NewObjectManager(namespace string, db db.DB) *ObjectManager {
	return &ObjectManager{
		namespace: namespace,
		db:        db,
	}
}

// Set saved an object into the key-value store as a blob of bytes
func (mgr *ObjectManager) Set(key []byte, data []byte, referenceList []string) error {
	obj := db.NewObject(mgr.namespace, key, mgr.db)
	obj.SetData(data)

	if err := obj.SetReferenceList(referenceList); err != nil {
		return err
	}

	return obj.Save()
}

func (mgr *ObjectManager) List(start, count int) ([][]byte, error) {
	prefix := fmt.Sprintf("%s:%s:", mgr.namespace, db.PrefixData)
	keys, err := mgr.db.List([]byte(prefix))
	if err != nil {
		return nil, err
	}

	// remove namespace prefix
	lenPrefix := len(prefix)
	for i := range keys {
		keys[i] = keys[i][lenPrefix:]
	}
	return keys, nil
}

func (mgr *ObjectManager) Get(key []byte) (*db.Object, error) {
	return db.NewObject(mgr.namespace, key, mgr.db), nil
}

func (mgr *ObjectManager) Delete(key []byte) error {
	obj := db.NewObject(mgr.namespace, key, mgr.db)
	return obj.Delete()
}

func (mgr *ObjectManager) Exists(key []byte) (bool, error) {
	obj := db.NewObject(mgr.namespace, key, mgr.db)
	return obj.Exists()
}

func (mgr *ObjectManager) UpdateReferenceList(key []byte, refList []string, op int) error {
	obj := db.NewObject(mgr.namespace, key, mgr.db)
	var err error

	switch op {
	case RefListOpSet:
		err = obj.SetReferenceList(refList)
	case RefListOpRemove:
		err = obj.RemoveReferenceList(refList)
	case RefListOpAppend:
		err = obj.AppendReferenceList(refList)
	default:
		err = fmt.Errorf("invalid reference list operation")
	}
	if err != nil {
		return err
	}

	return obj.Save()
}

type CheckStatus string

var (
	CheckStatusOK        CheckStatus = "ok"
	CheckStatusCorrupted CheckStatus = "corrupted"
	CheckStatusMissing   CheckStatus = "missing"
)

func (mgr *ObjectManager) Check(key []byte) (CheckStatus, error) {
	obj := db.NewObject(mgr.namespace, key, mgr.db)

	valid, err := obj.Validcrc()
	if err != nil {
		if err == errors.ErrNotFound {
			return CheckStatusMissing, nil
		}
		return CheckStatus(""), err
	}
	if !valid {
		return CheckStatusCorrupted, nil
	}

	return CheckStatusOK, nil
}
