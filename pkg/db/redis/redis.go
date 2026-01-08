package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bsm/redislock"
	"github.com/redis/go-redis/v9"
)

var (
	ErrLockNotObtained = errors.New("redis: 无法获取分布式锁")
	ErrLockReleased    = errors.New("redis: 锁已释放或不存在")
)

type Config struct {
	Host     string
	Port     string
	Password string
	Db       int
}

// RedisClient 封装了基础操作和分布式锁
type RedisClient struct {
	Client *redis.Client
	Locker *redislock.Client
}

func NewRedisClient(cf *Config) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cf.Host, cf.Port),
		Password: cf.Password,
		DB:       cf.Db,
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &RedisClient{
		Client: rdb,
		Locker: redislock.New(rdb),
	}, nil
}

// ======================== 基础操作 (Basic Operations) ========================

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.Client.Exists(ctx, key).Result()
	return n > 0, err
}

func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.Client.Incr(ctx, key).Result()
}

// ======================== 对象/列表操作 (JSON) ========================

// SetObj 自动将结构体序列化为 JSON 存储
func (r *RedisClient) SetObj(ctx context.Context, key string, obj interface{}, expiration time.Duration) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, key, data, expiration).Err()
}

// GetObj 自动将 JSON 反序列化到结构体对象中
func (r *RedisClient) GetObj(ctx context.Context, key string, obj interface{}) error {
	data, err := r.Client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

// ======================== 过期时间操作 (Expiration) ========================

// SetExpire 设置任意 Key 的过期时间
func (r *RedisClient) SetExpire(ctx context.Context, key string, expiration time.Duration) error {
	return r.Client.Expire(ctx, key, expiration).Err()
}

// GetTTL 获取 Key 的剩余生存时间
func (r *RedisClient) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return r.Client.TTL(ctx, key).Result()
}

// ======================== 集合操作 (Sets) ========================
// 场景：在线玩家列表、工会成员ID、好友列表

func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.Client.SAdd(ctx, key, members...).Err()
}

func (r *RedisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	return r.Client.SRem(ctx, key, members...).Err()
}

func (r *RedisClient) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.Client.SIsMember(ctx, key, member).Result()
}

func (r *RedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.Client.SMembers(ctx, key).Result()
}

func (r *RedisClient) SCard(ctx context.Context, key string) (int64, error) {
	return r.Client.SCard(ctx, key).Result()
}

// ======================== 哈希操作 (Hashes) ========================
// 场景：玩家动态属性（金币、等级、体力），支持部分更新

func (r *RedisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.Client.HSet(ctx, key, values...).Err()
}

func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	return r.Client.HGet(ctx, key, field).Result()
}

func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.Client.HGetAll(ctx, key).Result()
}

func (r *RedisClient) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return r.Client.HIncrBy(ctx, key, field, incr).Result()
}

func (r *RedisClient) HSetWithTTL(ctx context.Context, key string, expiration time.Duration, values ...interface{}) error {
	// 使用 Pipeline 保证原子性（减少网络往返）
	pipe := r.Client.Pipeline()
	pipe.HSet(ctx, key, values...)
	pipe.Expire(ctx, key, expiration)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisClient) HSetNXWithTTL(ctx context.Context, key string, field string, value interface{}, expiration time.Duration) error {
	// HSetNX 不支持多个 field，所以用 Lua 脚本保证原子性
	script := `
		local res = redis.call('HSETNX', KEYS[1], ARGV[1], ARGV[2])
		if res == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[3])
		end
		return res
	`
	return r.Client.Eval(ctx, script, []string{key}, field, value, int(expiration.Seconds())).Err()
}

// ======================== 分布式锁增强 (Advanced Locking) ========================

// GetLock 获取锁
func (r *RedisClient) GetLock(ctx context.Context, key string, ttl time.Duration, retryCount int) (*redislock.Lock, error) {
	options := &redislock.Options{}
	if retryCount > 0 {
		options.RetryStrategy = redislock.LimitRetry(redislock.LinearBackoff(100*time.Millisecond), retryCount)
	}

	lock, err := r.Locker.Obtain(ctx, "lock:"+key, ttl, options)
	if err != nil {
		if errors.Is(err, redislock.ErrNotObtained) {
			return nil, ErrLockNotObtained
		}
		return nil, err
	}
	return lock, nil
}

// RefreshLock 锁续期。如果业务逻辑执行时间超过预期，可以手动续期
func (r *RedisClient) RefreshLock(ctx context.Context, lock *redislock.Lock, ttl time.Duration) error {
	if lock == nil {
		return ErrLockReleased
	}
	return lock.Refresh(ctx, ttl, nil)
}

// ReleaseLock 释放锁。封装了 release 逻辑并过滤掉重复释放的错误
func (r *RedisClient) ReleaseLock(ctx context.Context, lock *redislock.Lock) error {
	if lock == nil {
		return nil
	}
	err := lock.Release(ctx)
	if errors.Is(err, redislock.ErrLockNotHeld) {
		// 锁已经过期或被他人持有，忽略此错误
		return nil
	}
	return err
}

// Close 关闭连接
func (r *RedisClient) Close() error {
	return r.Client.Close()
}
