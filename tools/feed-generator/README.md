# Trade Feed Generator

A sophisticated trade feed generator that produces realistic trading patterns with configurable fraud injection for testing the trade detection system.

## Features

- **Realistic Trader Profiles**: Simulates three types of traders with different behaviors
  - High-Frequency Traders (20% of users, 80% of volume)
  - Regular Traders (70% of users, 18% of volume)
  - Casual Traders (10% of users, 2% of volume)

- **Fraud Pattern Injection**: Configurable fraud patterns for testing detection algorithms
  - Wash Trades: Buy/sell pairs with minimal price difference
  - Velocity Spikes: Sudden bursts of trading activity
  - Anomalies: Unusual patterns (size, time, symbol, price)

- **Configurable Parameters**: Full control over generation behavior
  - Trades per second (TPS)
  - Generation duration
  - Fraud injection rate
  - Specific fraud types

- **Real-time Statistics**: Monitor generation progress
  - Total trades generated
  - Fraud patterns injected
  - Current throughput
  - Volume generated
  - Profile distribution

## Installation

### Prerequisites

- Go 1.22 or higher
- Redis (for trade stream)
- Access to main trade-detection-system

### Build from Source

```bash
cd tools/feed-generator
go build -o feed-generator ./cmd
```

### Using Docker

```bash
cd tools/feed-generator
docker build -t feed-generator .
docker run --rm feed-generator generate --tps 100 --duration 5m
```

## Usage

### Basic Commands

```bash
# Show help
./feed-generator --help

# Show generate command help
./feed-generator generate --help

# Show version
./feed-generator version
```

### Generate Trade Feed

```bash
# Generate 100 trades/sec for 5 minutes (default)
./feed-generator generate

# Generate 50 trades/sec for 10 minutes
./feed-generator generate --tps 50 --duration 10m

# Generate indefinitely (until Ctrl+C)
./feed-generator generate --tps 100 --duration 0

# Generate with 10% fraud rate
./feed-generator generate --tps 100 --fraud-rate 0.1

# Generate only wash trade patterns
./feed-generator generate --tps 50 --fraud-type WASH

# Verbose mode (print each trade)
./feed-generator generate --tps 10 --verbose
```

### Configuration

#### Using Config File

Create `.feed-generator.yaml` in current directory or home directory:

```yaml
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

generate:
  tps: 100
  duration: 5m
  fraud_rate: 0.05
  fraud_type: ALL
  verbose: false
  stats_interval: 10s

profiles:
  hft_ratio: 0.20
  regular_ratio: 0.70
  casual_ratio: 0.10
```

#### Using Environment Variables

```bash
export FEED_GEN_REDIS_HOST=production-redis.example.com
export FEED_GEN_REDIS_PORT=6379
export FEED_GEN_GENERATE_TPS=200
./feed-generator generate
```

#### Using CLI Flags

```bash
./feed-generator generate \
  --redis-host localhost \
  --redis-port 6379 \
  --tps 200 \
  --duration 10m \
  --fraud-rate 0.1 \
  --fraud-type VELOCITY
```

#### Configuration Priority

1. CLI Flags (highest priority)
2. Environment Variables
3. Config File
4. Defaults (lowest priority)

## Examples

### Load Testing

Generate high-volume trades for load testing:

```bash
./feed-generator generate --tps 1000 --duration 1h
```

### Fraud Detection Testing

Generate trades with high fraud rate:

```bash
./feed-generator generate --tps 100 --fraud-rate 0.3 --duration 30m
```

### Specific Pattern Testing

Test wash trade detection:

```bash
./feed-generator generate --tps 50 --fraud-type WASH --fraud-rate 0.5
```

Test velocity spike detection:

```bash
./feed-generator generate --tps 50 --fraud-type VELOCITY --fraud-rate 0.2
```

### Development & Debugging

Run with verbose output:

```bash
./feed-generator generate --tps 10 --verbose --duration 1m
```

## Output

### Statistics Display

