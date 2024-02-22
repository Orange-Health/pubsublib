package infrastructure

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisDatabase struct {
	Client *redis.Client
	Ctx    context.Context
}

var (
	Ctx       = context.Background()
	Rdb       *RedisDatabase
	keyPrefix = "PUBSUB"
)

func PubSubRedisClient(redisAddr, redisPass string, redisDb, redisPoolSize, redisMinIdleConn int) *RedisDatabase {
	/*
		A global redis client to be used in OMS for PUBSUB only
	*/
	if Rdb != nil {
		return Rdb
	}
	client := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPass,
		DB:           redisDb,
		PoolSize:     redisPoolSize,
		MinIdleConns: redisMinIdleConn,
	})
	Rdb = &RedisDatabase{
		Client: client,
		Ctx:    Ctx,
	}
	return Rdb
}

// Set a key/value pair in redis with expiry time in minutes
func (rdb RedisDatabase) Set(key string, data interface{}, expiryTime int) error {
	key = keyPrefix + ":" + key

	value, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = rdb.Client.Set(rdb.Ctx, key, value, time.Duration(expiryTime)*time.Minute).Err()
	if err != nil {
		return err
	}

	return nil
}

// Get gets a key from redis and unmarshals it to the destination
func (rdb RedisDatabase) Get(key string, dest interface{}) error {
	key = keyPrefix + ":" + key
	val, err := rdb.Client.Get(rdb.Ctx, key).Bytes()
	if err == redis.Nil {
		log.Println("Key does not exist:", key)
		return err
	} else if err != nil {
		log.Println("Error while fetching data from redis. Key,value", key, val)
		return err
	}
	err = json.Unmarshal(val, dest)
	if err != nil {
		log.Println("Error: ", err)
	}
	return err
}

// Delete delete a key
func (rdb RedisDatabase) Delete(key string) error {
	key = keyPrefix + ":" + key
	log.Println("Deleting following key from redis: ", key)
	return rdb.Client.Del(rdb.Ctx, key).Err()
}
