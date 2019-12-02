package kv

import (
	"log"
	"testing"
)

var s *Config

func init() {
	var err error
	s = NewConfig(WithPrefix("kvTest"))
	err = s.Connect()
	if err != nil {
		log.Fatalln(err)
	}
}

func TestPut1(t *testing.T) {
	a := &struct {
		Put string `json:"put"`
	}{"put1"}
	err := s.Put("test1", a)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestPut2(t *testing.T) {
	err := s.Put("test2", "put2")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestGet(t *testing.T) {
	ret := s.Get("test")
	if ret.Err() != nil {
		t.Error(ret.Err())
		return
	}

	t.Log(ret.Get("Test").String())
	t.Fail()
}

func TestScan(t *testing.T) {
	type Permission struct {
		Procy    int
		Rights   int
		Services string
		LogReq   bool
		LogRsp   bool
	}

	ret := s.Get("c2c")
	m := make(map[string]*Permission)
	ret.Get("Permissions").Scan(&m)
	t.Log(m)
	t.Fail()
}

func TestList(t *testing.T) {
	ls, err := s.list()
	if err != nil {
		t.Error(err)
		return
	}

	for _, v := range ls {
		t.Log(v)
	}
	t.Fail()
}
