package consul

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func init() {
	gjson.UnmarshalValidationEnabled(true)
}

// Option ...
type Option func(opt *Config)

// NewConfig ...
func NewConfig(opts ...Option) *Config {
	c := &Config{
		conf:     api.DefaultConfig(),
		watchers: make(map[string]*watcher),
	}

	for _, o := range opts {
		o(c)
	}

	return c
}

// WithPrefix ...
func WithPrefix(prefix string) Option {
	return func(c *Config) {
		c.prefix = prefix
	}
}

// WithAddress ...
func WithAddress(address string) Option {
	return func(c *Config) {
		c.conf.Address = address
	}
}

// WithAuth ...
func WithAuth(username, password string) Option {
	return func(c *Config) {
		c.conf.HttpAuth = &api.HttpBasicAuth{
			Username: username,
			Password: password,
		}
	}
}

// WithToken ...
func WithToken(token string) Option {
	return func(c *Config) {
		c.conf.Token = token
	}
}

// Config ...
type Config struct {
	prefix   string
	kv       *api.KV
	conf     *api.Config
	watchers map[string]*watcher
	sync.RWMutex
}

func (c *Config) checkWatcher(path string) error {
	c.RLock()
	defer c.RUnlock()

	if _, ok := c.watchers[c.path(path)]; ok {
		return errors.New("watch path already exist")
	}

	return nil
}

func (c *Config) getWatcher(path string) *watcher {
	c.RLock()
	defer c.RUnlock()

	return c.watchers[c.path(path)]
}

func (c *Config) addWatcher(path string, w *watcher) {
	c.Lock()
	defer c.Unlock()

	c.watchers[c.path(path)] = w
}

func (c *Config) removeWatcher(path string) {
	c.Lock()
	defer c.Unlock()

	delete(c.watchers, c.path(path))
}

func (c *Config) watcherLoop(path string) {
	log.WithField("path", path).Info("watcher start...")

	for {
		wp := c.getWatcher(path)
		if wp == nil {
			log.WithField("path", path).Info("watcher stop")
			return
		}

		if err := wp.RunWithConfig(c.conf.Address, c.conf); err != nil {
			log.WithField("path", path).Warning("watcher error")
		}

		time.Sleep(time.Second * 5)
	}
}

func (c *Config) path(keys ...string) string {
	if len(keys) == 0 {
		return c.prefix
	}

	if len(keys[0]) == 0 {
		return c.prefix
	}

	if len(c.prefix) == 0 {
		return strings.Join(keys, "/")
	}

	return c.prefix + "/" + strings.Join(keys, "/")
}

func (c *Config) list() ([]string, error) {
	keyPairs, _, err := c.kv.List(c.prefix, nil)
	if err != nil {
		return nil, err
	}

	list := make([]string, 0, len(keyPairs))
	for _, v := range keyPairs {
		if len(v.Value) != 0 {
			list = append(list, v.Key)
		}
	}

	return list, nil
}

// Connect ...
func (c *Config) Connect() error {
	client, err := api.NewClient(c.conf)
	if err != nil {
		return err
	}

	c.kv = client.KV()
	return nil
}

// Put ...
func (c *Config) Put(path string, value interface{}) error {
	var (
		data []byte
		err  error
	)

	data, err = json.Marshal(value)
	if err != nil {
		data = []byte(fmt.Sprintf("%v", value))
	}

	p := &api.KVPair{Key: c.path(path), Value: data}
	_, err = c.kv.Put(p, nil)
	return err
}

// Get ...
func (c *Config) Get(keys ...string) (ret *Result) {
	var (
		path   = c.path(keys...) + "/"
		fields []string
	)

	ret = &Result{}
	ks, err := c.list()
	if err != nil {
		ret.err = err
		return
	}

	for _, k := range ks {
		if !strings.HasPrefix(path, k+"/") {
			ret.err = errors.New("key not found")
			continue
		}

		field := strings.TrimSuffix(strings.TrimPrefix(path, k+"/"), "/")
		if len(field) != 0 {
			fields = strings.Split(field, "/")
		}

		kvPair, _, err := c.kv.Get(k, nil)
		ret.g = gjson.ParseBytes(kvPair.Value)
		ret.k = strings.TrimSuffix(strings.TrimPrefix(path, c.prefix+"/"), "/")
		ret.err = err
		break
	}

	if len(fields) == 0 {
		return
	}

	ret.g = ret.g.Get(strings.Join(fields, "."))
	ret.k += "/" + strings.Join(fields, "/")
	return
}

// Delete ...
func (c *Config) Delete(path string) error {
	_, err := c.kv.Delete(c.path(path), nil)
	return err
}

// WatchStart ...
func (c *Config) WatchStart(path string, handler WatchFunc) error {
	if err := c.checkWatcher(path); err != nil {
		return err
	}

	wp, err := watch.Parse(map[string]interface{}{"type": "keyprefix", "prefix": c.path(path)})
	if err != nil {
		return err
	}

	watcher := newWatcher(wp)
	watcher.hybridHandler(c.prefix, handler)
	c.addWatcher(path, watcher)

	go c.watcherLoop(path)
	return nil
}

// WatchStop ...
func (c *Config) WatchStop(path string) {
	wp := c.getWatcher(path)
	if wp == nil {
		log.WithField("path", path).Info("watcher already stop")
		return
	}

	wp.Stop()
	c.removeWatcher(path)
	log.WithField("path", path).Info("watcher stopping...")
}
