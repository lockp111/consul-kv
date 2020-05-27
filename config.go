package kv

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// Option ...
type Option func(opt *Config)

// NewConfig ...
func NewConfig(opts ...Option) *Config {
	c := &Config{
		conf:     api.DefaultConfig(),
		watchers: make(map[string]*watcher),
		loger:    log.New(),
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

// WithLoger ...
func WithLoger(loger *log.Logger) Option {
	return func(c *Config) {
		c.loger = loger
	}
}

// Config ...
type Config struct {
	prefix   string
	kv       *api.KV
	conf     *api.Config
	watchers map[string]*watcher
	loger    *log.Logger
	sync.RWMutex
}

// CheckWatcher ...
func (c *Config) CheckWatcher(path string) error {
	c.RLock()
	defer c.RUnlock()

	if _, ok := c.watchers[c.absPath(path)]; ok {
		return ErrAlreadyWatch
	}

	return nil
}

func (c *Config) getWatcher(path string) *watcher {
	c.RLock()
	defer c.RUnlock()

	return c.watchers[c.absPath(path)]
}

func (c *Config) addWatcher(path string, w *watcher) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.watchers[c.absPath(path)]; ok {
		return ErrAlreadyWatch
	}

	c.watchers[c.absPath(path)] = w
	return nil
}

func (c *Config) removeWatcher(path string) {
	c.Lock()
	defer c.Unlock()

	delete(c.watchers, c.absPath(path))
}

func (c *Config) cleanWatcher() {
	c.Lock()
	defer c.Unlock()

	for k, w := range c.watchers {
		w.Stop()
		delete(c.watchers, k)
	}
}

func (c *Config) getAllWatchers() []*watcher {
	c.RLock()
	defer c.RUnlock()

	watchers := make([]*watcher, 0, len(c.watchers))
	for _, w := range c.watchers {
		watchers = append(watchers, w)
	}

	return watchers
}

func (c *Config) watcherLoop(path string) {
	c.loger.WithField("path", path).Info("watcher start...")

	w := c.getWatcher(path)
	if w == nil {
		c.loger.WithField("path", path).Error("watcher not found")
		return
	}

	for {
		if err := w.run(c.conf.Address, c.conf); err != nil {
			c.loger.WithField("path", path).WithError(err).Warning("watcher connect error")
			time.Sleep(time.Second * 3)
		}

		w = c.getWatcher(path)
		if w == nil {
			c.loger.WithField("path", path).Info("watcher stop")
			return
		}

		c.loger.WithField("path", path).Warning("watcher reconnect...")
	}
}

func (c *Config) absPath(keys ...string) string {
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

func (c *Config) reconnect() error {
	watchMap := c.getAllWatchers()

	for _, w := range watchMap {
		w.stop()
	}

	return c.Connect()
}

// Connect ...
func (c *Config) Connect() error {
	client, err := api.NewClient(c.conf)
	if err != nil {
		return fmt.Errorf("connect fail: %w", err)
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

	p := &api.KVPair{Key: c.absPath(path), Value: data}
	_, err = c.kv.Put(p, nil)
	if err != nil {
		return fmt.Errorf("put fail: %w", err)
	}
	return nil
}

// Get ...
func (c *Config) Get(keys ...string) (ret *Result) {
	var (
		path   = c.absPath(keys...) + "/"
		fields []string
	)

	ret = &Result{}
	ks, err := c.list()
	if err != nil {
		ret.err = fmt.Errorf("get list fail: %w", err)
		return
	}

	for _, k := range ks {
		if !strings.HasPrefix(path, k+"/") {
			ret.err = ErrKeyNotFound
			continue
		}

		field := strings.TrimSuffix(strings.TrimPrefix(path, k+"/"), "/")
		if len(field) != 0 {
			fields = strings.Split(field, "/")
		}

		kvPair, _, err := c.kv.Get(k, nil)
		ret.g = gjson.ParseBytes(kvPair.Value)
		ret.k = strings.TrimSuffix(strings.TrimPrefix(path, c.prefix+"/"), "/")
		ret.err = fmt.Errorf("get fail: %w", err)
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
	_, err := c.kv.Delete(c.absPath(path), nil)
	if err != nil {
		return fmt.Errorf("delete fail: %w", err)
	}
	return nil
}

// Watch ...
func (c *Config) Watch(path string, handler func(*Result)) error {
	watcher, err := newWatcher(c.absPath(path))
	if err != nil {
		return fmt.Errorf("watch fail: %w", err)
	}

	watcher.setHybridHandler(c.prefix, handler)
	err = c.addWatcher(path, watcher)
	if err != nil {
		return err
	}

	go c.watcherLoop(path)
	return nil
}

// StopWatch ...
func (c *Config) StopWatch(path ...string) {
	if len(path) == 0 {
		c.cleanWatcher()
		return
	}

	for _, p := range path {
		wp := c.getWatcher(p)
		if wp == nil {
			c.loger.WithField("path", p).Info("watcher already stop")
			return
		}

		c.loger.WithField("path", p).Info("watcher stopping...")
		c.removeWatcher(p)
		wp.stop()
		for !wp.IsStopped() {
		}
	}
}
