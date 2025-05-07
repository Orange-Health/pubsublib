package aws

import (
	"context"
	"log"
	"sync"

	"github.com/go-redis/redis/v8"
)

// example picked up from github redis example to delete keys efficiently without TTL
type KeyChecker struct {
	redisClient *redis.Client
	batchSize   int
	keyChannel  chan string
	wg          sync.WaitGroup
	stopChannel chan struct{}
}

func NewKeyChecker(redisClient *redis.Client, batchSize int) *KeyChecker {
	return &KeyChecker{
		redisClient: redisClient,
		batchSize:   batchSize,
		keyChannel:  make(chan string, batchSize),
		stopChannel: make(chan struct{}),
	}
}

func (kc *KeyChecker) Start(ctx context.Context) {
	kc.wg.Add(1)
	go func() {
		defer kc.wg.Done()
		var keysBatch []string

		for {
			select {
			case key := <-kc.keyChannel:
				keysBatch = append(keysBatch, key)
				if len(keysBatch) >= kc.batchSize {
					kc.processBatch(ctx, keysBatch)
					keysBatch = nil
				}
			case <-kc.stopChannel:
				if len(keysBatch) > 0 {
					kc.processBatch(ctx, keysBatch)
				}
				return
			}
		}
	}()
}

func (kc *KeyChecker) Add(key string) {
	kc.keyChannel <- key
}

func (kc *KeyChecker) Stop() {
	close(kc.stopChannel)
	kc.wg.Wait()
}

func (kc *KeyChecker) processBatch(ctx context.Context, keys []string) {
	pipe := kc.redisClient.Pipeline()
	defer pipe.Close()

	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Error deleting keys in batch: %v", err)
	} else {
		log.Printf("Successfully deleted %d keys", len(keys))
	}
}
