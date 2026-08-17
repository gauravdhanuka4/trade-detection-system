-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ===== ENUM TYPES =====

-- User roles
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('ADMIN', 'ANALYST', 'VIEWER');
    END IF;
END$$;

-- Trade type
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'trade_type') THEN
        CREATE TYPE trade_type AS ENUM ('BUY', 'SELL');
    END IF;
END$$;

-- Alert risk type
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'risk_type') THEN
        CREATE TYPE risk_type AS ENUM ('FRAUD', 'AML');
    END IF;
END$$;

-- Rule types
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'rule_type') THEN
        CREATE TYPE rule_type AS ENUM ('WASH_TRADE', 'VELOCITY', 'ANOMALY', 'LAYERING', 'PUMP_DUMP');
    END IF;
END$$;

-- Alert severity
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'severity_type') THEN
        CREATE TYPE severity_type AS ENUM ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL');
    END IF;
END$$;

-- Alert status
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'alert_status') THEN
        CREATE TYPE alert_status AS ENUM ('OPEN', 'INVESTIGATING', 'RESOLVED', 'FALSE_POSITIVE');
    END IF;
END$$;

-- ===== TABLES =====

-- USERS table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username TEXT UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    role user_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- RULES table
CREATE TABLE IF NOT EXISTS rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    type rule_type NOT NULL,
    description TEXT,
    conditions JSONB,
    threshold NUMERIC(18,4),
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- TRADES table
CREATE TABLE IF NOT EXISTS trades (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    symbol TEXT NOT NULL,
    amount NUMERIC(18,4) NOT NULL,
    price NUMERIC(18,4) NOT NULL,
    trade_type trade_type NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    source TEXT,
    raw_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ALERTS table
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trade_id UUID NOT NULL REFERENCES trades(id) ON DELETE CASCADE,
    rule_id UUID REFERENCES rules(id) ON DELETE SET NULL,
    risk_type risk_type NOT NULL,
    severity severity_type NOT NULL,
    status alert_status NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Trading user statistics table (persistent cache baseline)
CREATE TABLE IF NOT EXISTS trading_user_stats (
    user_id VARCHAR(255) PRIMARY KEY,
    total_trades BIGINT NOT NULL DEFAULT 0,
    total_amount NUMERIC(18,4) NOT NULL DEFAULT 0,
    avg_trade_size NUMERIC(18,4) NOT NULL DEFAULT 0,
    first_trade_date TIMESTAMPTZ,
    last_trade_date TIMESTAMPTZ,
    trading_days INTEGER NOT NULL DEFAULT 0,
    risk_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    typical_symbols JSONB,
    trading_hours JSONB,
    last_sync_timestamp TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01'::TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


ALTER TABLE trading_user_stats 
ADD COLUMN IF NOT EXISTS last_sync_timestamp TIMESTAMPTZ DEFAULT '1970-01-01';


ALTER TABLE alerts 
ADD COLUMN decision_criteria JSONB;

-- Create GIN index for efficient JSONB queries
CREATE INDEX idx_alerts_decision_criteria 
ON alerts USING gin(decision_criteria);

-- Add comment for documentation
COMMENT ON COLUMN alerts.decision_criteria IS 'Structured JSON containing rule evaluation details, thresholds, detected values, and triggered conditions';


-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_trading_user_stats_risk_score ON trading_user_stats(risk_score);
CREATE INDEX IF NOT EXISTS idx_trading_user_stats_last_trade ON trading_user_stats(last_trade_date);
CREATE INDEX IF NOT EXISTS idx_trading_user_stats_updated ON trading_user_stats(updated_at);

-- GIN index for JSONB searches
CREATE INDEX IF NOT EXISTS idx_trading_user_stats_symbols_gin ON trading_user_stats USING GIN(typical_symbols);

-- Optimize trade queries for fraud detection
CREATE INDEX IF NOT EXISTS idx_trades_user_timestamp ON trades(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trades_timestamp_amount ON trades(timestamp DESC, amount);
CREATE INDEX IF NOT EXISTS idx_trades_user_symbol_timestamp ON trades(user_id, symbol, timestamp DESC);
