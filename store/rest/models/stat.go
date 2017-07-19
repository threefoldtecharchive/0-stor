package models

import (
	"bytes"
	"encoding/binary"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/store/db"
)

type StoreStatMgr struct {
	db db.DB
	mu sync.RWMutex
}

func NewStoreStatMgr(db db.DB) *StoreStatMgr {
	return &StoreStatMgr{
		db: db,
		mu: sync.RWMutex{},
	}
}

func (s *StoreStatMgr) GetStats() (StoreStat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	storeStat := StoreStat{}
	b, err := s.db.Get(storeStat.Key())
	if err != nil {
		log.Errorln(err.Error())
		return storeStat, err
	}

	if err := storeStat.Decode(b); err != nil {
		log.Errorln(err.Error())
		return storeStat, err
	}
	return storeStat, nil
}

func (s *StoreStatMgr) SetStats(available, used uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	storeStat := StoreStat{}
	b, err := s.db.Get(storeStat.Key())
	if err != nil {
		log.Errorln(err.Error())
		return err
	}

	if err := storeStat.Decode(b); err != nil {
		log.Errorln(err.Error())
		return err
	}

	storeStat.SizeAvailable = available
	storeStat.SizeUsed = used

	b, err = storeStat.Encode()
	if err != nil {
		return err
	}

	return s.db.Set(storeStat.Key(), b)
}

type StoreStat struct {
	// Size available = free disk space - reserved space (in bytes)
	SizeAvailable uint64 `json:"size_available" validate:"min=1,nonzero"`
	// SizeUsed = sum of all reservation size (in bytes)
	SizeUsed uint64 `json:"size_used" validate:"min=1,nonzero"`
}

func (s *StoreStat) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, s.SizeAvailable)
	binary.Write(buf, binary.LittleEndian, s.SizeUsed)
	return buf.Bytes(), nil
}

func (s *StoreStat) Decode(data []byte) error {
	s.SizeAvailable = binary.LittleEndian.Uint64(data[:8])
	s.SizeUsed = binary.LittleEndian.Uint64(data[8:16])
	return nil
}

func (s StoreStat) Key() string {
	return STORE_STATS_PREFIX
}
