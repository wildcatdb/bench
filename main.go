// Copyright 2025 WildcatDB Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wildcatdb/wildcat/v2"
)

type BenchmarkConfig struct {
	// Database configuration
	DBPath            string
	WriteBufferSize   int64
	SyncOption        string
	LevelCount        int
	BloomFilter       bool
	MaxCompactionConc int

	// Benchmark parameters
	NumOperations int64
	KeySize       int
	ValueSize     int
	NumThreads    int
	BatchSize     int

	// Test types
	Benchmarks []string
	ReadRatio  int // For mixed workloads (0-100)

	// Data distribution
	KeyDistribution string // sequential, random, zipfian
	ExistingKeys    int64  // Number of existing keys for read tests

	// Reporting
	ReportInterval time.Duration
	Histogram      bool
	Stats          bool

	// Advanced options
	UseTransactions  bool
	IteratorTests    bool
	CompressibleData bool
	Seed             int64

	// Cleanup
	CleanupAfter bool
}

type BenchmarkResult struct {
	TestName     string
	Operations   int64
	Duration     time.Duration
	OpsPerSecond float64
	LatencyP50   time.Duration
	LatencyP95   time.Duration
	LatencyP99   time.Duration
	LatencyMax   time.Duration
	BytesRead    int64
	BytesWritten int64
	Errors       int64
}

type LatencyTracker struct {
	mu        sync.Mutex
	latencies []time.Duration
}

func (lt *LatencyTracker) Record(latency time.Duration) {
	lt.mu.Lock()
	lt.latencies = append(lt.latencies, latency)
	lt.mu.Unlock()
}

func (lt *LatencyTracker) GetPercentiles() (p50, p95, p99, max time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.latencies) == 0 {
		return 0, 0, 0, 0
	}

	sort.Slice(lt.latencies, func(i, j int) bool {
		return lt.latencies[i] < lt.latencies[j]
	})

	n := len(lt.latencies)
	p50 = lt.latencies[int(float64(n)*0.50)]
	p95 = lt.latencies[int(float64(n)*0.95)]
	p99 = lt.latencies[int(float64(n)*0.99)]
	max = lt.latencies[n-1]

	return
}

func main() {
	config := parseFlags()
	fmt.Println(`
W)      ww I)iiii L)       D)dddd     C)ccc    A)aa   T)tttttt 
W)      ww   I)   L)       D)   dd   C)   cc  A)  aa     T)    
W)  ww  ww   I)   L)       D)    dd C)       A)    aa    T)    
W)  ww  ww   I)   L)       D)    dd C)       A)aaaaaa    T)    
W)  ww  ww   I)   L)       D)    dd  C)   cc A)    aa    T)    
 W)ww www  I)iiii L)llllll D)ddddd    C)ccc  A)    aa    T)`)

	fmt.Printf("Benchmark Tool\n\n")
	printConfig(config)

	if config.CleanupAfter {
		defer func() {
			if err := os.RemoveAll(config.DBPath); err != nil {
				log.Printf("Failed to cleanup database: %v", err)
			} else {
				fmt.Printf("Cleaned up database directory: %s\n", config.DBPath)
			}
		}()
	}

	results := runBenchmarks(config)

	printResults(results)
}

func parseFlags() *BenchmarkConfig {
	config := &BenchmarkConfig{}

	// Database configuration
	flag.StringVar(&config.DBPath, "db", "/tmp/wildcat_bench", "Database directory path")
	flag.Int64Var(&config.WriteBufferSize, "write_buffer_size", 64*1024*1024, "Write buffer size in bytes")
	flag.StringVar(&config.SyncOption, "sync", "none", "Sync option: none, partial, full")
	flag.IntVar(&config.LevelCount, "levels", 7, "Number of LSM levels")
	flag.BoolVar(&config.BloomFilter, "bloom_filter", true, "Enable bloom filters")
	flag.IntVar(&config.MaxCompactionConc, "max_compaction_concurrency", 4, "Max compaction concurrency")

	// Benchmark parameters
	flag.Int64Var(&config.NumOperations, "num", 10000, "Number of operations")
	flag.IntVar(&config.KeySize, "key_size", 16, "Size of keys in bytes")
	flag.IntVar(&config.ValueSize, "value_size", 100, "Size of values in bytes")
	flag.IntVar(&config.NumThreads, "threads", runtime.NumCPU(), "Number of concurrent threads")
	flag.IntVar(&config.BatchSize, "batch_size", 1, "Batch size for operations")

	// Test types
	benchmarksStr := flag.String("benchmarks", "fillseq,fillprefixed,readseq,readrandom,iterseq,iterrandom,iterprefix,concurrent_writers,high_contention_writes,batch_concurrent_writes", "Comma-separated list of benchmarks")
	flag.IntVar(&config.ReadRatio, "read_ratio", 50, "Read ratio for mixed workloads (0-100)")

	// Data distribution
	flag.StringVar(&config.KeyDistribution, "key_dist", "sequential", "Key distribution: sequential, random, zipfian")
	flag.Int64Var(&config.ExistingKeys, "existing_keys", 0, "Number of existing keys (0 = use num)")

	// Reporting
	flag.DurationVar(&config.ReportInterval, "report_interval", 10*time.Second, "Progress report interval")
	flag.BoolVar(&config.Histogram, "histogram", true, "Show latency histogram")
	flag.BoolVar(&config.Stats, "stats", true, "Show database stats after each benchmark")

	// Advanced options
	flag.BoolVar(&config.UseTransactions, "use_txn", false, "Use manual transactions instead of Update/View")
	flag.BoolVar(&config.IteratorTests, "iterator_tests", false, "Include iterator benchmarks")
	flag.BoolVar(&config.CompressibleData, "compressible", false, "Use compressible test data")
	flag.Int64Var(&config.Seed, "seed", time.Now().UnixNano(), "Random seed")

	// Cleanup
	flag.BoolVar(&config.CleanupAfter, "cleanup", true, "Cleanup database after benchmarks")

	flag.Parse()

	config.Benchmarks = strings.Split(*benchmarksStr, ",")

	if config.ExistingKeys == 0 {
		config.ExistingKeys = config.NumOperations
	}

	return config
}

