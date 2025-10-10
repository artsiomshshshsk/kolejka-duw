# DUW Queue Monitor

Go service that monitors DUW queue status via rotating proxies, stores data in PostgreSQL, and notifies via Telegram when ticket availability changes.

## Highlights

- 🔄 Proxy rotation (per-request random session)
- ⏱️ Polling every 10s within working hours (default 08:00–18:00, Europe/Warsaw)
- 📅 Weekdays-only fetching (skips Sat–Sun)
- 🗄️ PostgreSQL storage for "odbiór karty" and "Odbiór karty - wieczory"
- 🔔 Telegram notifications on transitions of `tickets_left`
  - `<= 0 → > 0`: tickets appeared
  - `> 0 → <= 0`: tickets finished

## Quick Start

```bash
cp env.example .env
# Fill in proxy creds; optionally DB and Telegram vars
docker-compose up -d
docker-compose logs -f duw-monitor
```

## Configuration (env)

```bash
# Proxy (required)
PROXY_USERNAME=...
PROXY_PASSWORD=...
PROXY_ADDRESS=...
PROXY_PORT=...

# Working hours (optional)
WORK_START_HOUR=8
WORK_END_HOUR=18

# Database (optional defaults)
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=duw_queue

# Telegram (optional)
TELEGRAM_BOT_TOKEN=...
TELEGRAM_CHAT_ID=@your_channel   # add bot as channel admin
```

Proxy format used by the app:
```
http://{username_with_session_id}:{password}@{address}:{port}
```

## Schema (brief)

- Tables: `odbior_karty`, `odbior_karty_wieczory`
- Common fields: `queue_id, name, location, ticket_count, tickets_served, workplaces, average_wait_time, average_service_time, registered_tickets, max_tickets, ticket_value, active, tickets_left, enabled, operations, created_at`
- Views: `recent_odbior_karty`, `recent_odbior_karty_wieczory`

## Useful Commands

```bash
# Start/stop
docker-compose up -d
docker-compose down

# Logs
docker-compose logs -f duw-monitor
docker-compose logs -f postgres
```

## License

MIT License
