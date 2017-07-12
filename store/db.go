package main


type Model interface{
	Key() (error, string) // Get key of the object used in Get()
	Decode([]byte) error
	Encode() ([]byte, error)
	Get(item *Model) error
	Save(item *Model) error
	Delete(item *Model)
	Exists(item *Model) (bool, error)
	New() *Model
	Validate() bool
}

/*
	Every Database implementation should implement.
 */
type DB interface {
	Get(key string) ([]byte, error)
	Exists(key string) (bool, error)
	GetAllStartingWith(prefix string, start int, count int) ([][]byte, error)
	ListAllRecordsStartingWith(prefix string) ([]string, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Close() error
}

type ParentModel struct{
	Config *Settings
	DB *DB
}