func printConfig(config *BenchmarkConfig) {
	fmt.Printf("Configuration\n")
	fmt.Printf("=========================\n")
	fmt.Printf("  Database Path: %s\n", config.DBPath)
	fmt.Printf("  Write Buffer Size: %d MB\n", config.WriteBufferSize/(1024*1024))
	fmt.Printf("  Sync Option: %s\n", config.SyncOption)
	fmt.Printf("  Levels: %d\n", config.LevelCount)
	fmt.Printf("  Bloom Filter: %t\n", config.BloomFilter)
	fmt.Printf("  Operations: %d\n", config.NumOperations)
	fmt.Printf("  Key Size: %d bytes\n", config.KeySize)
	fmt.Printf("  Value Size: %d bytes\n", config.ValueSize)
	fmt.Printf("  Threads: %d\n", config.NumThreads)
	fmt.Printf("  Batch Size: %d\n", config.BatchSize)
	fmt.Printf("  Benchmarks: %s\n", strings.Join(config.Benchmarks, ", "))
	fmt.Printf("  Key Distribution: %s\n", config.KeyDistribution)
	fmt.Printf("\n")
}

func runBenchmarks(config *BenchmarkConfig) []*BenchmarkResult {
	var results []*BenchmarkResult

	for _, benchmark := range config.Benchmarks {
		benchmark = strings.TrimSpace(benchmark)
		fmt.Printf("Running benchmark: %s\n", benchmark)

		result := runSingleBenchmark(config, benchmark)
		results = append(results, result)

		if config.Stats {
			printDatabaseStats(config)
		}

		fmt.Printf("Completed %s: %.2f ops/sec\n\n", benchmark, result.OpsPerSecond)
	}

	return results
}

