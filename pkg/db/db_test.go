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

	no := db.SIsMember(name, []byte("projects"), []byte("x"))
	if no == true {
		t.Fatal("x not in projects")
	}
	no = db.SIsMember(name, []byte("projects"), []byte("xxx"))
	if no == true {
		t.Fatal("xxx not in projects")
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

func TestTransaction(t *testing.T) {
	db, err := New("~/.rtfd.cfg")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	name := "pipeline"

	k0 := []byte("test")
	v0 := []byte("v0")

	k1 := []byte("hash")
	v1 := []byte("v1")

	k2 := []byte("list")
	v2 := []byte("v2")

	k3 := []byte("set")
	v3 := []byte("v3")

	k4 := []byte("deleted")
	v4 := []byte("v4")

	// 添加数据，测试管道删除操作
	err = db.Set(name, k0, v0)
	if err != nil {
		t.Fatal(err)
	}
	err = db.SAdd(name, k4, v4)
	if err != nil {
		t.Fatal(err)
	}

	// 管道开始事务
	tc, err := db.Pipeline()
	if err != nil {
		t.Fatal(err)
	}

	err = tc.Delete(name, k0)
	if err != nil {
		t.Fatal(err)
	}

	err = tc.Set(name, k1, v1)
	if err != nil {
		t.Fatal(err)
	}

	err = tc.RPush(name, k2, v2)
	if err != nil {
		t.Fatal(err)
	}

	err = tc.SAdd(name, k3, v3)
	if err != nil {
		t.Fatal(err)
	}
	err = tc.SRem(name, k4, v4)
	if err != nil {
		t.Fatal(err)
	}

	// 结束管道，执行命令
	err = tc.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// 测试 k0被删除  k4删除v4 k1值为v1 k2长度大于0 v3在k3中
	if db.Has(name, k0) != false {
		t.Fatal("key should be not found for k0")
	}
	if db.SIsMember(name, k4, v4) != false {
		t.Fatal("item should be not found in k4")
	}

	if db.Has(name, k1) != true {
		t.Fatal("key should be found for k1")
	}
	_v1, err := db.Get(name, k1)
	if string(_v1) != string(v1) {
		t.Fatal("pipe set fail")
	}
	num, _ := db.LSize(name, k2)
	if num <= 0 {
		t.Fatal("k2 should has values")
	}

	if db.SHasKey(name, k3) == false {
		t.Fatal("k3 should in bucket")
	}
	if db.SIsMember(name, k3, v3) != true {
		t.Fatal("v3 should in k3")
	}
	size, _ := db.SCard(name, k3)
	if size != 1 {
		t.Fatal("k3 length should be equal 1")
	}

}
