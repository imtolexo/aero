package postgres

import (
	"errors"
	"log"
	"math/rand"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-pg/pg"
	"github.com/imtolexo/aero/conf"
)

var (
	dbPostgresWrite *pg.DB
	dbPostgresRead  []*pg.DB
	readDebug       Debug
	writeDebug      Debug
	QL              *QueryLogger
	MasterContainer = "database.master"
	SlaveContainer  = "database.slaves"
)

const (
	GO_PG_PACKAGE = "/github.com/go-pg"
)

//Query log model contains method names and mux for locking
type QueryLogger struct {
	methodName map[string]bool
	mux        sync.Mutex
}

//Debugger model contains DBConn on which query log will be created
type Debug struct {
	DBConn *pg.DB
}

//init master connection
func initMaster() (err error) {
	if conf.Exists(MasterContainer) {
		if dbPostgresWrite != nil {
			return
		} else {
			var postgresWriteOption pg.Options
			if postgresWriteOption, err = getPostgresOptions(MasterContainer); err == nil {
				dbPostgresWrite = pg.Connect(&postgresWriteOption)
			}
		}
	} else {
		err = errors.New("Master config does not exists")
	}
	return
}

//init slave connections
func initSlaves() (err error) {
	if conf.Exists(SlaveContainer) {
		slaves := conf.StringSlice(SlaveContainer, []string{})
		if dbPostgresRead == nil {
			dbPostgresRead = make([]*pg.DB, len(slaves))
		}
		for i, container := range slaves {
			if dbPostgresRead[i] == nil {
				var postgresReadOption pg.Options
				if postgresReadOption, err = getPostgresOptions(container); err != nil {
					break
				}
				dbPostgresRead[i] = pg.Connect(&postgresReadOption)
			}
		}
	} else {
		err = errors.New("Slaves config does not exists")
	}
	return
}

//create new master connection
func CreateMaster() (err error) {
	if conf.Exists(MasterContainer) {
		var postgresWriteOption pg.Options
		if postgresWriteOption, err = getPostgresOptions(MasterContainer); err == nil {
			dbPostgresWrite = pg.Connect(&postgresWriteOption)
		}
	} else {
		err = errors.New("Master config does not exists")
	}
	return
}

//create new slave connections
func CreateSlave() (err error) {
	if conf.Exists(SlaveContainer) {
		slaves := conf.StringSlice(SlaveContainer, []string{})
		if dbPostgresRead == nil {
			dbPostgresRead = make([]*pg.DB, len(slaves))
		}
		for i, container := range slaves {
			var postgresReadOption pg.Options
			if postgresReadOption, err = getPostgresOptions(container); err != nil {
				break
			}
			dbPostgresRead[i] = pg.Connect(&postgresReadOption)
		}
	} else {
		err = errors.New("Slaves config does not exists")
	}
	return
}

//set postgres connection options from conf
func getPostgresOptions(container string) (pgOption pg.Options, err error) {
	if !conf.Exists(container) {
		err = errors.New("Container for postgres configuration not found")
		return
	}
	host := conf.String(container+".host", "")
	port := conf.String(container+".port", "")
	addr := ""
	if host != "" && port != "" {
		addr = host + ":" + port
	}
	pgOption.Addr = addr
	pgOption.User = conf.String(container+".username", "")
	pgOption.Password = conf.String(container+".password", "")
	pgOption.Database = conf.String(container+".db", "")
	pgOption.MaxRetries = conf.Int(container+".maxRetries", 3)
	pgOption.RetryStatementTimeout = conf.Bool(container+".retryStmTimeout", false)
	return
}

//Start query logging
func startQueryLog(debug *Debug, dbConn *pg.DB) {
	if debug == nil {
		debug = &Debug{DBConn: dbConn}
		debug.LogQuery()
	} else if reflect.DeepEqual(debug.DBConn, dbConn) == false {
		debug.DBConn = dbConn
		debug.LogQuery()
	}
}

//Get postgres connection
func Conn(writable bool) (dbConn *pg.DB, err error) {
	if writable {
		err = initMaster()
		dbConn = dbPostgresWrite
		startQueryLog(&writeDebug, dbConn)
	} else {
		err = initSlaves()
		if dbPostgresRead == nil || len(dbPostgresRead) == 0 {
			err = initMaster()
			dbConn = dbPostgresWrite
			startQueryLog(&writeDebug, dbConn)
		} else {
			dbConn = dbPostgresRead[rand.Intn(len(dbPostgresRead))]
			startQueryLog(&readDebug, dbConn)
		}
	}
	return
}

//Get postgres connection by container
func ConnByContainer(container string) (*pg.DB, error) {
	if strings.HasSuffix(container, "master") == true {
		MasterContainer = container
		return Conn(true)
	} else if strings.HasSuffix(container, "slaves") == true {
		SlaveContainer = container
		return Conn(false)
	}
	return nil, errors.New("No master or slaves container found in: " + container)
}

//Print postgresql query on terminal
func (d *Debug) LogQuery() {
	d.DBConn.OnQueryProcessed(func(event *pg.QueryProcessedEvent) {
		if conf.Bool("debug_query.log_query", false) == true && QL != nil {
			methodName := methodName(4)
			if QL.methodName[methodName] == true {
				if query, err := event.FormattedQuery(); err == nil {
					var queryError string
					if event.Error != nil {
						queryError = "\nQUERY ERROR: " + event.Error.Error()
					}
					log.Printf("\nFile: %v : %v\nFunction: %v\nQuery Execution Taken: %s\n%s%s\n\n",
						event.File, event.Line, event.Func, time.Since(event.StartTime), query, queryError)
				} else {
					log.Println("LogQuery Error: " + err.Error())
				}
			}
		}
	})
}

//Get method name of function caller
func methodName(depth int) (method string) {
	for i := depth; ; i++ {
		pc, file, _, ok := runtime.Caller(i)
		if ok == false {
			break
		}
		if strings.Contains(file, GO_PG_PACKAGE) {
			continue
		}
		methodName := pkgMethod(pc)
		if ind := strings.Index(methodName, "."); ind > 0 {
			method = methodName[ind+1:]
			break
		}
	}
	return
}

//Get package method name and method pointer name from program counter
func pkgMethod(pc uintptr) (method string) {
	if f := runtime.FuncForPC(pc); f != nil {
		method = f.Name()
		if ind := strings.LastIndex(method, "/"); ind > 0 {
			method = method[ind+1:]
		}
		if ind := strings.Index(method, "."); ind > 0 {
			method = method[ind+1:]
		}
	}
	return
}

//Add methodName in queryLogger model
func (q *QueryLogger) AddMethod(methodName string) {
	q.mux.Lock()
	q.methodName[methodName] = true
	q.mux.Unlock()
}

//Remove methodName from queryLogger model
func (q *QueryLogger) RemoveMethod(methodName string) {
	q.mux.Lock()
	if _, exists := q.methodName[methodName]; exists == true {
		delete(q.methodName, methodName)
	}
	q.mux.Unlock()
}

//Create new object of query logger
func NewQueryLogger() {
	QL = &QueryLogger{methodName: make(map[string]bool)}
}