func runSingleBenchmark(config *BenchmarkConfig, benchmarkName string) *BenchmarkResult {
	db := openDatabase(config)
	defer func(db *wildcat.DB) {
		_ = db.Close()
	}(db)

	tracker := &LatencyTracker{}

	var opsCompleted int64
	var bytesRead, bytesWritten int64
	var errors int64

	startTime := time.Now()

	stopReporting := make(chan bool)
	if config.ReportInterval > 0 {
		go func() {
			ticker := time.NewTicker(config.ReportInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					ops := atomic.LoadInt64(&opsCompleted)
					elapsed := time.Since(startTime)
					rate := float64(ops) / elapsed.Seconds()
					fmt.Printf("Progress: %d ops, %.2f ops/sec\n", ops, rate)
				case <-stopReporting:
					return
				}
			}
		}()
	}

	switch benchmarkName {
	case "fillseq":
		runFillSequential(db, config, tracker, &opsCompleted, &bytesWritten, &errors)
	case "fillrandom":
		runFillRandom(db, config, tracker, &opsCompleted, &bytesWritten, &errors)
	case "fillprefixed":
		runFillPrefixed(db, config, tracker, &opsCompleted, &bytesWritten, &errors)
	case "readseq":
		runReadSequential(db, config, tracker, &opsCompleted, &bytesRead, &errors)
	case "readrandom":
		runReadRandom(db, config, tracker, &opsCompleted, &bytesRead, &errors)
	case "readmissing":
		runReadMissing(db, config, tracker, &opsCompleted, &bytesRead)
	case "readwhilewriting":
		runReadWhileWriting(db, config, tracker, &opsCompleted, &bytesRead, &bytesWritten, &errors)
	case "mixedworkload":
		runMixedWorkload(db, config, tracker, &opsCompleted, &bytesRead, &bytesWritten, &errors)
	case "iterseq":
		runIteratorSequential(db, config, tracker, &opsCompleted, &bytesRead, &errors)
	case "iterrandom":
		runIteratorRandom(db, config, tracker, &opsCompleted, &bytesRead, &errors)
	case "iterprefix":
		runIteratorPrefix(db, config, tracker, &opsCompleted, &bytesRead, &errors)
	case "concurrent_writers":
		runConcurrentWriters(db, config, tracker, &opsCompleted, &bytesWritten, &errors)
	case "concurrent_transactions":
		runConcurrentTransactions(db, config, tracker, &opsCompleted, &bytesWritten, &errors)
	case "high_contention_writes":
		runHighContentionWrites(db, config, tracker, &opsCompleted, &bytesWritten, &errors)
	case "batch_concurrent_writes":
		runBatchConcurrentWrites(db, config, tracker, &opsCompleted, &bytesWritten, &errors)
	case "transaction_conflicts":
		runTransactionConflicts(db, config, tracker, &opsCompleted, &bytesWritten, &errors)
	case "concurrent_read_write":
		runConcurrentReadWrite(db, config, tracker, &opsCompleted, &bytesRead, &bytesWritten, &errors)
	case "heavy_contention":
		runHeavyContention(db, config, tracker, &opsCompleted, &bytesWritten, &errors)
	default:
		log.Fatalf("Unknown benchmark: %s", benchmarkName)
	}

	stopReporting <- true

	duration := time.Since(startTime)
	p50, p95, p99, mx := tracker.GetPercentiles()

	return &BenchmarkResult{
		TestName:     benchmarkName,
		Operations:   atomic.LoadInt64(&opsCompleted),
		Duration:     duration,
		OpsPerSecond: float64(atomic.LoadInt64(&opsCompleted)) / duration.Seconds(),
		LatencyP50:   p50,
		LatencyP95:   p95,
		LatencyP99:   p99,
		LatencyMax:   mx,
		BytesRead:    atomic.LoadInt64(&bytesRead),
		BytesWritten: atomic.LoadInt64(&bytesWritten),
		Errors:       atomic.LoadInt64(&errors),
	}
}

func openDatabase(config *BenchmarkConfig) *wildcat.DB {
	var syncOpt wildcat.SyncOption
	switch strings.ToLower(config.SyncOption) {
	case "none":
		syncOpt = wildcat.SyncNone
	case "partial":
		syncOpt = wildcat.SyncPartial
	case "full":
		syncOpt = wildcat.SyncFull
	default:
		log.Fatalf("Invalid sync option: %s", config.SyncOption)
	}

	opts := &wildcat.Options{
		Directory:                config.DBPath,
		WriteBufferSize:          config.WriteBufferSize,
		SyncOption:               syncOpt,
		LevelCount:               config.LevelCount,
		BloomFilter:              config.BloomFilter,
		MaxCompactionConcurrency: config.MaxCompactionConc,
		STDOutLogging:            false,
	}

	db, err := wildcat.Open(opts)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	return db
}

func generateKey(i int64, keySize int, distribution string) []byte {
	var key []byte

	switch distribution {
	case "sequential":
		key = []byte(fmt.Sprintf("%016d", i))
	case "random":
		key = make([]byte, 8)
		for j := 0; j < 8; j++ {
			key[j] = byte((i >> (j * 8)) & 0xFF)
		}
	case "zipfian":
		zipf := i % (i/10 + 1)
		key = []byte(fmt.Sprintf("%016d", zipf))
	default:
		key = []byte(fmt.Sprintf("%016d", i))
	}

	if len(key) < keySize {
		padding := make([]byte, keySize-len(key))
		if _, err := rand.Read(padding); err != nil {
			for i := range padding {
				padding[i] = byte(i % 256)
			}
		}
		key = append(key, padding...)
	} else if len(key) > keySize {
		key = key[:keySize]
	}

	return key
}

