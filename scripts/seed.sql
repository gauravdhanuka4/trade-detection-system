-- Seed data for rules table
-- Clean, DB-driven configuration with no environment variable references

-- WASH_TRADE - Active Rule
INSERT INTO rules (id, name, type, description, conditions, enabled, created_at, updated_at)
VALUES (
  '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
  'Wash Trade Detection',
  'WASH_TRADE',
  'Detects circular trading patterns where users buy and sell the same security within a short time window to artificially inflate volume or manipulate prices.',
  '{
    "time_window_seconds": 300,
    "min_transactions": 2,
    "price_variance_threshold": 0.02,
    "amount_similarity_threshold": 0.1
  }'::jsonb,
  true,
  NOW(),
  NOW()
);

-- VELOCITY - Active Rule
INSERT INTO rules (id, name, type, description, conditions, enabled, created_at, updated_at)
VALUES (
  '6ba7b811-9dad-11d1-80b4-00c04fd430c9',
  'High Velocity Trading Detection',
  'VELOCITY',
  'Detects abnormally high trading frequency that may indicate bot activity, algorithmic manipulation, or coordinated pump schemes.',
  '{
    "time_window_seconds": 60,
    "max_trades_per_window": 10,
    "spike_multiplier": 3.0
  }'::jsonb,
  true,
  NOW(),
  NOW()
);

-- ANOMALY - Active Rule
INSERT INTO rules (id, name, type, description, conditions, enabled, created_at, updated_at)
VALUES (
  '6ba7b812-9dad-11d1-80b4-00c04fd430ca',
  'Trade Size Anomaly Detection',
  'ANOMALY',
  'Detects unusual trade sizes or price deviations compared to user historical patterns using statistical analysis.',
  '{
    "lookback_days": 7,
    "size_deviation_threshold": 3.0,
    "price_deviation_threshold": 0.15
  }'::jsonb,
  true,
  NOW(),
  NOW()
);

-- LAYERING - Inactive Rule (requires order book data)
INSERT INTO rules (id, name, type, description, conditions, enabled, created_at, updated_at)
VALUES (
  '6ba7b813-9dad-11d1-80b4-00c04fd430cb',
  'Order Layering Detection',
  'LAYERING',
  'NOT_YET_IMPLEMENTED - Detects order spoofing and layering manipulation. Requires order book data including pending orders, cancellations, and order modifications.',
  '{}'::jsonb,
  false,
  NOW(),
  NOW()
);

-- PUMP_DUMP - Inactive Rule (requires cross-user analysis)
INSERT INTO rules (id, name, type, description, conditions, enabled, created_at, updated_at)
VALUES (
  '6ba7b814-9dad-11d1-80b4-00c04fd430cc',
  'Pump and Dump Detection',
  'PUMP_DUMP',
  'NOT_YET_IMPLEMENTED - Detects coordinated pump and dump schemes. Requires cross-user correlation engine and batch analysis capabilities.',
  '{}'::jsonb,
  false,
  NOW(),
  NOW()
);
