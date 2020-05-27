package kv

import "errors"

// Error defined
var (
	ErrAlreadyWatch = errors.New("watch path already exist")
	ErrKeyNotFound  = errors.New("key not found")
)
