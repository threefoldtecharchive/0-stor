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
func (b *Badger) Init(metaDir, dataDir string){

	log.Println("Initializing db directories")

	if err := os.MkdirAll(metaDir, 0774); err != nil{
		log.Fatal(err.Error())
	}

	log.Printf("\t\tMeta dir: %v", metaDir)

	if err := os.MkdirAll(dataDir, 0774); err != nil{
		log.Fatal(err.Error())
	}

	log.Printf("\t\tData dir: %v", dataDir)
}

/* Constructor */
func (b *Badger) New(metaDir, dataDir string) *Badger{
	opts := badger.DefaultOptions
	opts.Dir = metaDir
	opts.ValueDir = dataDir

	kv, err:= badger.NewKV(&opts)

	if err != nil{
		log.Fatal(err.Error())
	}

	log.Println("Successfully loaded db")

	return &Badger{
		store:kv,
	}
}

/* Close connection */
func (b *Badger) Close(){
	if err := b.store.Close(); err != nil{
		log.Fatal(err.Error())
	}
}

/* Get */
func (b *Badger) Get(key string) []byte{
	var item badger.KVItem
	if err := b.store.Get([]byte(key), &item); err != nil{
		log.Fatal(err.Error())
	}
	return item.Value()
}

/* Set */
func (b *Badger) Set(key string, val []byte){
	if err := b.store.Set([]byte(key), val); err != nil{
		log.Fatal(err.Error())
	}
}

/* Delete */
func (b *Badger) Delete(key string){
	if err := b.store.Delete([]byte(key)); err != nil{
		log.Fatal(err.Error())
	}
}

