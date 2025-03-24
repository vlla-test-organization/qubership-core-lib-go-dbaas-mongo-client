package model

// MongoConnProperties is a way to store information about db connection locally
type MongoConnProperties struct {
	Url        string
	Username   string
	Password   string
	AuthDbName string
	DbName     string
}
