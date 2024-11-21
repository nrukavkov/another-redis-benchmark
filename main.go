package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	redisAddr    string
	redisPass    string
	redisDB      int
	numClients   int
	numKeys      int
	keyPrefix    string
	ttl          time.Duration
	testDuration time.Duration
	setRatio     float64
	getRatio     float64
	delRatio     float64
)

func init() {
	flag.StringVar(&redisAddr, "addr", "localhost:6379", "Redis server address")
	flag.StringVar(&redisPass, "pass", "", "Redis password")
	flag.IntVar(&redisDB, "db", 0, "Redis database number")
	flag.IntVar(&numClients, "clients", 10, "Number of concurrent clients")
	flag.IntVar(&numKeys, "keys", 1000, "Number of keys to test")
	flag.StringVar(&keyPrefix, "prefix", "benchmark_", "Key prefix")
	flag.DurationVar(&ttl, "ttl", 60*time.Second, "Key TTL")
	flag.DurationVar(&testDuration, "duration", 10*time.Second, "Test duration")
	flag.Float64Var(&setRatio, "set", 0.5, "Proportion of SET operations")
	flag.Float64Var(&getRatio, "get", 0.4, "Proportion of GET operations")
	flag.Float64Var(&delRatio, "del", 0.1, "Proportion of DEL operations")
}

func main() {
	flag.Parse()

	// Normalize operation ratios
	totalRatio := setRatio + getRatio + delRatio
	setRatio /= totalRatio
	getRatio /= totalRatio
	delRatio /= totalRatio

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       redisDB,
	})
	defer rdb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Verify connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	fmt.Println("Starting Redis benchmark...")

	var wg sync.WaitGroup
	var setCount, getCount, delCount int
	var totalSize int64
	var lock sync.Mutex

	keys := generateKeys(numKeys, keyPrefix)

	// Start client workers
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go clientWorker(ctx, rdb, keys, ttl, setRatio, getRatio, delRatio, &setCount, &getCount, &delCount, &totalSize, &lock, &wg)
	}

	// Run for the specified duration
	time.Sleep(testDuration)
	cancel()

	wg.Wait()

	fmt.Println("Benchmark complete.")
	fmt.Printf("Total clients: %d\n", numClients)
	fmt.Printf("Total keys: %d\n", numKeys)
	fmt.Printf("Total data size: %.2f KB\n", float64(totalSize)/1024)
	fmt.Printf("Total time: %v\n", testDuration)
	fmt.Printf("SET operations: %d\n", setCount)
	fmt.Printf("GET operations: %d\n", getCount)
	fmt.Printf("DEL operations: %d\n", delCount)
	fmt.Printf("Average SET ops/sec: %.2f\n", float64(setCount)/testDuration.Seconds())
	fmt.Printf("Average GET ops/sec: %.2f\n", float64(getCount)/testDuration.Seconds())
	fmt.Printf("Average DEL ops/sec: %.2f\n", float64(delCount)/testDuration.Seconds())
}

func generateKeys(numKeys int, prefix string) []string {
	keys := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return keys
}

func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func clientWorker(
	ctx context.Context,
	rdb *redis.Client,
	keys []string,
	ttl time.Duration,
	setRatio, getRatio, delRatio float64,
	setCount, getCount, delCount *int,
	totalSize *int64,
	lock *sync.Mutex,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	rand.Seed(time.Now().UnixNano())
	for {
		select {
		case <-ctx.Done():
			return
		default:
			op := rand.Float64()
			key := keys[rand.Intn(len(keys))]
			if op < setRatio {
				// SET operation
				value := randomString(100)
				if err := rdb.Set(ctx, key, value, ttl).Err(); err != nil {
					log.Printf("Failed to set key %s: %v", key, err)
				} else {
					lock.Lock()
					*setCount++
					*totalSize += int64(len(value))
					lock.Unlock()
				}
			} else if op < setRatio+getRatio {
				// GET operation
				if _, err := rdb.Get(ctx, key).Result(); err != nil && err != redis.Nil {
					log.Printf("Failed to get key %s: %v", key, err)
				} else {
					lock.Lock()
					*getCount++
					lock.Unlock()
				}
			} else {
				// DEL operation
				if err := rdb.Del(ctx, key).Err(); err != nil {
					log.Printf("Failed to delete key %s: %v", key, err)
				} else {
					lock.Lock()
					*delCount++
					lock.Unlock()
				}
			}
		}
	}
}
