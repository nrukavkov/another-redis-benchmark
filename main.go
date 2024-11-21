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
	var totalSet, totalGet, totalDel int
	var lock sync.Mutex

	keys := generateKeys(numKeys, keyPrefix)

	// Statistics per user
	progress := make([]map[string]int, numClients)
	for i := range progress {
		progress[i] = map[string]int{"set": 0, "get": 0, "del": 0}
	}

	// Signal channel to stop clients
	stop := make(chan struct{})

	// Start client workers
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go clientWorker(ctx, rdb, keys, ttl, setRatio, getRatio, delRatio, progress[i], &totalSet, &totalGet, &totalDel, &lock, stop, &wg)
	}

	// Start statistics reporter
	go reportProgress(progress, &totalSet, &totalGet, &totalDel, stop)

	// Run for the specified duration
	time.Sleep(testDuration)

	// Signal workers to stop
	close(stop)

	// Wait for all workers to finish
	wg.Wait()

	fmt.Println("\nBenchmark complete.")
	fmt.Printf("Total clients: %d\n", numClients)
	fmt.Printf("Total keys: %d\n", numKeys)
	fmt.Printf("Total time: %v\n", testDuration)
	fmt.Printf("SET operations: %d\n", totalSet)
	fmt.Printf("GET operations: %d\n", totalGet)
	fmt.Printf("DEL operations: %d\n", totalDel)
	fmt.Printf("Average SET ops/sec: %.2f\n", float64(totalSet)/testDuration.Seconds())
	fmt.Printf("Average GET ops/sec: %.2f\n", float64(totalGet)/testDuration.Seconds())
	fmt.Printf("Average DEL ops/sec: %.2f\n", float64(totalDel)/testDuration.Seconds())
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
	progress map[string]int,
	totalSet, totalGet, totalDel *int,
	lock *sync.Mutex,
	stop <-chan struct{},
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	rand.Seed(time.Now().UnixNano())
	for {
		select {
		case <-stop:
			return
		default:
			op := rand.Float64()
			key := keys[rand.Intn(len(keys))]
			if op < setRatio {
				// SET operation
				value := randomString(100)
				if err := rdb.Set(ctx, key, value, ttl).Err(); err == nil {
					lock.Lock()
					progress["set"]++
					*totalSet++
					lock.Unlock()
				}
			} else if op < setRatio+getRatio {
				// GET operation
				if _, err := rdb.Get(ctx, key).Result(); err == nil || err == redis.Nil {
					lock.Lock()
					progress["get"]++
					*totalGet++
					lock.Unlock()
				}
			} else {
				// DEL operation
				if err := rdb.Del(ctx, key).Err(); err == nil {
					lock.Lock()
					progress["del"]++
					*totalDel++
					lock.Unlock()
				}
			}
		}
	}
}

func reportProgress(progress []map[string]int, totalSet, totalGet, totalDel *int, stop <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	numClients := len(progress)

	// Напечатать статический список клиентов и общую строку один раз
	for i := 0; i < numClients; i++ {
		fmt.Printf("Client %d: SET=0, GET=0, DEL=0\n", i+1)
	}
	fmt.Println("Total: SET=0, GET=0, DEL=0")

	for {
		select {
		case <-stop:
			fmt.Print("\033[0m") // Сброс форматирования при завершении
			return
		case <-ticker.C:
			// Переместить курсор вверх на количество строк клиентов + строку Total
			fmt.Printf("\033[%dA", numClients+1)

			// Обновить прогресс для каждого клиента
			for i, p := range progress {
				fmt.Printf("\033[KClient %d: SET=%d, GET=%d, DEL=%d\n", i+1, p["set"], p["get"], p["del"])
			}

			// Обновить общую строку
			fmt.Printf("\033[KTotal: SET=%d, GET=%d, DEL=%d\n", *totalSet, *totalGet, *totalDel)
		}
	}
}