```
🚀 Starting Trade Feed Generator...
Configuration:
  Redis: localhost:6379
  Stream: trades:stream
  Throughput: 100 trades/sec
  Duration: 5m0s
  Fraud Rate: 5.0%

✅ Connected to Redis at localhost:6379

[00:10] 1000 trades | 50 fraud | 100.0 tps | $0.5M volume
[00:20] 2000 trades | 100 fraud | 100.0 tps | $1.0M volume
[00:30] 3000 trades | 150 fraud | 100.0 tps | $1.5M volume

=== Final Statistics ===
Duration:       5m0s
Total Trades:   30000
Fraud Patterns: 1500 (5.0%)
Throughput:     100.0 trades/sec
Total Volume:   $15.2M

By Profile Type:
  HFT: 6000 (20.0%)
  REGULAR: 21000 (70.0%)
  CASUAL: 3000 (10.0%)

Generation complete! ✅
```

## Trader Profiles

### High-Frequency Trader (HFT)

- **Volume**: 80% of total trading volume
- **Users**: 20% of users
- **Trades/Hour**: 80-150
- **Average Size**: $50k-$100k
- **Symbols**: Blue chip stocks (AAPL, MSFT, GOOGL, etc.)
- **Active Hours**: Market hours (9 AM - 4 PM)
- **Volatility**: Low (0.2-0.3)

### Regular Trader

- **Volume**: 18% of total trading volume
- **Users**: 70% of users
- **Trades/Hour**: 1-3
- **Average Size**: $4k-$8k
- **Symbols**: Popular stocks and ETFs
- **Active Hours**: Selected hours during the day
- **Volatility**: Medium (0.4-0.6)

### Casual Trader

- **Volume**: 2% of total trading volume
- **Users**: 10% of users
- **Trades/Hour**: <1
- **Average Size**: $1k-$2k
- **Symbols**: ETFs
- **Active Hours**: Occasional
- **Volatility**: Low (0.3)

## Fraud Patterns

### Wash Trade

Generates matching buy/sell pairs:
- Same symbol
- Same amount
- Minimal price difference (<0.1%)
- Short time gap (1-4 seconds)

### Velocity Spike

Creates sudden burst of trades:
- 10-20 trades in rapid succession
- Same symbol
- Small price variations
- Triggers velocity rules

### Anomaly

Generates unusual patterns:
- **Size Anomaly**: 10x normal trade size
- **Time Anomaly**: Trading at unusual hours (2-5 AM)
- **Symbol Anomaly**: Penny stocks from regular traders
- **Price Anomaly**: ±25% deviation from market price

## Architecture

```
feed-generator/
├── cmd/                    # CLI commands
│   ├── main.go            # Entry point
│   ├── root.go            # Root command (Cobra)
│   └── generate.go        # Generate command
├── internal/
│   ├── config/            # Configuration management
│   │   └── config.go      # Viper integration
│   ├── generator/         # Core generation engine
│   │   └── generator.go   # Trade generation logic
│   ├── profiles/          # Trader profiles
│   │   └── profiles.go    # Profile definitions
│   └── patterns/          # Fraud patterns
│       └── patterns.go    # Pattern injection
└── configs/
    └── default.yaml       # Default configuration
```

## Development

### Run Tests

```bash
go test ./...
```

### Build for Multiple Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o feed-generator-linux ./cmd

# macOS
GOOS=darwin GOARCH=amd64 go build -o feed-generator-mac ./cmd

# Windows
GOOS=windows GOARCH=amd64 go build -o feed-generator.exe ./cmd
```

### Hot Reload during Development

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

## Troubleshooting

### Cannot Connect to Redis

```bash
# Check Redis is running
redis-cli ping

# Check Redis host/port
./feed-generator generate --redis-host localhost --redis-port 6379
```

### Low Throughput

- Reduce TPS if system is overloaded
- Check Redis performance
- Monitor system resources

### Fraud Patterns Not Detected

- Verify fraud rate is sufficient
- Check detection rules are configured
- Ensure worker is processing trades

## Performance

- **Throughput**: Up to 10,000 trades/sec
- **Memory**: ~50MB base + ~1KB per active trader profile
- **CPU**: Scales linearly with TPS
- **Network**: ~1KB per trade (Redis stream)

## License

Part of the Trade Detection System
Copyright (c) 2024

## Support

For issues or questions:
- GitHub Issues: https://github.com/gauravdhanuka4/trade-detection-system/issues
- Documentation: See main README.md
