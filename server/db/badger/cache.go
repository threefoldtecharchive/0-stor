/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package badger

import (
	"hash/crc32"
	"sync"

	"github.com/zero-os/0-stor/server/db"

	log "github.com/Sirupsen/logrus"
	badgerdb "github.com/dgraph-io/badger"
	"github.com/hashicorp/golang-lru/simplelru"
)

const (
	sequenceCacheBucketCount = 16
	sequenceCacheBucketSize  = 64
	sequenceBandwidth        = 512
)

func newSequenceCache(db *badgerdb.DB) *sequenceCache {
	sq := new(sequenceCache)
	for i := range sq.buckets {
		sq.buckets[i] = newSequenceCacheBucket(db)
	}
	return sq
}

type sequenceCache struct {
	buckets [sequenceCacheBucketCount]*sequenceCacheBucket
}

func (cache *sequenceCache) IncrementKey(scopeKey []byte) ([]byte, error) {
	// TODO: research if there is a faster algorithm that we might
	// want to use instead of the std ChecksumIEEE,
	// for now this one is used, inspired by golang's semi-standard memcache.
	index := crc32.ChecksumIEEE(scopeKey) % sequenceCacheBucketCount
	seqIndex, err := cache.buckets[index].IncrementKey(scopeKey)
	if err != nil {
		return nil, err
	}
	return db.ScopedSequenceKey(scopeKey, seqIndex), nil
}

func (cache *sequenceCache) Purge() {
	for _, bucket := range cache.buckets {
		bucket.Purge()
	}
}

func newSequenceCacheBucket(db *badgerdb.DB) *sequenceCacheBucket {
	lru, _ := simplelru.NewLRU(sequenceCacheBucketSize, onEvict)
	return &sequenceCacheBucket{lru: lru, db: db}
}

type sequenceCacheBucket struct {
	lru *simplelru.LRU
	db  *badgerdb.DB
	mux sync.Mutex
}

func (bucket *sequenceCacheBucket) IncrementKey(scopeKey []byte) (uint64, error) {
	bucket.mux.Lock()
	defer bucket.mux.Unlock()
	scopeKeyStr := string(scopeKey)
	seq, ok := bucket.lru.Get(scopeKeyStr)
	if !ok {
		var err error
		seqKey := db.UnlistedKey(scopeKey)
		for {
			seq, err = bucket.db.GetSequence(seqKey, sequenceBandwidth)
			if err == nil {
				break
			}
			if err == badgerdb.ErrConflict {
				continue
			}
			return 0, err
		}
		bucket.lru.Add(scopeKeyStr, seq)
	}

	var (
		err error
		x   uint64
	)
	for {
		x, err = seq.(*badgerdb.Sequence).Next()
		if err == nil {
			return x, nil
		}
		if err == badgerdb.ErrConflict {
			continue
		}
		return 0, err
	}
}

func (bucket *sequenceCacheBucket) Purge() {
	bucket.mux.Lock()
	defer bucket.mux.Unlock()
	bucket.lru.Purge()
}

func onEvict(k, v interface{}) {
	seq := v.(*badgerdb.Sequence)
	err := seq.Release()
	if err != nil {
		log.Errorf("error while releasing badger sequence '%s': %v",
			k.([]byte), err)
	}
}
