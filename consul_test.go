package consul

import (
	"testing"
)

func TestBytes(t *testing.T) {
	var bs = make([]byte, 0)

	if len(bs) != 0 {
		t.Error("bs len not zero")
		// return
	}

	t.Log(bs)

	s := string(bs)
	if len(s) != 0 {
		t.Error("s len not zero")
	}

	t.Log(s)
}
