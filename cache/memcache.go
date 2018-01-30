package cache

import (
	"github.com/kklis/gomemcache"
	"github.com/imtolexo/aero/conf"
	"github.com/imtolexo/aero/panik"
	"time"
)

type memCache struct {
	mc *gomemcache.Memcache
}

func NewMemcache(host string, port int) Cacher {
	serv, err := gomemcache.Connect(host, port)
	panik.On(err)

	m := memCache{
		mc: serv,
	}
	return m
	// TODO: close port on destruction
}

// memcache:
// - host
// - port
func MemcacheFromConfig(container string) Cacher {
	host := conf.String(container+".host", "")
	panik.If(host == "", "memcache host not specified")

	port := conf.Int(container+".port", 0)
	panik.If(port == 0, "memcache port not specified")

	return NewMemcache(host, port)
}

func (c memCache) Set(key string, data []byte, expireIn time.Duration) {
	c.mc.Set(getIndex(key), data, 0, int64(expireIn.Seconds()))
}

func (c memCache) Get(key string) ([]byte, error) {
	data, _, err := c.mc.Get(getIndex(key))
	if err != nil {
		return nil, err
	} else {
		return data, nil
	}
}
