package monit

import (
	"errors"
	"github.com/ooyala/go-dogstatsd"
	"github.com/imtolexo/aero/conf"
	"sync"
)

var agentObj *DataDogAgent
var once sync.Once

type DataDogAgent struct {
	Client *dogstatsd.Client
}

func (d *DataDogAgent) ClientExists() (exists bool) {
	if d.Client != nil {
		exists = true
	}
	return
}

func (d *DataDogAgent) Close() {
	d.Client.Close()
}

func (d *DataDogAgent) Count(name string, value int64, tags []string, rate float64) (err error) {
	exists := d.ClientExists()
	if exists {
		err = d.Client.Count(name, value, tags, rate)
	}
	return
}

func (d *DataDogAgent) Histogram(name string, value float64, tags []string, rate float64) (err error) {
	exists := d.ClientExists()
	if exists {
		err = d.Client.Histogram(name, value, tags, rate)
	}
	return
}

//TODO: handle errnous cases
func GetDataDogAgent() *DataDogAgent {
	once.Do(func() {
		agentObj = new(DataDogAgent)
		var errObj error
		enabled := conf.Bool("monitor.enabled", false)
		if enabled && agentObj.ClientExists() == false {
			host := conf.String("monitor.host", "")
			port := conf.String("monitor.port", "")
			if host == "" || port == "" {
				errObj = errors.New("Datadog config host and port missing")
			} else {
				client, err := dogstatsd.New(host + ":" + port)
				if err != nil {
					errObj = err
				} else {
					namespace := conf.String("monitor.namespace", "")
					if namespace != "" {
						client.Namespace = namespace
					}
					agentObj.Client = client
				}
			}
		}
		if errObj != nil {
			//log error message
		}
	})
	return agentObj
}
