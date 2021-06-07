package kv

import (
	"encoding/json"
	"time"

	"github.com/tidwall/gjson"
)

// Result ...
type Result struct {
	err  error
	key  string
	data gjson.Result
}

// Err ...
func (r *Result) Err() error {
	return r.err
}

// Get ...
func (r *Result) Get(path string) *Result {
	r.data = r.data.Get(path)
	return r
}

// Scan ...
func (r *Result) Scan(x interface{}) error {
	return json.Unmarshal([]byte(r.data.Raw), x)
}

// Float ...
func (r *Result) Float(defaultValue ...float64) float64 {
	var df float64
	if len(defaultValue) != 0 {
		df = defaultValue[0]
	}

	if !r.data.Exists() {
		return df
	}

	return r.data.Float()
}

// Int ...
func (r *Result) Int(defaultValue ...int64) int64 {
	var df int64
	if len(defaultValue) != 0 {
		df = defaultValue[0]
	}

	if !r.data.Exists() {
		return df
	}

	return r.data.Int()
}

// Uint ...
func (r *Result) Uint(defaultValue ...uint64) uint64 {
	var df uint64
	if len(defaultValue) != 0 {
		df = defaultValue[0]
	}

	if !r.data.Exists() {
		return df
	}

	return r.data.Uint()
}

// Bool ...
func (r *Result) Bool(defaultValue ...bool) bool {
	var df bool
	if len(defaultValue) != 0 {
		df = defaultValue[0]
	}

	if !r.data.Exists() {
		return df
	}

	return r.data.Bool()
}

// Bytes ...
func (r *Result) Bytes(defaultValue ...[]byte) []byte {
	var df []byte
	if len(defaultValue) != 0 {
		df = defaultValue[0]
	}

	if !r.data.Exists() {
		return df
	}

	return []byte(r.data.Raw)
}

// String
func (r *Result) String(defaultValue ...string) string {
	var df string
	if len(defaultValue) != 0 {
		df = defaultValue[0]
	}

	if !r.data.Exists() {
		return df
	}

	return r.data.String()
}

// Time ...
func (r *Result) Time(defaultValue ...time.Time) time.Time {
	var df time.Time
	if len(defaultValue) != 0 {
		df = defaultValue[0]
	}

	if !r.data.Exists() {
		return df
	}

	return r.data.Time()
}

// Key ...
func (r *Result) Key() string {
	return r.key
}
