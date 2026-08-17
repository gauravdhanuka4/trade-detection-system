# Trade Detection System

## 🎯 Goal
Build a high-performance, real-time financial trade fraud detection system that:
- Ingests financial trades via Redis Streams with guaranteed delivery
- Detects fraud/AML risks using intelligent context-aware rule engine
- Maintains persistent user trading contexts with smart caching (80/20 strategy)
- Generates alerts stored in PostgreSQL with sophisticated behavioral analysis
- Provides future API with JWT/RBAC for viewing flagged trades and alerts

---

## 🏗️ Architecture Overview

### **Tiered Data Storage Strategy**
```
Hot Data (Redis, 1-24h TTL)    → Real-time metrics, recent trades
Warm Data (Redis, 1-7d TTL)   → Behavioral profiles, risk scores  
Cold Data (PostgreSQL)        → All trades, alerts, historical stats
```

### **Data Flow**
```
Trade Data → Redis Stream → Worker → Persistent Context (Redis) → Rules Engine → Alerts → PostgreSQL
                                  ↓
                            Background Profile Updates
```

### **80/20 Performance Strategy**
- **20% of traders** generate **80% of volume** → Priority caching, persistent contexts
- **80% of traders** are infrequent → Cache can expire, rebuild on-demand
- Smart cache warming for high-volume traders

---

## 📂 Project Structure
```
cmd/
 ├── api/              # (Phase 2) API service with JWT/RBAC
 └── worker/           # Worker entrypoint (trade stream processor)
config/
 └── config.go         # Viper-based config with .env support
internal/
 ├── db/
 │   ├── db.go         # PostgreSQL connection management
 │   └── repositories/ # Repository pattern with fraud detection
 ├── models/           # Domain models (Trade, User, Alert, Rule, Config)
 ├── redis/            # Redis Streams client and caching operations
 ├── processor/        # Trade processor with persistent context
 ├── rules/            # Rules engine (wash trade, velocity, patterns)
 ├── queue/            # Redis stream consumer management
 └── utils/            # Logging, error handling
scripts/
 ├── init_schema.sql   # Database schema with fraud detection tables
 └── setup_local_env.sh
```

---

## 🗄️ Database Schema

### **Core Tables**
- **users**: id, username, email, password_hash, role, is_active, timestamps
- **trades**: id, user_id, symbol, amount, price, trade_type, timestamp, external_user_id
- **alerts**: id, trade_id, rule_id, risk_type, severity, status, description, timestamps
- **rules**: id, name, type, parameters, is_active, timestamps

### **Fraud Detection Features**
- **Sophisticated user profiling** with `GetUserTradeStats()` 
- **Behavioral pattern analysis** with `GetUserTradingPattern()`
- **Time-window analysis** with `GetByUserInTimeWindow()`
- **Anonymous user tracking** via external_user_id

---

## ⚙️ Core Components

### **1. Configuration Management** ✅
- **Environment-based**: Pure .env configuration (no YAML)
- **Validation**: Comprehensive config validation with sensible defaults
- **Encapsulation**: Private fields with getter methods
- **Sections**: Server, Database, Redis, JWT, Auth, Queue, App

### **2. Database Layer** ✅
- **Repository Pattern**: Clean separation with interfaces
- **Advanced Queries**: Fraud detection with user statistics and patterns
- **Connection Pooling**: pgxpool for high performance
- **SOLID Principles**: Interface-based design for testability

### **3. Redis Streams** 🔄
- **Guaranteed Delivery**: Redis Streams with consumer groups
- **Message Acknowledgment**: Ensures no trade loss
- **Persistent Context**: User trading contexts persist and update incrementally
- **Smart Caching**: 80/20 strategy for optimal performance

### **4. Trade Processing** 🔄
- **Stream Consumer**: Processes trades from Redis Stream
- **Context Building**: Incremental updates to persistent user contexts
- **Rules Engine**: Context-aware fraud detection
- **Alert Generation**: Sophisticated risk scoring and pattern detection

### **5. Rules Engine** 📋
- **Wash Trade Detection**: Advanced pattern matching
- **Velocity Checks**: Unusual trading frequency detection
- **Behavioral Analysis**: Deviation from user's typical patterns
- **Risk Scoring**: Dynamic risk assessment based on context

---

## 🚀 Current Implementation Status

