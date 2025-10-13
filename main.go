package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// Template represents proxy configuration
type Template struct {
	Username string
	Password string
	Address  string
	Port     string
}

// GetRandomProxyUrl generates a random proxy URL with session ID
func (p *Template) GetRandomProxyUrl() string {
	sessionId := strconv.Itoa(int(rand.Int31n(1000000)))
	return fmt.Sprintf("http://%s:%s@%s:%s", fmt.Sprintf(p.Username, sessionId), p.Password, p.Address, p.Port)
}

// GetRandomProxyUrl creates a proxy URL from environment variables
func GetRandomProxyUrl() string {
	template := Template{
		Username: os.Getenv("PROXY_USERNAME"),
		Password: os.Getenv("PROXY_PASSWORD"),
		Address:  os.Getenv("PROXY_ADDRESS"),
		Port:     os.Getenv("PROXY_PORT"),
	}
	return template.GetRandomProxyUrl()
}

// QueueData represents the API response structure
type QueueData struct {
	Result map[string][]QueueItem `json:"result"`
}

type QueueItem struct {
	ID                 int         `json:"id"`
	Name               string      `json:"name"`
	Operations         []Operation `json:"operations"`
	TicketCount        int         `json:"ticket_count"`
	TicketsServed      int         `json:"tickets_served"`
	Workplaces         int         `json:"workplaces"`
	AverageWaitTime    *int        `json:"average_wait_time"`
	AverageServiceTime *int        `json:"average_service_time"`
	RegisteredTickets  int         `json:"registered_tickets"`
	MaxTickets         *int        `json:"max_tickets"`
	TicketValue        string      `json:"ticket_value"`
	Active             bool        `json:"active"`
	Location           string      `json:"location"`
	TicketsLeft        int         `json:"tickets_left"`
	Enabled            bool        `json:"enabled"`
}

type Operation struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// Database connection
var db *sql.DB

func main() {
	err := sendTelegramMessage("Starting duw queue monitoring 🫢")
	if err != nil {
		log.Fatal(err)
	}

	// Initialize database connection
	initDB()
	defer db.Close()

	// Start the monitoring loop
	startMonitoring()
}

func initDB() {
	// Database connection string
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "duw_queue")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to PostgreSQL database")

	// Create tables if they don't exist
	createTables()
}

func createTables() {
	// Table for "odbiór karty" events
	createOdbiorkartyTable := `
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
	);`

	// Table for "Odbiór karty - wieczory" events
	createOdbiorkartyWieczoryTable := `
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
	);`

	// Create indexes for better performance
	createIndexes := `
	CREATE INDEX IF NOT EXISTS idx_odbior_karty_created_at ON odbior_karty(created_at);
	CREATE INDEX IF NOT EXISTS idx_odbior_karty_wieczory_created_at ON odbior_karty_wieczory(created_at);
	CREATE INDEX IF NOT EXISTS idx_odbior_karty_location ON odbior_karty(location);
	CREATE INDEX IF NOT EXISTS idx_odbior_karty_wieczory_location ON odbior_karty_wieczory(location);
	`

	if _, err := db.Exec(createOdbiorkartyTable); err != nil {
		log.Fatal("Failed to create odbior_karty table:", err)
	}

	if _, err := db.Exec(createOdbiorkartyWieczoryTable); err != nil {
		log.Fatal("Failed to create odbior_karty_wieczory table:", err)
	}

	if _, err := db.Exec(createIndexes); err != nil {
		log.Fatal("Failed to create indexes:", err)
	}

	log.Println("Database tables created successfully")
}

// isWithinWorkingHours checks if current time is within working hours in Europe/Warsaw timezone
func isWithinWorkingHours(startHour, endHour int) bool {
	// Load Warsaw timezone
	warsawTZ, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		log.Printf("Error loading Warsaw timezone: %v", err)
		return false
	}

	// Get current time in Warsaw timezone
	now := time.Now().In(warsawTZ)
	currentHour := now.Hour()

	// Check if current hour is within working hours
	return currentHour >= startHour && currentHour < endHour
}

// getEnvInt gets an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Printf("Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

func startMonitoring() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Get working hours from environment variables
	startHour := getEnvInt("WORK_START_HOUR", 8) // Default 8 AM
	endHour := getEnvInt("WORK_END_HOUR", 18)    // Default 6 PM

	log.Printf("Starting queue monitoring (every 10 seconds) during working hours: %02d:00 - %02d:00 Europe/Warsaw timezone", startHour, endHour)

	for {
		select {
		case <-ticker.C:
			if isWithinWorkingHours(startHour, endHour) {
				if !isWeekend() {
					go fetchAndSaveData()
				} else {
					log.Printf("Weekend (Sat-Sun), skipping fetch")
				}
			} else {
				log.Printf("Outside working hours (%02d:00 - %02d:00 Europe/Warsaw), skipping fetch", startHour, endHour)
			}
		}
	}
}

