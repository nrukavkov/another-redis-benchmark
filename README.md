# Another Redis Benchmark

Another Redis Benchmark is a Go-based application designed for performance testing of Redis instances. It simulates real-world client workloads to measure operations latency and throughput for `SET`, `GET`, and `DEL` commands.

---

## Features
- Simulates realistic Redis workloads with multiple concurrent clients.
- Measures latency for `SET`, `GET`, and `DEL` operations.
- Supports configurable test parameters (e.g., number of clients, key count, test duration).
- Provides detailed statistics, including:
  - Total operations.
  - Average operations per second.
  - Minimum, average, and maximum latency for each operation.
- Cross-platform support (Linux, MacOS, Windows, ARM64, and AMD64).

---

## Getting Started

### Prerequisites
- **Redis**: Ensure you have a Redis instance running and accessible.
- **Go**: Version 1.23 or higher.

---

## Installation

### Build from Source
1. Clone the repository:
   ```bash
   git clone https://github.com/<your-repo>/another-redis-benchmark.git
   cd another-redis-benchmark
   ```
2. Build the binary:
   ```bash
   go build -o another-redis-benchmark
   ```

### Prebuilt Binaries
You can download prebuilt binaries for your platform from the [Releases](https://github.com/<your-repo>/another-redis-benchmark/releases) page.

---

## Usage

Run the application with customizable parameters.

### Basic Command
```bash
./another-redis-benchmark [options]
```

---

## Command-Line Options

| Option              | Default Value  | Description                                                                          |
|---------------------|----------------|--------------------------------------------------------------------------------------|
| `-addr`             | `localhost:6379` | Redis server address.                                                               |
| `-pass`             | `""`           | Redis server password.                                                              |
| `-db`               | `0`            | Redis database index.                                                               |
| `-clients`          | `10`           | Number of concurrent clients to simulate.                                           |
| `-keys`             | `1000`         | Number of unique keys to test.                                                      |
| `-prefix`           | `benchmark_`   | Prefix for generated keys.                                                          |
| `-ttl`              | `60s`          | Time-to-live for `SET` operations.                                                  |
| `-duration`         | `10s`          | Test duration.                                                                       |
| `-set`              | `0.5`          | Proportion of `SET` operations (relative to the total workload).                    |
| `-get`              | `0.4`          | Proportion of `GET` operations (relative to the total workload).                    |
| `-del`              | `0.1`          | Proportion of `DEL` operations (relative to the total workload).                    |

---

### Examples

#### 1. Run with Default Parameters
```bash
./another-redis-benchmark
```

#### 2. Customize Redis Address and Password
```bash
./another-redis-benchmark -addr "redis.example.com:6379" -pass "my_redis_password"
```

#### 3. Increase Workload and Test Duration
```bash
./another-redis-benchmark -clients 50 -keys 10000 -duration 30s
```

---

## Output

### Real-Time Progress
```
Client 1: SET=123, GET=456, DEL=78
Client 2: SET=234, GET=567, DEL=89
...
Total: SET=357, GET=1023, DEL=167
```

### Final Summary
```
Benchmark complete.
Total clients: 10
Total keys: 1000
Total time: 10s
SET operations: 5000
GET operations: 4000
DEL operations: 1000
Average SET ops/sec: 500.0
Average GET ops/sec: 400.0
Average DEL ops/sec: 100.0
SET Latency (ms): Min=0.50, Avg=1.23, Max=10.45
GET Latency (ms): Min=0.45, Avg=1.10, Max=8.97
DEL Latency (ms): Min=0.60, Avg=1.45, Max=12.34
```

---

## Advanced Configuration

### Key Distribution
- Use the `-keys` parameter to define the number of unique keys.
- Higher values spread operations across more keys, reducing key collisions.

### Custom Workload
- Adjust `-set`, `-get`, and `-del` ratios to simulate specific workloads.
- Example: `-set 0.7 -get 0.2 -del 0.1` focuses on `SET` operations.

---

## Cross-Platform Support
Download the appropriate binary for your platform:
- **Linux**: `another-redis-benchmark-linux-amd64.zip`
- **MacOS**: `another-redis-benchmark-darwin-arm64.zip`
- **Windows**: `another-redis-benchmark-windows-amd64.zip`

Extract the ZIP file and run the executable.

---

## Development

### Running Tests
To test the application:
```bash
go test ./...
```

### Contributing
Feel free to open issues or submit pull requests. Follow the [CONTRIBUTING.md](https://github.com/<your-repo>/another-redis-benchmark/blob/main/CONTRIBUTING.md) for guidelines.

---

## License
This project is licensed under the MIT License. See [LICENSE](https://github.com/<your-repo>/another-redis-benchmark/blob/main/LICENSE) for details.

---

## Known Issues
1. **High Latency on Large Key Sets**:
   - Ensure Redis server is properly configured for high throughput.
2. **Limited Resources on Local Machines**:
   - Consider testing on a dedicated Redis server for accurate benchmarks.
