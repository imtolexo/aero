package orm

import (
	"github.com/jinzhu/gorm"
	"github.com/imtolexo/aero/db"
)

var engines map[string]gorm.DB

func init() {
	engines = make(map[string]gorm.DB)
}

func Get(writable bool) gorm.DB {
	connStr := db.GetDefaultMySqlConn(writable)
	return getOrm(connStr)
}

func GetFromConf(container string) gorm.DB {
	connStr := db.GetMySqlConnFromConfig(container)
	return getOrm(connStr)
}

func getOrm(connStr string) gorm.DB {
	var ormObj gorm.DB
	var ok bool
	var err error

	if ormObj, ok = engines[connStr]; ok {
		return ormObj
	}
	// http://go-database-sql.org/accessing.html
	// the sql.DB object is designed to be long-lived
	if ormObj, err = gorm.Open("mysql", connStr); err == nil {
		if ormInit != nil {
			for _, fn := range ormInit {
				fn(&ormObj)
			}
		}
		engines[connStr] = ormObj
		return engines[connStr]
	} else {
		panic(err)
	}
}

// orm initializers
var ormInit []func(*gorm.DB)

func DoOrmInit(fn func(*gorm.DB)) {
	// TODO: use mutex
	if ormInit == nil {
		ormInit = make([]func(*gorm.DB), 0)
	}
	ormInit = append(ormInit, fn)
}
