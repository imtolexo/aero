package monit

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/imtolexo/aero/conf"
)

var (
	logFp      *os.File
	fileErr    error
	logger     *log.Logger
	prevDay    int
	currentDay int
	isLog      bool
	syncOnce   sync.Once
)

func createPanicLog(sTime time.Time, panicMsg interface{}) {
	path := conf.String("logs.panic_log", "panic_log")
	path = fmt.Sprintf("%s_%d-%d-%d.log", path, sTime.Day(), sTime.Month(), sTime.Year())
	if logFp, fileErr = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); fileErr != nil {
		fmt.Println("Could not create the panic log file")
		panic(panicMsg)
	}
}

func logPanic(panicMsg interface{}, serviceID string, requestURI string, curTime time.Time) {
	logger = log.New(logFp, "\n", log.Lshortfile)
	logger.Print("Time: ", curTime, "  serviceID: ", serviceID, "  requestURI: ", requestURI, "\n", string(debug.Stack()))
	logger.Panic(panicMsg)
}

func PanicLogger(panicMsg interface{}, serviceID string, requestURI string, curTime time.Time) {

	syncOnce.Do(func() {
		isLog = conf.Bool("monitor.panic_log", false)
	})

	if isLog == true {
		sTime := time.Now().UTC()
		currentDay = sTime.Day()
		if currentDay != prevDay {
			if prevDay != 0 && logFp != nil {
				logFp.Close()
			}
			createPanicLog(sTime, panicMsg)
			prevDay = currentDay
		}
		if logFp != nil {
			logPanic(panicMsg, serviceID, requestURI, curTime)
		} else {
			fmt.Println("lost the pointer to the log file, creating again")
			createPanicLog(sTime, panicMsg)
			logPanic(panicMsg, serviceID, requestURI, curTime)
		}
	} else {
		panic(panicMsg)
	}
}
