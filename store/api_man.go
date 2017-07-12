package main


type Models struct{
	Store *DB
}


type APIManager struct {
	Models Models
	DB *DB
	Config *Settings
}

func NewAPIManager(db DB, config *Settings) *APIManager {
	man := new(APIManager)
	man.Config = config
	man.DB = &db
	return man
}