func fetchAndSaveData() {
	log.Println("Fetching queue data...")

	// Get queue data
	queueData, err := fetchQueueData()
	if err != nil {
		log.Printf("Error fetching queue data: %v", err)
		return
	}

	// Process and save specific events
	for location, items := range queueData.Result {
		for _, item := range items {
			// Save "odbiór karty" events
			if item.Name == "odbiór karty" {
				if err := saveOdbiorkartyEvent(item, location); err != nil {
					log.Printf("Error saving odbiór karty event: %v", err)
				}
			}

			// Save "Odbiór karty - wieczory" events
			if item.Name == "Odbiór karty - wieczory" {
				if err := saveOdbiorkartyWieczoryEvent(item, location); err != nil {
					log.Printf("Error saving Odbiór karty - wieczory event: %v", err)
				}
			}
		}
	}

	log.Println("Data saved successfully")
}

func fetchQueueData() (*QueueData, error) {
	// Get random proxy URL
	proxyURL := GetRandomProxyUrl()
	log.Printf("Using proxy: %s", maskProxyURL(proxyURL))

	// Parse proxy URL
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy URL: %v", err)
	}

	// Create HTTP client with proxy
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
		Timeout: 30 * time.Second,
	}

	// Make request
	req, err := http.NewRequest("GET", "https://rezerwacje.duw.pl/status_kolejek/query.php?status", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/html, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse JSON response
	var queueData QueueData
	if err := json.NewDecoder(resp.Body).Decode(&queueData); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %v", err)
	}

	return &queueData, nil
}

func saveOdbiorkartyEvent(item QueueItem, location string) error {
	operationsJSON, err := json.Marshal(item.Operations)
	if err != nil {
		return fmt.Errorf("failed to marshal operations: %v", err)
	}

	// Check previous tickets_left for this queue and location
	var prevTicketsLeft sql.NullInt64
	prevQuery := `
		SELECT tickets_left
		FROM odbior_karty
		WHERE queue_id = $1 AND location = $2
		ORDER BY created_at DESC
		LIMIT 1`
	if err := db.QueryRow(prevQuery, item.ID, location).Scan(&prevTicketsLeft); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("failed to get previous tickets_left: %v", err)
		}
	}

	// Send notifications on state transitions
	if prevTicketsLeft.Valid {
		// <=0 -> >0 (tickets appeared)
		if prevTicketsLeft.Int64 <= 0 && item.TicketsLeft > 0 {
			message := fmt.Sprintf("🎉 Внимание! Появились талоны по услуге \"получение карты\" в %s (очередь %d). Доступно: %d ✅", location, item.ID, item.TicketsLeft)
			if err := sendTelegramMessage(message); err != nil {
				log.Printf("failed to send Telegram notification: %v", err)
			}
		}
		// >0 -> <=0 (tickets finished)
		if prevTicketsLeft.Int64 > 0 && item.TicketsLeft <= 0 {
			message := fmt.Sprintf("⛔️ Талоны по услуге \"получение карты\" закончились в %s (очередь %d).", location, item.ID)
			if err := sendTelegramMessage(message); err != nil {
				log.Printf("failed to send Telegram notification: %v", err)
			}
		}
	}

	query := `
	INSERT INTO odbior_karty (
		queue_id, name, location, ticket_count, tickets_served, workplaces,
		average_wait_time, average_service_time, registered_tickets, max_tickets,
		ticket_value, active, tickets_left, enabled, operations
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err = db.Exec(query,
		item.ID, item.Name, location, item.TicketCount, item.TicketsServed,
		item.Workplaces, item.AverageWaitTime, item.AverageServiceTime,
		item.RegisteredTickets, item.MaxTickets, item.TicketValue,
		item.Active, item.TicketsLeft, item.Enabled, string(operationsJSON))

	return err
}

func saveOdbiorkartyWieczoryEvent(item QueueItem, location string) error {
	operationsJSON, err := json.Marshal(item.Operations)
	if err != nil {
		return fmt.Errorf("failed to marshal operations: %v", err)
	}

	query := `
	INSERT INTO odbior_karty_wieczory (
		queue_id, name, location, ticket_count, tickets_served, workplaces,
		average_wait_time, average_service_time, registered_tickets, max_tickets,
		ticket_value, active, tickets_left, enabled, operations
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err = db.Exec(query,
		item.ID, item.Name, location, item.TicketCount, item.TicketsServed,
		item.Workplaces, item.AverageWaitTime, item.AverageServiceTime,
		item.RegisteredTickets, item.MaxTickets, item.TicketValue,
		item.Active, item.TicketsLeft, item.Enabled, string(operationsJSON))

	return err
}

func maskProxyURL(proxyURL string) string {
	// Mask password in proxy URL for logging
	u, err := url.Parse(proxyURL)
	if err != nil {
		return "invalid-proxy-url"
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	return u.String()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// isWeekend checks if today is Saturday or Sunday in Europe/Warsaw timezone
func isWeekend() bool {
	warsawTZ, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		log.Printf("Error loading Warsaw timezone: %v", err)
		return false
	}

	now := time.Now().In(warsawTZ)
	day := now.Weekday()
	return day == time.Saturday || day == time.Sunday
}

// sendTelegramMessage posts a message to a Telegram chat using bot API credentials from env
func sendTelegramMessage(text string) error {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if botToken == "" || chatID == "" {
		// Missing configuration; do not treat as fatal, just log and skip
		log.Printf("Telegram not configured (TELEGRAM_BOT_TOKEN/TELEGRAM_CHAT_ID missing), skipping message: %s", text)
		return nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)

	resp, err := http.PostForm(apiURL, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram sendMessage failed with status %d", resp.StatusCode)
	}
	return nil
}
