package kv

var kvConf *Config

// Init ...
func Init(opts ...Option) error {
	if kvConf != nil {
		return nil
	}

	kvConf = NewConfig(opts...)
	return kvConf.Connect()
}

// Reset ...
func Reset(opts ...Option) error {
	kvConf.WatchStop()
	kvConf = nil
	return Init(opts...)
}

// Put ...
func Put(path string, value interface{}) error {
	return kvConf.Put(path, value)
}

// Delete ...
func Delete(path string) error {
	return kvConf.Delete(path)
}

// Get ...
func Get(keys ...string) *Result {
	return kvConf.Get(keys...)
}

// WatchStart ...
func WatchStart(path string, handler func(*Result)) error {
	return kvConf.WatchStart(path, handler)
}

// WatchStop ...
func WatchStop(path ...string) {
	kvConf.WatchStop(path...)
}
