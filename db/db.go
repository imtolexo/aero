package db

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/imtolexo/aero/conf"
	"math/rand"
	"net/url"
)

func init() {
	initMySql()
	initMongo()
}

var connMySqlWrite string
var connMySqlRead []string

func initMySql() {
	initMySqlMaster()
	initMySqlSlaves()
}

func initMySqlMaster() {
	lookup := "database.master"
	if conf.Exists(lookup) {
		connMySqlWrite = GetMySqlConnFromConfig(lookup)
	}
}

func initMySqlSlaves() {
	lookup := "database.slaves"
	if conf.Exists(lookup) {
		slaves := conf.StringSlice(lookup, []string{})
		connMySqlRead = make([]string, len(slaves))
		for i, container := range slaves {
			connMySqlRead[i] = GetMySqlConnFromConfig(container)
		}
	}
}

// Expected attributes for MySQL connection
// container:
// 	 username: user
//   password: pass
//   host:     localhost
//   port:     3306
//   db:       db-name
//   timezone:
func GetMySqlConnFromConfig(container string) string {
	if !conf.Exists(container) {
		panic("Container for mysql configuration not found")
	}

	username := conf.String(container+".username", "")
	password := conf.String(container+".password", "")
	host := conf.String(container+".host", "")
	port := conf.String(container+".port", "")
	db := conf.String(container+".db", "")
	timezone := conf.String(container+".timezone", "")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=%s",
		username, password,
		host, port, db,
		url.QueryEscape(timezone),
	)
}

func GetDefaultMySqlConn(write bool) string {
	if write {
		return connMySqlWrite
	}

	if connMySqlRead == nil || len(connMySqlRead) == 0 {
		return connMySqlWrite
	}

	return connMySqlRead[rand.Intn(len(connMySqlRead))]
}

var connMongo string
var dbMongo string

func initMongo() {
	lookup := "database.mongo"
	if conf.Exists(lookup) {
		connMongo = GetMongoConnFromConfig(lookup)
		dbMongo = conf.String("database.mongo.db", "")
	}
}

func GetDefaultMongoConn() (string, string) {
	return connMongo, dbMongo
}

func GetMongoConnFromConfig(container string) string {
	if !conf.Exists(container) {
		panic("Container for mongo configuration not found")
	}

	username := conf.String(container+".username", "")
	password := conf.String(container+".password", "")
	host := conf.String(container+".host", "")
	port := conf.String(container+".port", "")
	db := conf.String(container+".db", "")
	replicas := conf.String(container+".replicas", "")
	options := conf.String(container+".options", "")

	if port != "" {
		port = ":" + port
	}
	if replicas != "" {
		replicas = "," + replicas
	}
	if options != "" {
		options = "?" + options
	}
	auth := ""
	if username != "" || password != "" {
		auth = username + ":" + password + "@"
	}

	return fmt.Sprintf("mongodb://%s%s%s%s/%s%s",
		auth, host, port, replicas,
		db, options,
	)
}