func generateKeyWithPrefix(i int64, keySize int, prefix string, distribution string) []byte {
	prefixBytes := []byte(prefix)

	var suffix []byte
	switch distribution {
	case "sequential":
		suffix = []byte(fmt.Sprintf("%016d", i))
	case "random":
		suffix = make([]byte, 8)
		for j := 0; j < 8; j++ {
			suffix[j] = byte((i >> (j * 8)) & 0xFF)
		}
	case "zipfian":
		zipf := i % (i/10 + 1)
		suffix = []byte(fmt.Sprintf("%016d", zipf))
	default:
		suffix = []byte(fmt.Sprintf("%016d", i))
	}

	key := append(prefixBytes, suffix...)

	if len(key) < keySize {
		padding := make([]byte, keySize-len(key))
		if _, err := rand.Read(padding); err != nil {
			for i := range padding {
				padding[i] = byte(i % 256)
			}
		}
		key = append(key, padding...)
	} else if len(key) > keySize {
		key = key[:keySize]
	}

	return key
}

func generateValue(valueSize int, compressible bool) []byte {
	value := make([]byte, valueSize)

	if compressible {
		pattern := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
		for i := 0; i < valueSize; i++ {
			value[i] = pattern[i%len(pattern)]
		}
	} else {
		if _, err := rand.Read(value); err != nil {
			for i := range value {
				value[i] = byte(i % 256)
			}
		}
	}

	return value
}

