-- Initialize database schema
-- This file is executed when PostgreSQL container starts for the first time

-- Create database if it doesn't exist
-- (This is handled by POSTGRES_DB environment variable)

-- Create tables for queue monitoring
CREATE TABLE IF NOT EXISTS odbior_karty (
    id SERIAL PRIMARY KEY,
    queue_id INTEGER,
    name VARCHAR(255),
    location VARCHAR(100),
    ticket_count INTEGER,
    tickets_served INTEGER,
    workplaces INTEGER,
    average_wait_time INTEGER,
    average_service_time INTEGER,
    registered_tickets INTEGER,
    max_tickets INTEGER,
    ticket_value VARCHAR(255),
    active BOOLEAN,
    tickets_left INTEGER,
    enabled BOOLEAN,
    operations JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS odbior_karty_wieczory (
    id SERIAL PRIMARY KEY,
    queue_id INTEGER,
    name VARCHAR(255),
    location VARCHAR(100),
    ticket_count INTEGER,
    tickets_served INTEGER,
    workplaces INTEGER,
    average_wait_time INTEGER,
    average_service_time INTEGER,
    registered_tickets INTEGER,
    max_tickets INTEGER,
    ticket_value VARCHAR(255),
    active BOOLEAN,
    tickets_left INTEGER,
    enabled BOOLEAN,
    operations JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_odbior_karty_created_at ON odbior_karty(created_at);
CREATE INDEX IF NOT EXISTS idx_odbior_karty_wieczory_created_at ON odbior_karty_wieczory(created_at);
CREATE INDEX IF NOT EXISTS idx_odbior_karty_location ON odbior_karty(location);
CREATE INDEX IF NOT EXISTS idx_odbior_karty_wieczory_location ON odbior_karty_wieczory(location);
CREATE INDEX IF NOT EXISTS idx_odbior_karty_queue_id ON odbior_karty(queue_id);
CREATE INDEX IF NOT EXISTS idx_odbior_karty_wieczory_queue_id ON odbior_karty_wieczory(queue_id);

-- Create a view for easy querying of recent data
CREATE OR REPLACE VIEW recent_odbior_karty AS
SELECT 
    *,
    EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - created_at)) as seconds_ago
FROM odbior_karty 
WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '1 hour'
ORDER BY created_at DESC;

CREATE OR REPLACE VIEW recent_odbior_karty_wieczory AS
SELECT 
    *,
    EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - created_at)) as seconds_ago
FROM odbior_karty_wieczory 
WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '1 hour'
ORDER BY created_at DESC;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;
