package auth

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/imtolexo/aero/cache"
	"github.com/imtolexo/aero/db/orm"
	"net/http"
	"strconv"
	"time"
)

const TIMESTAMP_VALIDITY int = 60000
const APP_SECRET string = "TOLEXOANDROIDAPP"

//authentication error codes
const INVALID_REQUEST int = 601
const INVALID_TIMESTAMP int = 602
const INVALID_HASH int = 603
const INVALID_APP_SESSION int = 604

//time format
const DB_DATE_FORMAT string = "2006-01-02 15:04:05"

type AuthParams struct {
	AppSession string
	Timestamp  string
	Hash       string
}

type SessionInCache struct {
	CustomerId   string `json:"customer_id"`
	CustomerType string `json:"customer_type"`
	ExpiryDate   string `json:"expiry_date"`
}

type Session struct {
	CustomerId   int       `json:"customer_id"`
	CustomerType string    `json:"customer_type"`
	ExpiryDate   time.Time `json:"expiry_date"`
}

//Authenticate request from tokens in header
func AuthenticateFromHeader(r *http.Request) (bool, string) {
	code, s := validateRequestHeader(r)

	//if valid request, then set session details in header
	if code == 200 {
		r.Header.Set("ModSession", s)
		return true, ""
	}

	errMessage := getAuthenticationError(code)

	return false, string(errMessage[:])
}

func validateRequestHeader(r *http.Request) (int, string) {
	a := getAuthParamsFromHeader(r)
	code := CheckTimestamp(a)
	if code != 200 {
		return code, ""
	}

	code = CheckHash(a)
	if code != 200 {
		return code, ""
	}

	code, session := getSessionInfo(a)
	if code != 200 {
		return code, ""
	}

	se, err := json.Marshal(session)
	if err == nil {
		return 200, string(se[:]) //user authenticated
	}

	return INVALID_REQUEST, ""
}

func getAuthParamsFromHeader(r *http.Request) (authParam AuthParams) {
	if len(r.Header["Xappsession"]) > 0 {
		authParam.AppSession = r.Header["Xappsession"][0]
	} else {
		authParam.AppSession = ""
	}
	if len(r.Header["Xtimestamp"]) > 0 {
		authParam.Timestamp = r.Header["Xtimestamp"][0]
	} else {
		authParam.Timestamp = ""
	}
	if len(r.Header["Xhash"]) > 0 {
		authParam.Hash = r.Header["Xhash"][0]
	} else {
		authParam.Hash = ""
	}

	return authParam
}

func CheckTimestamp(a AuthParams) (code int) {
	today := time.Now()
	i, err := strconv.ParseInt(a.Timestamp, 10, 64)
	if err != nil {
		return INVALID_TIMESTAMP
	}
	t := time.Unix(i, 0)

	duration := -1
	if err == nil {
		duration = int(today.Sub(t).Minutes())
	}

	if duration < 0 { //convert time lapse to positive integer
		duration = -duration
	}
	if duration > TIMESTAMP_VALIDITY {
		return INVALID_TIMESTAMP
	}
	return 200
}

func CheckHash(a AuthParams) (code int) {
	data := a.Timestamp + "|" + a.AppSession + "|" + APP_SECRET
	h := md5.New()
	h.Write([]byte(data))
	hash := hex.EncodeToString(h.Sum(nil))

	if hash != a.Hash {
		return INVALID_HASH
	}
	return 200

}

func getSessionInfo(a AuthParams) (code int, s Session) {
	s = Session{}
	s, ok := getSessionDetailsFromCache(a.AppSession)
	if ok {
		return 200, s
	}

	s, ok = getSessionDetailsFromDb(a.AppSession)
	if ok {
		saveSessionInCache(a.AppSession, s)
		return 200, s
	}

	return INVALID_APP_SESSION, s
}

func getSessionDetailsFromCache(appSession string) (s Session, ok bool) {
	c := cache.RedisFromConfig("session")
	key := "APPSESSION-" + appSession
	val, err := c.Get(key)
	if err == nil {
		sc := SessionInCache{}
		err := json.Unmarshal(val, &sc)
		if err == nil {
			s.CustomerId, err = strconv.Atoi(sc.CustomerId)
			s.CustomerType = sc.CustomerType
			t, err := time.Parse(DB_DATE_FORMAT, sc.ExpiryDate)
			if err == nil {
				s.ExpiryDate = t
			}
			return s, true
		}
	}
	return s, false
}

func saveSessionInCache(appSession string, s Session) {
	c := cache.RedisFromConfig("session")
	key := "APPSESSION-" + appSession

	//create json compatible with php session
	sc := SessionInCache{}
	sc.CustomerId = strconv.Itoa(s.CustomerId)
	sc.CustomerType = s.CustomerType
	sc.ExpiryDate = s.ExpiryDate.Format(DB_DATE_FORMAT)

	val, _ := json.Marshal(sc)
	ttl := s.ExpiryDate.Sub(time.Now())
	c.Set(key, val, ttl)
}

func getSessionDetailsFromDb(appSession string) (s Session, ok bool) {

	db := orm.Get(false)

	sql := "Select customer_id, customer_type, expiry_date from customer_auth where app_session = ? and status = 'active' and expiry_date > now()"
	db.Raw(sql, appSession).Row().Scan(&s.CustomerId, &s.CustomerType, &s.ExpiryDate)

	if s.CustomerType != "" { //valid session

		return s, true
	} else {
		return s, false
	}
}

func getAuthenticationError(errCode int) (out []byte) {
	response := make(map[string]interface{})
	if errCode != 200 {
		e := make(map[string]interface{})
		e["error_code"] = errCode
		e["error_message"] = getErrorMessage(errCode)

		response["status"] = false
		response["response_code"] = INVALID_REQUEST
		response["response_message"] = getErrorMessage(601)
		response["error"] = e //specific error code

	}
	out, _ = json.Marshal(response)
	return out
}

func getErrorMessage(errCode int) (errMsg string) {
	switch errCode {
	case INVALID_REQUEST:
		errMsg = "Invalid Request"
	case INVALID_TIMESTAMP:
		errMsg = "Invalid Timestamp"
	case INVALID_HASH:
		errMsg = "Invalid Hash"
	case INVALID_APP_SESSION:
		errMsg = "Invalid App Session"
	default:
		errMsg = ""
	}
	return errMsg
}