func runFillSequential(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesWritten, errors *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * opsPerThread
			end := start + opsPerThread
			if threadID == config.NumThreads-1 {
				end = config.NumOperations
			}

			for i := start; i < end; i++ {
				key := generateKey(i, config.KeySize, config.KeyDistribution)
				value := generateValue(config.ValueSize, config.CompressibleData)

				startTime := time.Now()

				err := db.Update(func(txn *wildcat.Txn) error {
					return txn.Put(key, value)
				})

				latency := time.Since(startTime)
				tracker.Record(latency)

				if err != nil {
					atomic.AddInt64(errors, 1)
				} else {
					atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runFillPrefixed(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesWritten, errors *int64) {

	prefixes := []string{"user_", "order_", "product_", "session_", "config_"}

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * opsPerThread
			end := start + opsPerThread
			if threadID == config.NumThreads-1 {
				end = config.NumOperations
			}

			for i := start; i < end; i++ {
				prefix := prefixes[i%int64(len(prefixes))]
				key := generateKeyWithPrefix(i, config.KeySize, prefix, config.KeyDistribution)
				value := generateValue(config.ValueSize, config.CompressibleData)

				startTime := time.Now()

				err := db.Update(func(txn *wildcat.Txn) error {
					return txn.Put(key, value)
				})

				latency := time.Since(startTime)
				tracker.Record(latency)

				if err != nil {
					atomic.AddInt64(errors, 1)
				} else {
					atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runFillRandom(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesWritten, errors *int64) {

	indices := make([]int64, config.NumOperations)
	for i := int64(0); i < config.NumOperations; i++ {
		indices[i] = i
	}

	rng := rand.New(rand.NewSource(config.Seed))
	for i := len(indices) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		indices[i], indices[j] = indices[j], indices[i]
	}

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * opsPerThread
			end := start + opsPerThread
			if threadID == config.NumThreads-1 {
				end = config.NumOperations
			}

			for i := start; i < end; i++ {
				keyIndex := indices[i]
				key := generateKey(keyIndex, config.KeySize, config.KeyDistribution)
				value := generateValue(config.ValueSize, config.CompressibleData)

				startTime := time.Now()

				err := db.Update(func(txn *wildcat.Txn) error {
					return txn.Put(key, value)
				})

				latency := time.Since(startTime)
				tracker.Record(latency)

				if err != nil {
					atomic.AddInt64(errors, 1)
				} else {
					atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runReadSequential(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesRead, errors *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * opsPerThread
			end := start + opsPerThread
			if threadID == config.NumThreads-1 {
				end = config.NumOperations
			}

			for i := start; i < end; i++ {
				keyIndex := i % config.ExistingKeys
				key := generateKey(keyIndex, config.KeySize, config.KeyDistribution)

				startTime := time.Now()

				var value []byte
				err := db.View(func(txn *wildcat.Txn) error {
					var err error
					value, err = txn.Get(key)
					return err
				})

				latency := time.Since(startTime)
				tracker.Record(latency)

				if err != nil {
					atomic.AddInt64(errors, 1)
				} else {
					atomic.AddInt64(bytesRead, int64(len(key)+len(value)))
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runReadRandom(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesRead, errors *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * opsPerThread
			end := start + opsPerThread
			if threadID == config.NumThreads-1 {
				end = config.NumOperations
			}

			for i := start; i < end; i++ {
				keyIndex := (i*1103515245 + 12345) % config.ExistingKeys
				key := generateKey(keyIndex, config.KeySize, config.KeyDistribution)

				startTime := time.Now()

				var value []byte
				err := db.View(func(txn *wildcat.Txn) error {
					var err error
					value, err = txn.Get(key)
					return err
				})

				latency := time.Since(startTime)
				tracker.Record(latency)

				if err != nil {
					atomic.AddInt64(errors, 1)
				} else {
					atomic.AddInt64(bytesRead, int64(len(key)+len(value)))
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runReadMissing(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesRead *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * opsPerThread
			end := start + opsPerThread
			if threadID == config.NumThreads-1 {
				end = config.NumOperations
			}

			for i := start; i < end; i++ {
				keyIndex := config.ExistingKeys + i
				key := generateKey(keyIndex, config.KeySize, config.KeyDistribution)

				startTime := time.Now()

				var value []byte
				err := db.View(func(txn *wildcat.Txn) error {
					var err error
					value, err = txn.Get(key)
					return err
				})

				latency := time.Since(startTime)
				tracker.Record(latency)

				if err != nil {
					// This is expected for missing keys
				} else {
					atomic.AddInt64(bytesRead, int64(len(key)+len(value)))
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runReadWhileWriting(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesRead, bytesWritten, errors *int64) {

	var wg sync.WaitGroup

	readThreads := config.NumThreads / 2
	writeThreads := config.NumThreads - readThreads

	opsPerReadThread := config.NumOperations / int64(readThreads) / 2
	opsPerWriteThread := config.NumOperations / int64(writeThreads) / 2

	for t := 0; t < readThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			for i := int64(0); i < opsPerReadThread; i++ {
				keyIndex := (i*1103515245 + 12345) % config.ExistingKeys
				key := generateKey(keyIndex, config.KeySize, "random")

				startTime := time.Now()

				var value []byte
				err := db.View(func(txn *wildcat.Txn) error {
					var err error
					value, err = txn.Get(key)
					return err
				})

				latency := time.Since(startTime)
				tracker.Record(latency)

				if err != nil {
					atomic.AddInt64(errors, 1)
				} else {
					atomic.AddInt64(bytesRead, int64(len(key)+len(value)))
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	for t := 0; t < writeThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			for i := int64(0); i < opsPerWriteThread; i++ {
				keyIndex := (i*1103515245 + 12345) % config.ExistingKeys
				key := generateKey(keyIndex, config.KeySize, "random")
				value := generateValue(config.ValueSize, config.CompressibleData)

				startTime := time.Now()

				err := db.Update(func(txn *wildcat.Txn) error {
					return txn.Put(key, value)
				})

				latency := time.Since(startTime)
				tracker.Record(latency)

				if err != nil {
					atomic.AddInt64(errors, 1)
				} else {
					atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runMixedWorkload(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesRead, bytesWritten, errors *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * opsPerThread
			end := start + opsPerThread
			if threadID == config.NumThreads-1 {
				end = config.NumOperations
			}

			for i := start; i < end; i++ {
				keyIndex := (i*1103515245 + 12345) % config.ExistingKeys
				key := generateKey(keyIndex, config.KeySize, "random")

				isRead := (i*100)%100 < int64(config.ReadRatio)

				startTime := time.Now()

				if isRead {
					var value []byte
					err := db.View(func(txn *wildcat.Txn) error {
						var err error
						value, err = txn.Get(key)
						return err
					})

					latency := time.Since(startTime)
					tracker.Record(latency)

					if err != nil {
						atomic.AddInt64(errors, 1)
					} else {
						atomic.AddInt64(bytesRead, int64(len(key)+len(value)))
					}
				} else {
					value := generateValue(config.ValueSize, config.CompressibleData)
					err := db.Update(func(txn *wildcat.Txn) error {
						return txn.Put(key, value)
					})

					latency := time.Since(startTime)
					tracker.Record(latency)

					if err != nil {
						atomic.AddInt64(errors, 1)
					} else {
						atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
					}
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runIteratorSequential(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesRead, errors *int64) {

	var keysIterated int64

	startTime := time.Now()

	err := db.View(func(txn *wildcat.Txn) error {
		iter, err := txn.NewIterator(true)
		if err != nil {
			return err
		}

		for {
			key, value, _, ok := iter.Next()
			if !ok {
				break
			}

			atomic.AddInt64(&keysIterated, 1)
			atomic.AddInt64(bytesRead, int64(len(key)+len(value)))

			if keysIterated >= config.NumOperations {
				break
			}
		}

		return nil
	})

	latency := time.Since(startTime)
	tracker.Record(latency)

	if err != nil {
		atomic.AddInt64(errors, 1)
	}

	atomic.StoreInt64(opsCompleted, keysIterated)
}

func runIteratorRandom(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesRead, errors *int64) {
	var iterationsCompleted int64
	iterationsToRun := config.NumOperations / 100
	if iterationsToRun == 0 {
		iterationsToRun = 10
	}

	for i := int64(0); i < iterationsToRun; i++ {
		rangeStart := i * 100
		rangeEnd := rangeStart + 100

		startKey := generateKey(rangeStart, config.KeySize, config.KeyDistribution)
		endKey := generateKey(rangeEnd, config.KeySize, config.KeyDistribution)

		startTime := time.Now()

		err := db.View(func(txn *wildcat.Txn) error {
			iter, err := txn.NewRangeIterator(startKey, endKey, true)
			if err != nil {
				return err
			}

			var keysInRange int64
			for {
				key, value, _, ok := iter.Next()
				if !ok {
					break
				}

				keysInRange++
				atomic.AddInt64(bytesRead, int64(len(key)+len(value)))

				if keysInRange >= 100 { // Limit keys per iteration
					break
				}
			}

			return nil
		})

		latency := time.Since(startTime)
		tracker.Record(latency)

		if err != nil {
			atomic.AddInt64(errors, 1)
		}

		atomic.AddInt64(&iterationsCompleted, 1)
	}

	atomic.StoreInt64(opsCompleted, iterationsCompleted)
}

func runIteratorPrefix(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesRead, errors *int64) {

	prefixes := []string{"user_", "order_", "product_", "session_", "config_"}

	var iterationsCompleted int64
	iterationsToRun := config.NumOperations / 50
	if iterationsToRun == 0 {
		iterationsToRun = int64(len(prefixes))
	}

	for i := int64(0); i < iterationsToRun; i++ {
		prefix := prefixes[i%int64(len(prefixes))]

		startTime := time.Now()

		err := db.View(func(txn *wildcat.Txn) error {
			iter, err := txn.NewPrefixIterator([]byte(prefix), true)
			if err != nil {
				return err
			}

			var keysWithPrefix int64
			for {
				key, value, _, ok := iter.Next()
				if !ok {
					break
				}

				keysWithPrefix++
				atomic.AddInt64(bytesRead, int64(len(key)+len(value)))

				if keysWithPrefix >= 200 {
					break
				}
			}

			return nil
		})

		latency := time.Since(startTime)
		tracker.Record(latency)

		if err != nil {
			atomic.AddInt64(errors, 1)
		}

		atomic.AddInt64(&iterationsCompleted, 1)
	}

	atomic.StoreInt64(opsCompleted, iterationsCompleted)
}

func runConcurrentWriters(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesWritten, errors *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * opsPerThread
			end := start + opsPerThread
			if threadID == config.NumThreads-1 {
				end = config.NumOperations
			}

			for i := start; i < end; i++ {
				key := generateKey(i, config.KeySize, config.KeyDistribution)
				value := generateValue(config.ValueSize, config.CompressibleData)

				startTime := time.Now()

				// Each thread manages its own transaction
				txn, err := db.Begin()
				if err != nil {
					atomic.AddInt64(errors, 1)
					atomic.AddInt64(opsCompleted, 1)
					continue
				}

				err = txn.Put(key, value)
				if err != nil {
					_ = txn.Rollback()
					atomic.AddInt64(errors, 1)
				} else {
					err = txn.Commit()
					if err != nil {
						atomic.AddInt64(errors, 1)
					} else {
						atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
					}
				}

				latency := time.Since(startTime)
				tracker.Record(latency)
				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runConcurrentTransactions(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesWritten, errors *int64) {

	var wg sync.WaitGroup
	batchSize := int64(config.BatchSize)
	if batchSize <= 0 {
		batchSize = 10
	}

	numBatches := config.NumOperations / batchSize
	batchesPerThread := numBatches / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * batchesPerThread
			end := start + batchesPerThread
			if threadID == config.NumThreads-1 {
				end = numBatches
			}

			for batch := start; batch < end; batch++ {
				startTime := time.Now()

				txn, err := db.Begin()
				if err != nil {
					atomic.AddInt64(errors, batchSize)
					atomic.AddInt64(opsCompleted, batchSize)
					continue
				}

				var batchBytesWritten int64
				batchErrors := false

				for i := int64(0); i < batchSize; i++ {
					opIndex := batch*batchSize + i
					key := generateKey(opIndex, config.KeySize, config.KeyDistribution)
					value := generateValue(config.ValueSize, config.CompressibleData)

					err = txn.Put(key, value)
					if err != nil {
						batchErrors = true
						break
					}
					batchBytesWritten += int64(len(key) + len(value))
				}

				if batchErrors {
					_ = txn.Rollback()
					atomic.AddInt64(errors, batchSize)
				} else {
					err = txn.Commit()
					if err != nil {
						atomic.AddInt64(errors, batchSize)
					} else {
						atomic.AddInt64(bytesWritten, batchBytesWritten)
					}
				}

				latency := time.Since(startTime)
				tracker.Record(latency)
				atomic.AddInt64(opsCompleted, batchSize)
			}
		}(t)
	}

	wg.Wait()
}

func runHighContentionWrites(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesWritten, errors *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	contentionRange := config.NumOperations / 4 // All threads compete for 25% of key space

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			for i := int64(0); i < opsPerThread; i++ {
				keyIndex := i % contentionRange
				key := generateKey(keyIndex, config.KeySize, "sequential")
				value := generateValue(config.ValueSize, config.CompressibleData)

				startTime := time.Now()

				txn, err := db.Begin()
				if err != nil {
					atomic.AddInt64(errors, 1)
					atomic.AddInt64(opsCompleted, 1)
					continue
				}

				err = txn.Put(key, value)
				if err != nil {
					_ = txn.Rollback()
					atomic.AddInt64(errors, 1)
				} else {
					err = txn.Commit()
					if err != nil {
						atomic.AddInt64(errors, 1)
					} else {
						atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
					}
				}

				latency := time.Since(startTime)
				tracker.Record(latency)
				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runBatchConcurrentWrites(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesWritten, errors *int64) {

	var wg sync.WaitGroup
	batchSize := int64(config.BatchSize)
	if batchSize <= 0 {
		batchSize = 100 // Default larger batch size
	}

	numBatches := config.NumOperations / batchSize
	batchesPerThread := numBatches / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			start := int64(threadID) * batchesPerThread
			end := start + batchesPerThread
			if threadID == config.NumThreads-1 {
				end = numBatches
			}

			for batch := start; batch < end; batch++ {
				startTime := time.Now()

				txn, err := db.Begin()
				if err != nil {
					atomic.AddInt64(errors, batchSize)
					atomic.AddInt64(opsCompleted, batchSize)
					continue
				}

				var batchBytesWritten int64
				batchErrors := false

				for i := int64(0); i < batchSize; i++ {
					opIndex := batch*batchSize + i
					key := generateKey(opIndex, config.KeySize, config.KeyDistribution)
					value := generateValue(config.ValueSize, config.CompressibleData)

					err = txn.Put(key, value)
					if err != nil {
						batchErrors = true
						break
					}
					batchBytesWritten += int64(len(key) + len(value))
				}

				if batchErrors {
					_ = txn.Rollback()
					atomic.AddInt64(errors, batchSize)
				} else {
					err = txn.Commit()
					if err != nil {
						atomic.AddInt64(errors, batchSize)
					} else {
						atomic.AddInt64(bytesWritten, batchBytesWritten)
					}
				}

				latency := time.Since(startTime)
				tracker.Record(latency)
				atomic.AddInt64(opsCompleted, batchSize)
			}
		}(t)
	}

	wg.Wait()
}

func runTransactionConflicts(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesWritten, errors *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	conflictKeySpace := int64(10)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			for i := int64(0); i < opsPerThread; i++ {
				// All threads compete for the same small set of keys
				keyIndex := i % conflictKeySpace
				key := generateKey(keyIndex, config.KeySize, "sequential")
				value := generateValue(config.ValueSize, config.CompressibleData)

				startTime := time.Now()

				txn, err := db.Begin()
				if err != nil {
					atomic.AddInt64(errors, 1)
					atomic.AddInt64(opsCompleted, 1)
					continue
				}

				_, err = txn.Get(key)
				if err != nil && err.Error() != "key not found" {
					_ = txn.Rollback()
					atomic.AddInt64(errors, 1)
					atomic.AddInt64(opsCompleted, 1)
					continue
				}

				err = txn.Put(key, value)
				if err != nil {
					_ = txn.Rollback()
					atomic.AddInt64(errors, 1)
				} else {
					err = txn.Commit()
					if err != nil {
						atomic.AddInt64(errors, 1)
					} else {
						atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
					}
				}

				latency := time.Since(startTime)
				tracker.Record(latency)
				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runConcurrentReadWrite(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesRead, bytesWritten, errors *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			for i := int64(0); i < opsPerThread; i++ {
				keyIndex := (i*1103515245 + 12345) % config.ExistingKeys
				key := generateKey(keyIndex, config.KeySize, config.KeyDistribution)

				// 70% reads, 30% writes for realistic workload..
				isRead := (i*100)%100 < 70

				startTime := time.Now()

				if isRead {
					var value []byte
					err := db.View(func(txn *wildcat.Txn) error {
						var err error
						value, err = txn.Get(key)
						return err
					})

					latency := time.Since(startTime)
					tracker.Record(latency)

					if err != nil {
						atomic.AddInt64(errors, 1)
					} else {
						atomic.AddInt64(bytesRead, int64(len(key)+len(value)))
					}
				} else {
					value := generateValue(config.ValueSize, config.CompressibleData)

					txn, err := db.Begin()
					if err != nil {
						atomic.AddInt64(errors, 1)
						atomic.AddInt64(opsCompleted, 1)
						continue
					}

					err = txn.Put(key, value)
					if err != nil {
						_ = txn.Rollback()
						atomic.AddInt64(errors, 1)
					} else {
						err = txn.Commit()
						if err != nil {
							atomic.AddInt64(errors, 1)
						} else {
							atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
						}
					}

					latency := time.Since(startTime)
					tracker.Record(latency)
				}

				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func runHeavyContention(db *wildcat.DB, config *BenchmarkConfig, tracker *LatencyTracker,
	opsCompleted, bytesWritten, errors *int64) {

	var wg sync.WaitGroup
	opsPerThread := config.NumOperations / int64(config.NumThreads)

	// Only 3 keys for extreme contention
	contentionKeys := int64(3)

	for t := 0; t < config.NumThreads; t++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			for i := int64(0); i < opsPerThread; i++ {
				keyIndex := i % contentionKeys
				key := generateKey(keyIndex, config.KeySize, "sequential")
				value := generateValue(config.ValueSize, config.CompressibleData)

				startTime := time.Now()

				txn, err := db.Begin()
				if err != nil {
					atomic.AddInt64(errors, 1)
					atomic.AddInt64(opsCompleted, 1)
					continue
				}

				// Read-modify-write pattern to increase contention
				oldValue, err := txn.Get(key)
				if err != nil && err.Error() != "key not found" {
					_ = txn.Rollback()
					atomic.AddInt64(errors, 1)
					atomic.AddInt64(opsCompleted, 1)
					continue
				}

				time.Sleep(1 * time.Microsecond)

				if oldValue != nil {
					value = append(oldValue, value...)
				}

				err = txn.Put(key, value)
				if err != nil {
					_ = txn.Rollback()
					atomic.AddInt64(errors, 1)
				} else {
					err = txn.Commit()
					if err != nil {
						atomic.AddInt64(errors, 1)
					} else {
						atomic.AddInt64(bytesWritten, int64(len(key)+len(value)))
					}
				}

				latency := time.Since(startTime)
				tracker.Record(latency)
				atomic.AddInt64(opsCompleted, 1)
			}
		}(t)
	}

	wg.Wait()
}

func printDatabaseStats(config *BenchmarkConfig) {
	db := openDatabase(config)
	defer func(db *wildcat.DB) {
		_ = db.Close()
	}(db)

	stats := db.Stats()
	fmt.Printf("Database Stats:\n%s\n", stats)
}

func printResults(results []*BenchmarkResult) {
	fmt.Printf("\n")
	fmt.Printf("Benchmark Results\n")
	fmt.Printf("=================\n")
	fmt.Printf("%-25s %12s %12s %12s %12s %12s %12s %8s\n",
		"Test", "Ops", "Ops/sec", "P50", "P95", "P99", "Max", "Errors")
	fmt.Printf("%-25s %12s %12s %12s %12s %12s %12s %8s\n",
		"----", "---", "-------", "---", "---", "---", "---", "------")

	for _, result := range results {
		fmt.Printf("%-25s %12d %12.2f %12s %12s %12s %12s %8d\n",
			result.TestName,
			result.Operations,
			result.OpsPerSecond,
			formatDuration(result.LatencyP50),
			formatDuration(result.LatencyP95),
			formatDuration(result.LatencyP99),
			formatDuration(result.LatencyMax),
			result.Errors)
	}

	fmt.Printf("\n")

	var totalOps int64
	var totalDuration time.Duration
	var totalBytesRead, totalBytesWritten int64

	for _, result := range results {
		totalOps += result.Operations
		totalDuration += result.Duration
		totalBytesRead += result.BytesRead
		totalBytesWritten += result.BytesWritten
	}

	fmt.Printf("Summary\n")
	fmt.Printf("=========================\n")
	fmt.Printf("  Total Operations: %d\n", totalOps)
	fmt.Printf("  Total Duration: %s\n", totalDuration)
	fmt.Printf("  Average Ops/sec: %.2f\n", float64(totalOps)/totalDuration.Seconds())
	fmt.Printf("  Total Bytes Read: %s\n", formatBytes(totalBytesRead))
	fmt.Printf("  Total Bytes Written: %s\n", formatBytes(totalBytesWritten))

	if totalBytesRead > 0 {
		fmt.Printf("  Read Throughput: %s/sec\n", formatBytes(int64(float64(totalBytesRead)/totalDuration.Seconds())))
	}
	if totalBytesWritten > 0 {
		fmt.Printf("  Write Throughput: %s/sec\n", formatBytes(int64(float64(totalBytesWritten)/totalDuration.Seconds())))
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1000000.0)
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
