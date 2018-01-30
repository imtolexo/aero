package tmongo

import (
	"errors"
	"fmt"
	"github.com/imtolexo/aero/conf"
	"gopkg.in/mgo.v2"
	"time"
)

type MgoOption struct {
	Timeout time.Duration
}

// create the mongo connection string
func getMongoConnStr(container string) (conn string, db string, err error) {
	username := conf.String(container+".username", "")
	password := conf.String(container+".password", "")
	host := conf.String(container+".host", "")
	port := conf.String(container+".port", "")
	db = conf.String(container+".db", "")
	replicas := conf.String(container+".replicas", "")
	options := conf.String(container+".options", "")

	if db == "" {
		err = errors.New("mongo database name missing")
		return
	}

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

	conn = fmt.Sprintf("mongodb://%s%s%s%s/%s%s", auth, host, port, replicas, db, options)
	return
}

// validate the container string
func validateContainer(container string) (db string, err error) {

	if conf.Exists(container) == false {
		err = errors.New("mongo configuration missing")
		return
	}
	db = container
	return
}

// create mongo connection
// TODO introduce parameter for connection additional settings like socket timeout
func GetMongoConn(container string, param ...MgoOption) (sess *mgo.Session, mdb string, err error) {
	var db string
	if db, err = validateContainer(container); err == nil {
		var conn string
		if conn, mdb, err = getMongoConnStr(db); err != nil {
			return
		}

		pLen := len(param)
		if pLen == 0 {
			sess, err = mgo.Dial(conn)
		} else if pLen == 1 {
			if param[0].Timeout != time.Duration(0) {
				sess, err = mgo.DialWithTimeout(conn, param[0].Timeout)
			} else {
				sess, err = mgo.Dial(conn)
			}
		} else {
			err = errors.New("More than one MgoOption structures not supported")
		}

		if err == nil {
			sess.SetMode(mgo.Monotonic, true)
		}
	}
	return
}
