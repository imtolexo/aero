package tpostgres

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/imtolexo/aero/conf"
	"math/rand"
)

var connPostgresWrite string
var connPostgresRead []string
var validSSLMode = map[string]int{"disable": 1, "allow": 1, "prefer": 1, "require": 1, "verify-ca": 1, "verify-full": 1}

func initMaster(sslMode string) (err error) {
	lookup := "database.master"
	if conf.Exists(lookup) {
		if connPostgresWrite, err = getPostgresConnString(lookup, sslMode); err != nil {
			return
		}
	}
	return
}
func initSlaves(sslMode string) (err error) {
	lookup := "database.slaves"
	if conf.Exists(lookup) {
		slaves := conf.StringSlice(lookup, []string{})
		connPostgresRead = make([]string, len(slaves))
		for i, container := range slaves {
			if connPostgresRead[i], err = getPostgresConnString(container, sslMode); err != nil {
				return
			}
		}
	}
	return
}
func getPostgresConnString(container string, sslmode string) (string, error) {
	if !conf.Exists(container) {
		return "", errors.New("Container for postgres configuration not found")
	}

	username := conf.String(container+".username", "")
	password := conf.String(container+".password", "")
	host := conf.String(container+".host", "")
	port := conf.String(container+".port", "")
	db := conf.String(container+".db", "")
	return fmt.Sprintf("host=%s port=%s sslmode=%s user=%s dbname=%s password=%s", host, port, sslmode, username, db, password), nil
}

func getDefaultConn(write bool, sslModeVal ...string) (string, error) {
	var err error
	sslMode := "disable"
	if len(sslModeVal) > 0 {
		sslMode = sslModeVal[0]
	}
	if write {
		err = initMaster(sslMode)
		return connPostgresWrite, err
	} else {
		err = initSlaves(sslMode)
		if connPostgresRead == nil || len(connPostgresRead) == 0 {
			err = initMaster(sslMode)
			return connPostgresWrite, err
		}
		return connPostgresRead[rand.Intn(len(connPostgresRead))], err
	}
}

//Get MySql connection
func GetPostgresConn(writable bool, sslMode ...string) (dbConn gorm.DB, err error) {
	if isValidSSLMode(sslMode...) == true {
		var connStr string
		if connStr, err = getDefaultConn(writable, sslMode...); err != nil {
			return
		}
		if dbConn, err = gorm.Open("postgres", connStr); err != nil {
			return
		}
	} else {
		err = errors.New("Invalid sslMode given")
	}
	return
}

//check if sslmode is valid or not
func isValidSSLMode(sslMode ...string) (flag bool) {
	found := false
	if len(sslMode) > 0 {
		for _, curSSLMode := range sslMode {
			if _, exists := validSSLMode[curSSLMode]; exists == true {
				found = true
				break
			}
		}
	} else {
		return true
	}

	if found == true {
		flag = true
	}
	return
}
