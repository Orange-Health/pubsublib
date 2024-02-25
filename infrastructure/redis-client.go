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

func NewRedisDatabase(address, password string, db, poolSize, minIdleConn int) (*RedisDatabase, error) {
	/*
		A global redis client to be used for PUBSUB
	*/
	if Rdb != nil {
		return Rdb, nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:         address,
		Password:     password,
		DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConn,
	})
	if err := client.Ping(Ctx).Err(); err != nil {
		return nil, err
	}
	Rdb = &RedisDatabase{
		Client: client,
		Ctx:    Ctx,
	}
	return Rdb, nil
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
