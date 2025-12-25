# Redis Stream Debugging Guide

## Quick Reference for Inspecting Trade Streams

### 1. Connect to Redis CLI

```bash
# Connect to local Redis
redis-cli

# Connect to Redis with password
redis-cli -a your_password

# Connect to remote Redis
redis-cli -h hostname -p 6379 -a password
```

### 2. Check Stream Information

```bash
# Check if the stream exists and get info
XINFO STREAM trades:stream

# Get stream length (number of messages)
XLEN trades:stream

# Get range of stream IDs
XRANGE trades:stream - +
```

### 3. Read Latest Trades from Stream

```bash
# Read last 10 messages (most recent)
XREVRANGE trades:stream + - COUNT 10

# Read first 10 messages (oldest)
XRANGE trades:stream - + COUNT 10

# Read all messages (be careful with large streams!)
XRANGE trades:stream - +
```

### 4. Read Specific Trade Fields

```bash
# Read last message with all fields
XREVRANGE trades:stream + - COUNT 1

# Example output:
# 1) 1) "1735058400000-0"
#    2) 1) "trade_id"
#       2) "123e4567-e89b-12d3-a456-426614174000"
#       3) "user_id"
#       4) "HFT_001"
#       5) "symbol"
#       6) "AAPL"
#       7) "amount"
#       8) "75000"
#       9) "price"
#      10) "175.50"
#      11) "trade_type"
#      12) "BUY"
#      13) "timestamp"
#      14) "1735058400"
#      15) "trade_data"
#      16) "{\"id\":\"...\",\"user_id\":\"HFT_001\",...}"
```

### 5. Monitor Stream in Real-Time

```bash
# Watch new messages as they arrive (blocking read)
XREAD BLOCK 0 STREAMS trades:stream $

# Read new messages with timeout (5 seconds)
XREAD BLOCK 5000 STREAMS trades:stream $

# Use redis-cli in monitoring mode
redis-cli --csv XREAD BLOCK 0 STREAMS trades:stream $
```

### 6. Consumer Group Commands

```bash
# List all consumer groups for the stream
XINFO GROUPS trades:stream

# Check pending messages in a consumer group
XPENDING trades:stream trade-processors

# Get consumer group info
XINFO CONSUMERS trades:stream trade-processors
```

### 7. Useful Filtering and Analysis

```bash
# Count messages by searching for pattern
# (This requires scanning, use with caution)
XLEN trades:stream

# Delete old messages (trim to keep last 1000)
XTRIM trades:stream MAXLEN 1000

# Delete the entire stream
DEL trades:stream
```

### 8. Pretty Print Last Trade (Using redis-cli + jq)

```bash
# Install jq if not installed
# macOS: brew install jq
# Ubuntu: apt-get install jq

# Get last trade and pretty print the JSON data
redis-cli XREVRANGE trades:stream + - COUNT 1 | grep -o '{.*}' | jq '.'
```

### 9. Monitoring Script

Create a monitoring script `monitor-stream.sh`:

```bash
#!/bin/bash

echo "=== Monitoring trades:stream ==="
echo "Stream Length: $(redis-cli XLEN trades:stream)"
echo ""
echo "Last 5 trades:"
redis-cli XREVRANGE trades:stream + - COUNT 5

echo ""
echo "Watching for new trades (Ctrl+C to stop)..."
redis-cli XREAD BLOCK 0 STREAMS trades:stream $
```

Make it executable:
```bash
chmod +x monitor-stream.sh
./monitor-stream.sh
```

### 10. Real-Time Dashboard (One-Liner)

```bash
# Show stream stats every 2 seconds
watch -n 2 'echo "Length: $(redis-cli XLEN trades:stream)" && \
redis-cli XREVRANGE trades:stream + - COUNT 1'
```

## Common Use Cases

### See the Full Trade JSON

```bash
redis-cli XREVRANGE trades:stream + - COUNT 1 | \
  grep -A 1 "trade_data" | \
  tail -n 1 | \
  jq '.'
```

### Count Trades by Type

```bash
# Get all trades and count by type (use carefully with large streams)
redis-cli XRANGE trades:stream - + | \
  grep "trade_type" -A 1 | \
  grep -v "trade_type" | \
  sort | uniq -c
```

### Monitor Generation Rate

```bash
# Check how many trades per second are being generated
watch -n 1 'redis-cli XLEN trades:stream'
```

### Find Fraud Patterns

Look for trades from fraud users:

```bash
# Search for wash trade patterns
redis-cli XRANGE trades:stream - + | grep "FRAUD_WASH"

# Search for specific symbols
redis-cli XRANGE trades:stream - + | grep "PENNY_A"
```

## Example Session

```bash
$ redis-cli

# Check stream status
127.0.0.1:6379> XINFO STREAM trades:stream
 1) "length"
 2) (integer) 1500
 3) "radix-tree-keys"
 4) (integer) 1
 5) "radix-tree-nodes"
 6) (integer) 2
 7) "groups"
 8) (integer) 0
 9) "last-generated-id"
10) "1735058400000-0"

# Read last trade
127.0.0.1:6379> XREVRANGE trades:stream + - COUNT 1
1) 1) "1735058400000-0"
   2)  1) "trade_id"
       2) "abc123..."
       3) "user_id"
       4) "HFT_001"
       5) "symbol"
       6) "AAPL"
       7) "amount"
       8) "75234.56"
       9) "price"
      10) "175.89"
      11) "trade_type"
      12) "BUY"
      13) "timestamp"
      14) "1735058400"
      15) "trade_data"
      16) "{\"id\":\"abc123...\",\"user_id\":\"HFT_001\",\"symbol\":\"AAPL\",...}"

# Watch for new trades
127.0.0.1:6379> XREAD BLOCK 0 STREAMS trades:stream $
(waiting for new messages...)
```

## Troubleshooting

### Stream doesn't exist
```bash
127.0.0.1:6379> XLEN trades:stream
(integer) 0  # Stream is empty or doesn't exist yet
```
**Solution**: Start the feed generator first.

### Permission denied
```bash
127.0.0.1:6379> AUTH your_password_here
```

### Too many messages
```bash
# Trim to keep only last 10000 messages
XTRIM trades:stream MAXLEN 10000

# Or use approximate trimming (more efficient)
XTRIM trades:stream MAXLEN ~ 10000
```

## Redis Desktop Tools

For a GUI experience, consider:

1. **RedisInsight** (Official, Free)
   - Download: https://redis.com/redis-enterprise/redis-insight/
   - Best for stream visualization

2. **Another Redis Desktop Manager**
   - GitHub: https://github.com/qishibo/AnotherRedisDesktopManager
   - Open source, cross-platform

3. **redis-commander** (Web UI)
   ```bash
   npm install -g redis-commander
   redis-commander
   # Open http://localhost:8081
   ```

## Performance Tips

- Use `COUNT` to limit results
- Use `~` with XTRIM for approximate trimming (faster)
- Don't scan entire large streams frequently
- Use consumer groups for processing instead of XRANGE
- Monitor memory usage: `INFO MEMORY`
