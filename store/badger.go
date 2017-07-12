package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/dgraph-io/badger"
)

type Badger struct {
	KV *badger.KV
	Config *Settings
}

/* Constructor */
func NewBadger(settings *Settings) (*Badger, error) {
	log.Println("Initializing db directories")

	if err := os.MkdirAll(settings.DB.Dirs.Meta, 0774); err != nil {
		log.Printf("\t\tMeta dir: %v [ERROR]", settings.DB.Dirs.Meta)
		return nil, err
	}

	log.Printf("\t\tMeta dir: %v [SUCCESS]", settings.DB.Dirs.Meta)

	if err := os.MkdirAll(settings.DB.Dirs.Data, 0774); err != nil {
		log.Printf("\t\tData dir: %v [ERROR]", settings.DB.Dirs.Data)
		return nil, err
	}

	log.Printf("\t\tData dir: %v [SUCCESS]", settings.DB.Dirs.Data)

	opts := badger.DefaultOptions
	opts.Dir = settings.DB.Dirs.Meta
	opts.ValueDir = settings.DB.Dirs.Data

	kv, err := badger.NewKV(&opts)

	if err == nil {
		log.Println("Loading db [SUCCESS]")
	} else {
		log.Println("Loading db [ERROR]")
	}

	return &Badger{
		KV: kv,
		Config: settings,
	}, err
}

func (b Badger) New(settings *Settings) (*Badger, error) {
	return NewBadger(settings)
}

func (b Badger) Close() error {
	err := b.KV.Close()
	if err != nil{
		log.Errorln(err.Error())
		err = ErrorDB
	}
	return err
}

func (b Badger) Delete(key string) error {
	err :=  b.KV.Delete([]byte(key))
	if err != nil{
		log.Errorln(err.Error())
		err = ErrorDB
	}
	return err
}

func (b Badger) Set(key string, val []byte) error {
	err := b.KV.Set([]byte(key), val)
	if err != nil{
		log.Errorln(err.Error())
		err = ErrorDB
	}
	return err
}

func (b Badger) Get(key string) ([]byte, error) {
	var item badger.KVItem

	err := b.KV.Get([]byte(key), &item)

	if err != nil{
		log.Errorln(err.Error())
		err = ErrorDB
		return nil, err
	}

	v := item.Value()

	if len(v) == 0{
		err = ErrorNotFound

	}

	return v, err
}

func (b Badger) Exists(key string) (bool, error) {
	exists, err :=  b.KV.Exists([]byte(key))
	if err != nil{
		log.Errorln(err.Error())
		err = ErrorDB
	}
	return exists, err
}

func (b Badger) GetAllStartingWith(prefix string, start int, count int) ([][]byte, error) {
	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = b.Config.DB.Iterator.PreFetchSize

	it := b.KV.NewIterator(opt)
	defer it.Close()

	result := [][]byte{}

	counter := 0 // Number of namespaces encountered

	prefixBytes := []byte(prefix)

	for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
		item := it.Item()

		// Found a namespace
		counter++

		// Skip this namespace if its index < intended startingIndex
		if counter < start {
			continue
		}

		value := item.Value()
		result = append(result, value)



		if len(result) == count {
			break
		}
	}

	return result, nil
}

func (b Badger) ListAllRecordsStartingWith(prefix string) ([]string, error) {
	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = b.Config.DB.Iterator.PreFetchSize
	opt.FetchValues = false

	it := b.KV.NewIterator(opt)
	defer it.Close()

	result := []string{}

	prefixBytes := []byte(prefix)

	for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
		item := it.Item()
		key := string(item.Key()[:])
		result = append(result, key)
	}

	return result, nil
}

/* Get File */
func (b Badger) GetFile(key string) (*File, error) {
	bytes, err := b.Get(key)

	if err != nil {
		return nil, err
	}

	if bytes == nil {
		return nil, nil
	}

	file := &File{}
	err = file.FromBytes(bytes)
	if err != nil {
		return nil, err
	}
	return file, nil
}
