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
./wildcat_bench

# Run specific benchmarks
./wildcat_bench -benchmarks="fillseq,readseq,readrandom"

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

## Example Output
```bash

W)      ww I)iiii L)       D)dddd     C)ccc    A)aa   T)tttttt
W)      ww   I)   L)       D)   dd   C)   cc  A)  aa     T)
W)  ww  ww   I)   L)       D)    dd C)       A)    aa    T)
W)  ww  ww   I)   L)       D)    dd C)       A)aaaaaa    T)
W)  ww  ww   I)   L)       D)    dd  C)   cc A)    aa    T)
W)ww www  I)iiii L)llllll D)ddddd    C)ccc  A)    aa    T)
Benchmark Tool

Configuration
=========================
Database Path: /tmp/wildcat_bench
Write Buffer Size: 64 MB
Sync Option: none
Levels: 7
Bloom Filter: true
Operations: 10000
Key Size: 16 bytes
Value Size: 100 bytes
Threads: 16
Batch Size: 1
Benchmarks: fillseq, fillprefixed, readseq, readrandom, iterseq, iterrandom, iterprefix, concurrent_writers, high_contention_writes, batch_concurrent_writes
Key Distribution: sequential

Running benchmark: fillseq
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160000             │
│ Active Memtable Entries    : 10000               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10000               │
└──────────────────────────────────────────────────┘
Completed fillseq: 195080.78 ops/sec

Running benchmark: fillprefixed
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160580             │
│ Active Memtable Entries    : 10005               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10005               │
└──────────────────────────────────────────────────┘
Completed fillprefixed: 232962.31 ops/sec

Running benchmark: readseq
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160580             │
│ Active Memtable Entries    : 10005               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10005               │
└──────────────────────────────────────────────────┘
Completed readseq: 1857008.14 ops/sec

Running benchmark: readrandom
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160580             │
│ Active Memtable Entries    : 10005               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10005               │
└──────────────────────────────────────────────────┘
Completed readrandom: 4235972.47 ops/sec

Running benchmark: iterseq
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160580             │
│ Active Memtable Entries    : 10005               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10005               │
└──────────────────────────────────────────────────┘
Completed iterseq: 9533024.78 ops/sec

Running benchmark: iterrandom
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160580             │
│ Active Memtable Entries    : 10005               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10005               │
└──────────────────────────────────────────────────┘
Completed iterrandom: 67455.31 ops/sec

Running benchmark: iterprefix
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160580             │
│ Active Memtable Entries    : 10005               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10005               │
└──────────────────────────────────────────────────┘
Completed iterprefix: 1197568.94 ops/sec

Running benchmark: concurrent_writers
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160580             │
│ Active Memtable Entries    : 10005               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10005               │
└──────────────────────────────────────────────────┘
Completed concurrent_writers: 233548.24 ops/sec

Running benchmark: high_contention_writes
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160580             │
│ Active Memtable Entries    : 10005               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10005               │
└──────────────────────────────────────────────────┘
Completed high_contention_writes: 234198.95 ops/sec

Running benchmark: batch_concurrent_writes
Database Stats:
┌──────────────────────────────────────────────────┐
│ Wildcat DB Stats and Configuration               │
├──────────────────────────────────────────────────┤
│ Write Buffer Size          : 67108864            │
│ Sync Option                : 0                   │
│ Level Count                : 7                   │
│ Bloom Filter Enabled       : false               │
│ Max Compaction Concurrency : 4                   │
│ Compaction Cooldown        : 5s                  │
│ Compaction Batch Size      : 8                   │
│ Compaction Size Ratio      : 1.1                 │
│ Compaction Threshold       : 8                   │
│ Score Size Weight          : 0.8                 │
│ Score Count Weight         : 0.2                 │
│ Flusher Interval           : 1ms                 │
│ Compactor Interval         : 250ms               │
│ Bloom FPR                  : 0.01                │
│ WAL Retry                  : 10                  │
│ WAL Backoff                : 128µs               │
│ SSTable B-Tree Order       : 10                  │
│ LRU Size                   : 1024                │
│ LRU Evict Ratio            : 0.2                 │
│ LRU Access Weight          : 0.8                 │
│ File Version               : 2                   │
│ Magic Number               : 1464421444          │
│ Directory                  : /tmp/wildcat_bench/ │
├──────────────────────────────────────────────────┤
│ ID Generator State                               │
├──────────────────────────────────────────────────┤
│ Last SST ID                : 0                   │
│ Last WAL ID                : 1                   │
├──────────────────────────────────────────────────┤
│ Runtime Statistics                               │
├──────────────────────────────────────────────────┤
│ Active Memtable Size       : 1160580             │
│ Active Memtable Entries    : 10005               │
│ Active Transactions        : 0                   │
│ Oldest Read Timestamp      : 0                   │
│ WAL Files                  : 0                   │
│ Total SSTables             : 0                   │
│ Total Entries              : 10005               │
└──────────────────────────────────────────────────┘
Completed batch_concurrent_writes: 240140.76 ops/sec


Benchmark Results
=================
Test                               Ops      Ops/sec          P50          P95          P99          Max   Errors
----                               ---      -------          ---          ---          ---          ---   ------
fillseq                          10000    195080.78       43.3μs      232.8μs      585.6μs        2.7ms        0
fillprefixed                     10000    232962.31       44.3μs      195.9μs      340.3μs      790.5μs        0
readseq                          10000   1857008.14        1.6μs        7.0μs       63.9μs      270.0μs        0
readrandom                       10000   4235972.47        1.8μs        3.6μs        5.6μs       23.6μs        0
iterseq                          10000   9533024.78        1.0ms        1.0ms        1.0ms        1.0ms        0
iterrandom                         100     67455.31       13.3μs       21.9μs       35.5μs       35.5μs        0
iterprefix                         200   1197568.94        656ns        1.0μs        5.5μs        6.5μs        0
concurrent_writers               10000    233548.24       41.8μs      192.8μs      347.6μs      904.1μs        0
high_contention_writes           10000    234198.95       39.6μs      201.6μs      350.3μs        1.2ms        0
batch_concurrent_writes          10000    240140.76       43.7μs      188.9μs      310.3μs      868.2μs        0

Summary
=========================
Total Operations: 80300
Total Duration: 231.789097ms
Average Ops/sec: 346435.62
Total Bytes Read: 4.4 MB
Total Bytes Written: 5.5 MB
Read Throughput: 19.2 MB/sec
Write Throughput: 23.9 MB/sec
Cleaned up database directory: /tmp/wildcat_bench

Process finished with the exit code 0
```