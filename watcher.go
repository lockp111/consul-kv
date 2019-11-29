package consul

import (
	"bytes"
	"strings"
	"sync"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/tidwall/gjson"
)

// WatchFunc ...
type WatchFunc func(*Result)

func newWatcher(wp *watch.Plan) *watcher {
	return &watcher{
		Plan:       wp,
		lastValues: make(map[string][]byte),
	}
}

type watcher struct {
	*watch.Plan
	lastValues map[string][]byte
	sync.RWMutex
}

func (w *watcher) getValue(path string) []byte {
	w.RLock()
	defer w.RUnlock()

	return w.lastValues[path]
}

func (w *watcher) updateValue(path string, value []byte) {
	w.Lock()
	defer w.Unlock()

	if len(value) == 0 {
		delete(w.lastValues, path)
	} else {
		w.lastValues[path] = value
	}
}

func (w *watcher) hybridHandler(prefix string, handler WatchFunc) {
	w.HybridHandler = func(bp watch.BlockingParamVal, data interface{}) {
		kvPairs := data.(api.KVPairs)
		ret := &Result{}

		for _, k := range kvPairs {
			path := strings.TrimSuffix(strings.TrimPrefix(k.Key, prefix+"/"), "/")
			v := w.getValue(path)

			if len(k.Value) == 0 && len(v) == 0 {
				continue
			}

			if bytes.Equal(k.Value, v) {
				continue
			}

			ret.g = gjson.ParseBytes(k.Value)
			ret.k = path
			w.updateValue(path, k.Value)
			handler(ret)
		}
	}
}
