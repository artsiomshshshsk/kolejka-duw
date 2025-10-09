# DUW Queue Monitor

A Go application that monitors DUW (Dolnośląski Urząd Wojewódzki) queue status using rotating proxies and stores specific events in PostgreSQL.

## Features

- 🔄 **Proxy Rotation**: Uses different proxy for each request with random session IDs
- 📊 **Queue Monitoring**: Monitors specific queue events every 10 seconds
- ⏰ **Working Hours**: Configurable polling schedule (default: 8 AM - 6 PM Europe/Warsaw)
- 🗄️ **PostgreSQL Storage**: Stores "odbiór karty" and "Odbiór karty - wieczory" events
- 🐳 **Docker Support**: Complete Docker Compose setup with PostgreSQL and pgAdmin
- 📈 **Database Views**: Pre-built views for easy data analysis

## Quick Start

1. **Clone and setup**:
   ```bash
   git clone <your-repo>
   cd duw-queue-monitor
   ```

2. **Configure environment**:
   ```bash
   cp env.example .env
   # Edit .env with your proxy credentials
   ```

3. **Start the services**:
   ```bash
   docker-compose up -d
   ```

4. **Access pgAdmin** (optional):
   - URL: http://localhost:8080
   - Email: admin@duw.local
   - Password: admin123

## Configuration

### Required Environment Variables

```bash
# Proxy configuration (REQUIRED)
PROXY_USERNAME=your_proxy_username
PROXY_PASSWORD=your_proxy_password
PROXY_ADDRESS=your_proxy_address
PROXY_PORT=your_proxy_port

# Working hours configuration (optional - defaults to 8 AM - 6 PM Europe/Warsaw)
WORK_START_HOUR=8
WORK_END_HOUR=18

# Database configuration (optional - defaults provided)
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=duw_queue
```

### Proxy Format

The application uses the proxy format you specified:
```
http://{username_with_session_id}:{password}@{address}:{port}
```

Where `session_id` is a random number (0-999999) generated for each request.

## Database Schema

### Tables

1. **`odbior_karty`** - Stores "odbiór karty" events
2. **`odbior_karty_wieczory`** - Stores "Odbiór karty - wieczory" events

### Fields (both tables)
- `queue_id` - Original queue ID from API
- `name` - Queue name
- `location` - Location (Wrocław, Jelenia Góra, etc.)
- `ticket_count` - Current ticket count
- `tickets_served` - Tickets served
- `workplaces` - Number of workplaces
- `average_wait_time` - Average wait time
- `average_service_time` - Average service time
- `registered_tickets` - Registered tickets
- `max_tickets` - Maximum tickets
- `ticket_value` - Ticket value
- `active` - Whether queue is active
- `tickets_left` - Tickets remaining
- `enabled` - Whether queue is enabled
- `operations` - JSON array of operations
- `created_at` - Timestamp

### Views

- `recent_odbior_karty` - Recent "odbiór karty" events (last hour)
- `recent_odbior_karty_wieczory` - Recent "Odbiór karty - wieczory" events (last hour)

## Usage

### Starting the Monitor

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f duw-monitor

# Stop services
docker-compose down
```

### Database Queries

```sql
-- Get recent "odbiór karty" events
SELECT * FROM recent_odbior_karty;

-- Get events by location
SELECT * FROM odbior_karty WHERE location = 'Wrocław' ORDER BY created_at DESC;

-- Count events per hour
SELECT 
    DATE_TRUNC('hour', created_at) as hour,
    COUNT(*) as event_count
FROM odbior_karty 
GROUP BY hour 
ORDER BY hour DESC;
```

## Development

### Local Development

```bash
# Install dependencies
go mod download

# Run locally (requires PostgreSQL)
go run main.go
```

### Building

```bash
# Build Docker image
docker build -t duw-monitor .

# Run with docker-compose
docker-compose up -d
```

## Monitoring

The application logs:
- Proxy usage (with masked credentials)
- Successful data fetches
- Database save operations
- Errors and warnings

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   DUW Monitor   │───▶│  Rotating Proxy │───▶│   DUW API       │
│   (Go App)      │    │  (Random URLs)  │    │   (External)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │
         ▼
┌─────────────────┐
│   PostgreSQL    │
│   Database      │
└─────────────────┘
```

## Troubleshooting

### Common Issues

1. **Proxy connection errors**: Check proxy credentials in `.env`
2. **Database connection**: Ensure PostgreSQL container is running
3. **No data**: Check if proxy is working and API is accessible

### Logs

```bash
# View application logs
docker-compose logs duw-monitor

# View database logs
docker-compose logs postgres
```

## License

MIT License
