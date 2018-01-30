package monit

import (
	"fmt"
	"github.com/imtolexo/aero/conf"
	"github.com/imtolexo/aero/panik"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"
)

type MonitorParams struct {
	ServiceId    string
	RespTime     float64
	ResponseCode int64
	CacheHit     bool
}

func MonitorMe(params MonitorParams) {
	if params.ServiceId != "" {
		dataDogAgent := GetDataDogAgent()

		dataDogAgent.Count("throughput", 1, nil, 1)
		dataDogAgent.Count(params.ServiceId, 1, nil, 1)

		dataDogAgent.Histogram("resptime", params.RespTime, nil, 1)
		dataDogAgent.Histogram(params.ServiceId+".resptime", params.RespTime, nil, 1)

		intervalTag := GetTimeIntervalTag(params.RespTime)
		dataDogAgent.Histogram("resptimeinterval", params.RespTime, intervalTag, 1)
		dataDogAgent.Histogram(params.ServiceId+".resptimeinterval", params.RespTime, intervalTag, 1)

		if params.CacheHit {
			dataDogAgent.Count("cachehit", 1, nil, 1)
			dataDogAgent.Count(params.ServiceId+".cachehit", 1, nil, 1)
			dataDogAgent.Histogram("resptime.c", params.RespTime, nil, 1)
			dataDogAgent.Histogram(params.ServiceId+".resptime.c", params.RespTime, nil, 1)
		} else {
			dataDogAgent.Histogram("resptime.nc", params.RespTime, nil, 1)
			dataDogAgent.Histogram(params.ServiceId+".resptime.nc", params.RespTime, nil, 1)

		}

		if params.ResponseCode > 0 {
			statusCode := FormatHttpStatusCode(int64(params.ResponseCode))
			dataDogAgent.Count(statusCode, 1, nil, 1)
			dataDogAgent.Count(params.ServiceId+"."+statusCode, 1, nil, 1)
		}

		enablelog := conf.Bool("monitor.enablelog", false)
		slowLogTiming := conf.Float64("monitor.slowLogTiming", 0.0)
		if enablelog || (slowLogTiming > 0.0 && params.RespTime > slowLogTiming) {
			logfile := conf.String("monitor.filelog", "")
			if logfile != "" {
				t := time.Now()
				logfileSuffix := fmt.Sprintf("%d-%d-%d", t.Month(), t.Day(), t.Hour())
				f, err := os.OpenFile(logfile+logfileSuffix+".csv", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				panik.On(err)
				l := log.New(f, "", log.LstdFlags)
				l.Printf("%s %f %s %d %t", params.ServiceId, params.RespTime, intervalTag, params.ResponseCode, params.CacheHit)
			}
		}
	}
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	fmt.Fprintf(w, "%s", "4041 page not found")

	dataDogAgent := GetDataDogAgent()
	var statusCode int64 = 4041
	dataDogAgent.Count(FormatHttpStatusCode(statusCode), 1, nil, 1)
}

func FormatHttpStatusCode(httpStatusCode int64) string {
	return "http_" + strconv.FormatInt(httpStatusCode, 10)
}

func GetTimeIntervalTag(respTime float64) (ret []string) {
	interval := conf.String("monitor.interval", "")
	if interval != "" {
		intervalArr := strings.Split(interval, ",")
		var lowerLimit, higherLimit float64
		var matched bool
		for i := range intervalArr {
			higherLimit, _ = strconv.ParseFloat(intervalArr[i], 64)
			if respTime <= higherLimit {
				matched = true
				break
			} else {
				lowerLimit = higherLimit
			}
		}
		var intervalGroup string
		if matched == true {
			intervalGroup = "from_" + strconv.FormatInt(int64(lowerLimit), 10) + "_to_" + strconv.FormatInt(int64(higherLimit), 10)
		} else {
			intervalGroup = "above_" + strconv.FormatInt(int64(higherLimit), 10)
		}
		ret = append(ret, intervalGroup)
	}
	return
}

func ModRecorder() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)

			for k, v := range rec.Header() {
				w.Header()[k] = v
			}

			len := rec.Body.Len()
			w.Header().Set("Content-Length", strconv.Itoa(len))
			w.WriteHeader(rec.Code)
			w.Write(rec.Body.Bytes())

			go func() {
				dur := time.Since(start).Seconds() * 1000
				intervalTag := GetTimeIntervalTag(float64(dur))
				statusCode := FormatHttpStatusCode(int64(rec.Code))

				dataDogAgent := GetDataDogAgent()
				dataDogAgent.Count("throughput", 1, nil, 1)
				dataDogAgent.Count(statusCode, 1, nil, 1)
				dataDogAgent.Histogram("resptime", dur, nil, 1)
				dataDogAgent.Histogram("resptimeinterval", dur, intervalTag, 1)
			}()

		})
	}
}
