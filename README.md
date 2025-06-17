# Wildcat Bench

A research tool used to benchmark the Wildcat storage engine.

[Wildcat](https://github.com/wildcatdb/wildcat)

## Features
- Sequential/random reads and writes, iterators, concurrent operations
- Adjust operations count, key/value sizes, thread count, and more
- Latency percentiles (P50, P95, P99), throughput, error rates
- Monitor benchmark progress with configurable intervals
- View detailed database stats after each benchmark
- Iterator full, range, and prefix iteration benchmarks

## Quick Start

```bash
# Build the benchmark tool
go build -o wildcat_bench main.go

# Run default benchmarks (recommended first run)
./bench

# Run specific benchmarks
./bench -benchmarks="fillseq,readrandom"

# Run with custom parameters
./wildcat_bench -num=50000 -threads=8 -key_size=32 -value_size=1024
```

## Benchmark Types

### **Fill Operations**
- **`fillseq`** - Sequential key insertion for baseline write performance
- **`fillprefixed`** - Insert keys with common prefixes (user_, order_, product_, etc.)
- **`fillrandom`** - Random key insertion testing hash-based access patterns

### **Read Operations**
- **`readseq`** - Sequential key reads for optimal cache behavior testing
- **`readrandom`** - Random key reads simulating real-world access patterns
- **`readmissing`** - Read non-existent keys to test bloom filter effectiveness

### **Iterator Operations**
- **`iterseq`** - Full database iteration testing sequential scan performance
- **`iterrandom`** - Range iteration with random key ranges
- **`iterprefix`** - Prefix-based iteration testing targeted queries

### **Concurrent Operations**
- **`concurrent_writers`** - Multiple threads writing independently
- **`high_contention_writes`** - Threads competing for overlapping key ranges
- **`batch_concurrent_writes`** - Batched operations with concurrent execution
- **`concurrent_transactions`** - Manual transaction management under load
- **`transaction_conflicts`** - Intentional conflict scenarios testing MVCC
- **`concurrent_read_write`** - Mixed read/write workload (70/30 split)
- **`heavy_contention`** - Extreme contention on very few keys

### **Mixed Workloads**
- **`readwhilewriting`** - Concurrent reads and writes
- **`mixedworkload`** - Configurable read/write ratio

## Configuration Options

### Database Configuration
```bash
-db="/tmp/wildcat_bench"              # Database directory path
-write_buffer_size=67108864           # Write buffer size (64MB default)
-sync="none"                          # Sync option: none, partial, full
-levels=7                             # Number of LSM levels
-bloom_filter=true                    # Enable bloom filters
-max_compaction_concurrency=4         # Max concurrent compactions
```

### Benchmark Parameters
```bash
-num=10000                           # Number of operations per benchmark
-key_size=16                         # Key size in bytes
-value_size=100                      # Value size in bytes
-threads=16                          # Number of concurrent threads (uses all by default)
-batch_size=1                        # Operations per batch/transaction
```

### Workload Configuration
```bash
-benchmarks="fillseq,readseq"        # Comma-separated benchmark list
-read_ratio=50                       # Read percentage for mixed workloads (0-100)
-key_dist="sequential"               # Key distribution: sequential, random, zipfian
-existing_keys=0                     # Number of existing keys (0 = use num)
```

### Advanced Options
```bash
-report_interval=10s                 # Progress reporting interval
-histogram=true                      # Show latency histograms
-stats=true                          # Show database stats after each benchmark
-use_txn=false                       # Use manual transactions vs Update/View
-compressible=false                  # Generate compressible test data
-seed=1234567890                     # Random seed for reproducible results
-cleanup=true                        # Cleanup database after completion
```
