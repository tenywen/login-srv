package etcdmutex

import (
	log "github.com/GameGophers/nsq-logger"
	"github.com/coreos/go-etcd/etcd"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	LOCK_PATH    = "/locks/"
	TTL          = 60 // max num of secs to live for a lock
	DEFAULT_ETCD = "http://127.0.0.1:2379"
	RETRY_MAX    = 10                    // depends on concurrency
	RETRY_WAIT   = 10 * time.Millisecond // depends on how long can a lock hold
	VALUE        = "L"
	SERVICE      = "[ETCD-MUTEX-API]"
)

var (
	// etcd client pool
	_client_pool sync.Pool
)

type Mutex struct {
	key       string
	prevIndex uint64
}

func init() {
	// determine etcd hosts
	machines := []string{DEFAULT_ETCD}
	if env := os.Getenv("ETCD_HOST"); env != "" {
		machines = strings.Split(env, ";")
	}

	_client_pool.New = func() interface{} {
		return etcd.NewClient(machines)
	}
}

// unlock a key on etcd, returns false if the key cannot be successfully unlocked
func (m *Mutex) Unlock() bool {
	client := _client_pool.Get().(*etcd.Client)
	_, err := client.CompareAndDelete(LOCK_PATH+m.key, VALUE, m.prevIndex)
	if err != nil {
		log.Error(SERVICE, err)
		return false
	}
	return true
}

// lock a key on etcd and return mutex,returns nil if cannot lock the key
func Lock(key string) *Mutex {
	m := &Mutex{}
	m.key = key
	client := _client_pool.Get().(*etcd.Client)
	defer func() {
		_client_pool.Put(client)
	}()

	for i := 0; i < RETRY_MAX; i++ {
		// create a key with ttl
		resp, err := client.Create(LOCK_PATH+key, VALUE, TTL)
		if err != nil {
			log.Error(SERVICE, err)
			<-time.After(RETRY_WAIT)
			continue
		}
		// remember index
		m.prevIndex = resp.Node.ModifiedIndex
		return m
	}
	return nil
}