### **✅ Completed (Phase 1A)**
- [x] **Database repositories** with sophisticated fraud detection queries
- [x] **Configuration management** with .env support and validation
- [x] **Models and enums** for all domain entities
- [x] **Project structure** following clean architecture principles

### **🔄 In Progress (Phase 1B)**
- [ ] **Redis Streams client** with consumer group management
- [ ] **Persistent context management** (incremental updates, not rebuilds)
- [ ] **Trade processor** with context-aware analysis
- [ ] **Rules engine** implementation
- [ ] **Alert handling** and persistence

### **📋 Next Steps (Phase 1C)**
- [ ] **Background workers** for profile updates
- [ ] **Cache warming** for frequent traders
- [ ] **Performance optimization** and monitoring

---

## 🎯 Design Decisions

### **Persistent Context Approach**
Instead of rebuilding context for each trade:
1. **Load existing context** from Redis (if exists)
2. **Update incrementally** with new trade data
3. **Persist updated context** back to Redis
4. **Smart expiration** - frequent traders stay cached, infrequent expire naturally

### **80/20 Caching Strategy**
- **High-volume traders** (20%): Persistent contexts, priority cache warming
- **Low-volume traders** (80%): Cache expires after 1 day, rebuild on-demand
- **Performance focus** on traders who generate most volume

### **Stream-Based Processing**
- **Redis Streams** over simple pub/sub for guaranteed delivery
- **Consumer groups** for horizontal scaling
- **Message acknowledgment** prevents trade loss
- **Replay capability** for error recovery

---

## 🛠️ Tech Stack
- **Go** - High-performance backend
- **PostgreSQL** - ACID compliance for financial data
- **Redis Streams** - Reliable message processing
- **Viper** - Configuration management
- **pgxpool** - High-performance PostgreSQL driver

---

## 📈 Future Roadmap

### **Phase 2: API Layer** (Later)
- JWT authentication with refresh tokens
- Role-based access control (ADMIN, ANALYST, VIEWER)
- REST API for trades, alerts, and user management
- Protected routes with middleware

### **Phase 3: Real-time Features** (Later)
- Real-time alert notifications
- WebSocket connections for live updates
- Dashboard for monitoring trade flows
- Alert management interface

### **Phase 4: Advanced Analytics** (Future)
- Machine learning integration
- Advanced pattern recognition
- Predictive risk modeling
- Performance analytics and reporting

### **Phase 5: Production Deployment** (Future)
- Docker containerization
- Kubernetes deployment
- Monitoring and observability
- Load testing and optimization

---

## 🔧 Development Setup

### **Prerequisites**
- Go 1.21+
- PostgreSQL 15+
- Redis 7+

### **Environment Configuration**
```bash
# Copy and customize environment file
cp .env.example .env

# Key configurations:
DATABASE_HOST=localhost
DATABASE_DATABASE=trade_detection
REDIS_HOST=localhost
JWT_SECRET=your-secret-key-here
QUEUE_PROVIDER=redis
```

### **Database Setup**
```bash
# Initialize database schema
psql -d trade_detection -f scripts/init_schema.sql
```

### **Running the System**
```bash
# Start the trade processor worker
go run cmd/worker/main.go

# (Future) Start the API server
go run cmd/api/main.go
```

---

## 📊 Performance Characteristics

### **Throughput Targets**
- **Trade Processing**: 10,000+ trades/second
- **Alert Generation**: Sub-100ms latency
- **Context Lookup**: Sub-10ms from Redis
- **Database Writes**: Batched for efficiency

### **Scalability Features**
- **Horizontal scaling** via Redis consumer groups
- **Cache-first architecture** reduces database load
- **Persistent contexts** eliminate redundant computations
- **Background processing** for non-critical updates

---

## 🔒 Security Considerations

### **Data Protection**
- **Encrypted connections** to PostgreSQL and Redis
- **Sensitive data masking** in logs
- **Input validation** and sanitization
- **SQL injection prevention** via parameterized queries

### **Access Control** (Phase 2)
- **JWT-based authentication** with secure secrets
- **Role-based permissions** (ADMIN, ANALYST, VIEWER)
- **API rate limiting** and request validation
- **Audit logging** for compliance

---

This architecture provides a **production-ready foundation** for high-performance financial fraud detection with intelligent caching, persistent contexts, and sophisticated behavioral analysis.
