// 封装 redigo 客户端

package db

import (
	"errors"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"tcw.im/ufc"
)

// DB 一个数据库连接结构
type DB struct {
	// key前缀
	Prefix string

	pool *redis.Pool
}

// 允许适配前缀的命令
var commandsWithPrefix = []string{
	"GET", "SET", "EXISTS", "DEL", "TYPE",
	"RPUSH", "LPOP", "RPOP", "LLEN", "LRANGE",
	"SADD", "SREM", "SISMEMBER", "SMEMBERS", "SCARD",
	"HSET", "HMSET", "HGET", "HGETALL",
}

// 将key加入到v切片头部
func kpv(key string, values []string) []interface{} {
	a := append([]string{key}, values...)

	//converting a []string to a []interface{}
	x := make([]interface{}, len(a))
	for i, v := range a {
		x[i] = v
	}

	return x
}

// New 打开一个DB连接，rawurl是redis连接串
func New(rawurl string) (c *DB, err error) {
	pool := &redis.Pool{
		// 最多有多少个空闲连接保留
		MaxIdle: 5,
		// 最多有多少活跃的连接数
		MaxActive: 500,
		// 空闲连接最长空闲时间
		IdleTimeout: 5 * time.Minute,
		// 如果超过最大连接，是报错，还是等待
		Wait: true,
		// 连接 redis 的函数
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(rawurl)
		},
	}
	return &DB{pool: pool}, nil
}

// Do 从连接池获取连接并执行命令
func (c *DB) Do(command string, args ...interface{}) (reply interface{}, err error) {
	command = strings.ToUpper(command)
	key := args[0].(string)
	if ufc.StrInSlice(command, commandsWithPrefix) && key != "" {
		args[0] = c.Prefix + key
	}
	rc := c.pool.Get()
	defer rc.Close()

	return rc.Do(command, args...)
}

// Type 查看键类型
func (c *DB) Type(key string) (string, error) {
	return redis.String(c.Do("TYPE", key))
}

// Keys 查找所有符合给定模式 pattern 的 key
func (c *DB) Keys(pattern string) ([]string, error) {
	return redis.Strings(c.Do("KEYS", pattern))
}

// Set 添加数据
func (c *DB) Set(key, value string) (bool, error) {
	ret, err := redis.String(c.Do("SET", key, value))
	if err != nil {
		return false, err
	}
	if ret == "OK" {
		return true, nil
	}
	return false, errors.New("set failed")
}

// Get 获取数据
func (c *DB) Get(key string) (string, error) {
	return redis.String(c.Do("GET", key))
}

// Exsits 判断是否有键
func (c *DB) Exsits(key string) (bool, error) {
	return redis.Bool(c.Do("EXISTS", key))
}

// Del 删除单个Key
func (c *DB) Del(key string) (bool, error) {
	return redis.Bool(c.Do("DEL", key))
}

// RPush 将一个或多个值插入到列表的尾部(最右边)
func (c *DB) RPush(key string, values ...string) (uint64, error) {
	args := kpv(key, values)
	return redis.Uint64(c.Do("RPUSH", args...))
}

// LPop 移除并返回列表的第一个元素
func (c *DB) LPop(key string) (string, error) {
	return redis.String(c.Do("LPOP", key))
}

// RPop 移除列表的最后一个元素
func (c *DB) RPop(key string) (string, error) {
	return redis.String(c.Do("RPOP", key))
}

// LLen 返回列表的长度
func (c *DB) LLen(key string) (uint64, error) {
	return redis.Uint64(c.Do("LLEN", key))
}

// LRange 返回列表中指定区间内的元素，区间以偏移量 START 和 END 指定
func (c *DB) LRange(key string, start, end int) ([]string, error) {
	return redis.Strings(c.Do("LRANGE", key, start, end))
}

// SAdd 将一个或多个成员元素加入到集合中，已经存在于集合的成员元素将被忽略
func (c *DB) SAdd(key string, members ...string) (uint64, error) {
	args := kpv(key, members)
	return redis.Uint64(c.Do("SADD", args...))
}

// SRem 移除集合中的一个或多个成员元素，不存在的成员元素会被忽略
func (c *DB) SRem(key string, members ...string) (uint64, error) {
	args := kpv(key, members)
	return redis.Uint64(c.Do("SREM", args...))
}

// SIsMember 判断元素是否是集合的成员
func (c *DB) SIsMember(key, member string) (bool, error) {
	return redis.Bool(c.Do("SISMEMBER", key, member))
}

// SMembers 返回集合中的所有的成员
func (c *DB) SMembers(key string) ([]string, error) {
	return redis.Strings(c.Do("SMEMBERS", key))
}

// SCard 返回集合中元素的数量
func (c *DB) SCard(key string) (uint64, error) {
	return redis.Uint64(c.Do("SCARD", key))
}

// HSet 为哈希表中的字段赋值
func (c *DB) HSet(name, key, value string) (uint64, error) {
	return redis.Uint64(c.Do("HSET", name, key, value))
}

// HMSet 为哈希表中的多个字段赋值
func (c *DB) HMSet(name string, hash map[string]string) (bool, error) {
	args := []interface{}{name}
	for k, v := range hash {
		args = append(args, k, v)
	}
	ret, err := c.Do("HMSET", args...)
	if err != nil {
		return false, err
	}
	if ret == "OK" {
		return true, nil
	}
	return false, errors.New("hmset failed")
}

// HGet 返回哈希表中指定字段的值
func (c *DB) HGet(name, key string) (string, error) {
	return redis.String(c.Do("HGET", name, key))
}

// HGetAll 返回哈希表中所有的字段和值
func (c *DB) HGetAll(key string) (map[string]string, error) {
	return redis.StringMap(c.Do("HGETALL", key))
}

// Pipeline 开启事务，使用 Execute 方法提交事务。
// 使用示例：
//
// t := instance.Pipeline()
//
// t.Set/RPush/Del...
//
// t.Execute()
func (c *DB) Pipeline() *TranCommand {
	rc := c.pool.Get()
	rc.Send("MULTI")
	return &TranCommand{c.Prefix, rc}
}

// TranCommand 表示事务管道
type TranCommand struct {
	prefix string
	conn   redis.Conn
}

// Send 将命令写入客户端的输出缓冲区。
func (t *TranCommand) Send(command string, args ...interface{}) error {
	command = strings.ToUpper(command)
	key := args[0].(string)
	if ufc.StrInSlice(command, commandsWithPrefix) && key != "" {
		args[0] = t.prefix + key
	}

	return t.conn.Send(command, args...)
}

// Execute 执行提交事务
func (t *TranCommand) Execute() ([]interface{}, error) {
	defer t.conn.Close()
	return redis.Values(t.conn.Do("EXEC"))
}

// Set 管道中的 Set
func (t *TranCommand) Set(key, value string) error {
	return t.Send("SET", key, value)
}

// Del 管道中的 Del
func (t *TranCommand) Del(key string) error {
	return t.Send("DEL", key)
}

// RPush 管道中的 RPush
func (t *TranCommand) RPush(key string, values ...string) error {
	args := kpv(key, values)
	return t.Send("RPUSH", args...)
}

// SAdd 管道中的 SAdd
func (t *TranCommand) SAdd(key string, members ...string) error {
	args := kpv(key, members)
	return t.Send("SADD", args...)
}

// SRem 管道中的 SRem
func (t *TranCommand) SRem(key string, members ...string) error {
	args := kpv(key, members)
	return t.Send("SREM", args...)
}
