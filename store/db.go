package main

import (
	"github.com/zaibon/badger/badger"
	"os"
	"log"
)


type Badger struct {
	store *badger.KV
}

/* Initialize */
func (b *Badger) Init(metaDir, dataDir string) error{

	log.Println("Initializing db directories")

	if err := os.MkdirAll(metaDir, 0774); err != nil{
		log.Printf("\t\tMeta dir: %v [ERROR]", metaDir)
		return err
	}

	log.Printf("\t\tMeta dir: %v [SUCCESS]", metaDir)

	if err := os.MkdirAll(dataDir, 0774); err != nil{
		log.Printf("\t\tData dir: %v [ERROR]", dataDir)
		return err
	}

	log.Printf("\t\tData dir: %v [SUCCESS]", dataDir)

	return nil
}

/* Constructor */
func (b *Badger) New(metaDir, dataDir string) (*Badger, error){
	opts := badger.DefaultOptions
	opts.Dir = metaDir
	opts.ValueDir = dataDir

	kv, err:= badger.NewKV(&opts)

	if err == nil{
		log.Println("Loading db [SUCCESS]")
	}else{
		log.Println("Loading db [ERROR]")
	}

	return &Badger{
		store:kv,
	}, err
}

/* Close connection */
func (b *Badger) Close() error{
	return b.store.Close()
}

/* Get */
func (b *Badger) Get(key string) ([]byte, error){
	var item badger.KVItem
	err := b.store.Get([]byte(key), &item)
	return item.Value(), err
}

/* Get File */
func (b *Badger) GetFile(key string) (*File, error){
	bytes, err := b.Get(key)

	if err != nil{
		return nil, err
	}

	if bytes == nil{
		return nil, nil
	}

	file := &File{}
	err = file.FromBytes(bytes)
	if err != nil{
		return nil, err
	}
	return file, nil
}


/* Set */
func (b *Badger) Set(key string, val []byte) error{
	return b.store.Set([]byte(key), val)
}

/* Delete */
func (b *Badger) Delete(key string) error{
	return b.store.Delete([]byte(key))
}

/* Exists */
func (b *Badger) Exists(key string) bool{
	config := loadSettings()

	opt := badger.DefaultIteratorOptions
	opt.FetchValues = false
	opt.PrefetchSize = config.Iterator.PreFetchSize


	it := b.store.NewIterator(opt)
	defer it.Close()

	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		k := string(item.Key()[:])

		if key == k {
			return true
		}
	}
	return false
}
