package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
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

type operationStats struct {
	minTime   float64
	maxTime   float64
	totalTime float64
	count     int
	mu        sync.Mutex
}

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

	// Statistics
	setStats := operationStats{minTime: math.MaxFloat64}
	getStats := operationStats{minTime: math.MaxFloat64}
	delStats := operationStats{minTime: math.MaxFloat64}

	keys := generateKeys(numKeys, keyPrefix)

	// Progress tracking
	progress := make([]map[string]int, numClients)
	for i := range progress {
		progress[i] = map[string]int{"set": 0, "get": 0, "del": 0}
	}

	// Signal channel to stop clients
	stop := make(chan struct{})

	// Start client workers
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go clientWorker(ctx, rdb, keys, ttl, setRatio, getRatio, delRatio, progress[i],
			&totalSet, &totalGet, &totalDel, &lock, stop, &setStats, &getStats, &delStats, &wg)
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

	// Print latency statistics
	printStats("SET", &setStats)
	printStats("GET", &getStats)
	printStats("DEL", &delStats)
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
	setStats, getStats, delStats *operationStats,
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
				start := time.Now()
				if err := rdb.Set(ctx, key, value, ttl).Err(); err == nil {
					duration := time.Since(start).Seconds() * 1000
					updateStats(setStats, duration)
					lock.Lock()
					progress["set"]++
					*totalSet++
					lock.Unlock()
				}
			} else if op < setRatio+getRatio {
				// GET operation
				start := time.Now()
				if _, err := rdb.Get(ctx, key).Result(); err == nil || err == redis.Nil {
					duration := time.Since(start).Seconds() * 1000
					updateStats(getStats, duration)
					lock.Lock()
					progress["get"]++
					*totalGet++
					lock.Unlock()
				}
			} else {
				// DEL operation
				start := time.Now()
				if err := rdb.Del(ctx, key).Err(); err == nil {
					duration := time.Since(start).Seconds() * 1000
					updateStats(delStats, duration)
					lock.Lock()
					progress["del"]++
					*totalDel++
					lock.Unlock()
				}
			}
		}
	}
}

func updateStats(stats *operationStats, duration float64) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.count++
	stats.totalTime += duration
	if duration < stats.minTime {
		stats.minTime = duration
	}
	if duration > stats.maxTime {
		stats.maxTime = duration
	}
}

func printStats(operation string, stats *operationStats) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	avgTime := 0.0
	if stats.count > 0 {
		avgTime = stats.totalTime / float64(stats.count)
	}

	fmt.Printf("%s Latency (ms): Min=%.2f, Avg=%.2f, Max=%.2f\n",
		operation, stats.minTime, avgTime, stats.maxTime)
}

func reportProgress(progress []map[string]int, totalSet, totalGet, totalDel *int, stop <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	numClients := len(progress)

	// Print initial rows
	for i := 0; i < numClients; i++ {
		fmt.Printf("Client %d: SET=0, GET=0, DEL=0\n", i+1)
	}
	fmt.Println("Total: SET=0, GET=0, DEL=0")

	for {
		select {
		case <-stop:
			fmt.Print("\033[0m") // Reset formatting
			return
		case <-ticker.C:
			// Move cursor up
			fmt.Printf("\033[%dA", numClients+1)

			// Print updated rows
			for i, p := range progress {
				fmt.Printf("\033[KClient %d: SET=%d, GET=%d, DEL=%d\n", i+1, p["set"], p["get"], p["del"])
			}

			// Print updated total
			fmt.Printf("\033[KTotal: SET=%d, GET=%d, DEL=%d\n", *totalSet, *totalGet, *totalDel)
		}
	}
}
