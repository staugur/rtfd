package db

import (
	"encoding/json"
	"testing"
)

type options struct {
	Path string
	Has  bool
}

func TestDB(t *testing.T) {
	db, err := New("~/.rtfd.cfg")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// set/get/delete
	name := "test"
	k1 := []byte("hash")
	v1 := []byte("v1")
	err = db.Set(name, k1, v1)
	if err != nil {
		t.Fatal(err)
	}
	_v1, err := db.Get(name, k1)
	if err != nil {
		t.Fatal(err)
	}
	if string(v1) != string(_v1) {
		t.Fatal("value not equal")
	}
	err = db.Delete(name, k1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Get(name, k1)
	if err == nil {
		t.Fatal("key not found")
	}

	// set/get json
	jk := []byte("json")
	opt := options{"I am path", true}
	data, err := json.Marshal(opt)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Set(name, jk, data)
	if err != nil {
		t.Fatal(err)
	}
	dataFromDB, err := db.Get(name, jk)
	if err != nil {
		t.Fatal(err)
	}
	var opt2 options
	err = json.Unmarshal(dataFromDB, &opt2)
	if err != nil {
		t.Fatal(err)
	}
	if opt2.Has != true {
		t.Fatal("Unmarshal options error")
	}

	// list
	k2 := []byte("list")
	v2 := []byte("v2")
	err = db.RPush(name, k2, v2)
	if err != nil {
		t.Fatal(err)
	}

	lv, err := db.LRange(name, k2, 0, -1)
	if err != nil {
		t.Fatal(err)
	}
	length, err := db.LSize(name, k2)
	if length != len(lv) {
		t.Fatal("list key number should be equal")
	}
	_v2, err := db.RPop(name, k2)
	if err != nil {
		t.Fatal(err)
	}
	if string(_v2) != string(v2) {
		t.Fatal("k2 value should be equal")
	}
	delLen, _ := db.LSize(name, k2)
	if delLen != (length - 1) {
		t.Fatal("RPop error")
	}

	// set
	k3 := []byte("set")
	v3 := []byte("v3")
	has := db.SHasKey(name, []byte("non_exists_key"))
	if has == true {
		t.Fatal("set key should not exist")
	}
	err = db.SAdd(name, k3, v3)
	if err != nil {
		t.Fatal(err)
	}
	has = db.SHasKey(name, k3)
	if has != true {
		t.Fatal("set key should exist")
	}

	isMember := db.SIsMember(name, k3, v3)
	if isMember != true {
		t.Fatal("the value of key should be in k3")
	}

	v3data, err := db.SMembers(name, k3)
	if err != nil {
		t.Fatal(err)
	}
	v3len, err := db.SCard(name, k3)
	if err != nil {
		t.Fatal(err)
	}

	if len(v3data) != v3len {
		t.Fatal("k3 number should be equal")
	}

	err = db.SRem(name, k3, v3)
	if err != nil {
		t.Fatal(err)
	}
	v3data, _ = db.SMembers(name, k3)
	if len(v3data) != (v3len - 1) {
		t.Fatal("srem error")
	}

}
