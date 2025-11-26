package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Database struct {
	db *sql.DB
}

type StoredPacket struct {
	ID            int64     `json:"id"`
	Time          string    `json:"time"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Satellites    int       `json:"satellites"`
	AccelerationX float64   `json:"acceleration_x"`
	AccelerationY float64   `json:"acceleration_y"`
	AccelerationZ float64   `json:"acceleration_z"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewDatabase creates a new database connection
func NewDatabase(dsn string) (*Database, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &Database{db: db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// InsertPacket inserts a packet into the database
func (d *Database) InsertPacket(packet Packet) (int64, error) {
	query := `
		INSERT INTO packets (time, latitude, longitude, satellites, acceleration_x, acceleration_y, acceleration_z)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.db.Exec(query,
		packet.Time,
		packet.Latitude,
		packet.Longitude,
		packet.Satellites,
		packet.Acceleration[0],
		packet.Acceleration[1],
		packet.Acceleration[2],
	)

	if err != nil {
		return 0, fmt.Errorf("failed to insert packet: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// GetPackets retrieves packets from the database with optional limit
func (d *Database) GetPackets(limit int) ([]StoredPacket, error) {
	query := `
		SELECT id, time, latitude, longitude, satellites, 
		       acceleration_x, acceleration_y, acceleration_z, 
		       created_at, updated_at
		FROM packets 
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query packets: %w", err)
	}
	defer rows.Close()

	var packets []StoredPacket
	for rows.Next() {
		var p StoredPacket
		err := rows.Scan(
			&p.ID,
			&p.Time,
			&p.Latitude,
			&p.Longitude,
			&p.Satellites,
			&p.AccelerationX,
			&p.AccelerationY,
			&p.AccelerationZ,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan packet: %w", err)
		}
		packets = append(packets, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating packets: %w", err)
	}

	return packets, nil
}

// GetLatestPacket retrieves the most recent packet from the database
func (d *Database) GetLatestPacket() (*StoredPacket, error) {
	query := `
		SELECT id, time, latitude, longitude, satellites, 
		       acceleration_x, acceleration_y, acceleration_z, 
		       created_at, updated_at
		FROM packets 
		ORDER BY created_at DESC 
		LIMIT 1
	`

	var p StoredPacket
	err := d.db.QueryRow(query).Scan(
		&p.ID,
		&p.Time,
		&p.Latitude,
		&p.Longitude,
		&p.Satellites,
		&p.AccelerationX,
		&p.AccelerationY,
		&p.AccelerationZ,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No packets found
		}
		return nil, fmt.Errorf("failed to get latest packet: %w", err)
	}

	return &p, nil
}

// GetPacketCount returns the total number of packets in the database
func (d *Database) GetPacketCount() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM packets").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get packet count: %w", err)
	}
	return count, nil
}

// DeleteAllPackets removes all packets from the database (for testing)
func (d *Database) DeleteAllPackets() error {
	_, err := d.db.Exec("DELETE FROM packets")
	if err != nil {
		return fmt.Errorf("failed to delete all packets: %w", err)
	}
	return nil
}

// GetAccelerationSeries retrieves acceleration Z values for graphing
func (d *Database) GetAccelerationSeries(limit int) ([]float32, error) {
	query := `
		SELECT acceleration_z 
		FROM packets 
		ORDER BY created_at ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query acceleration series: %w", err)
	}
	defer rows.Close()

	var series []float32
	for rows.Next() {
		var value float64
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("failed to scan acceleration value: %w", err)
		}
		series = append(series, float32(value))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating acceleration values: %w", err)
	}

	return series, nil
}

// CreateTestPacket creates a test packet with mock data
func CreateTestPacket() Packet {
	return Packet{
		Time:       time.Now().Format("15:04:05"),
		Latitude:   54.687157 + (float64(time.Now().UnixNano()%1000) / 100000.0), // Vilnius area with variation
		Longitude:  25.279652 + (float64(time.Now().UnixNano()%1000) / 100000.0), // Vilnius area with variation
		Satellites: 8 + int(time.Now().UnixNano()%5),                             // 8-12 satellites
		Acceleration: [3]float64{
			float64(time.Now().UnixNano()%200-100) / 100.0, // -1.0 to 1.0
			float64(time.Now().UnixNano()%200-100) / 100.0, // -1.0 to 1.0
			float64(time.Now().UnixNano()%200-100) / 100.0, // -1.0 to 1.0
		},
	}
}

// getDatabaseDSN returns the database connection string
func getDatabaseDSN() string {
	// Try to get from environment variables first
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		return dsn
	}

	// Default configuration
	host := getEnvOrDefault("DB_HOST", "127.0.0.1")
	port := getEnvOrDefault("DB_PORT", "3306")
	user := getEnvOrDefault("DB_USER", "root")
	password := getEnvOrDefault("DB_PASSWORD", "")
	dbname := getEnvOrDefault("DB_NAME", "komkomunikacijos")

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SavePacketsToCSV exports packets to a CSV file
func (d *Database) SavePacketsToCSV(filename string, limit int) error {
	// Get packets from database
	packets, err := d.GetPackets(limit)
	if err != nil {
		return fmt.Errorf("failed to get packets: %w", err)
	}

	// Create the file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"ID", "Time", "Latitude", "Longitude", "Satellites",
		"AccelerationX", "AccelerationY", "AccelerationZ",
		"CreatedAt", "UpdatedAt",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write data rows
	for _, p := range packets {
		row := []string{
			strconv.FormatInt(p.ID, 10),
			p.Time,
			strconv.FormatFloat(p.Latitude, 'f', 6, 64),
			strconv.FormatFloat(p.Longitude, 'f', 6, 64),
			strconv.Itoa(p.Satellites),
			strconv.FormatFloat(p.AccelerationX, 'f', 3, 64),
			strconv.FormatFloat(p.AccelerationY, 'f', 3, 64),
			strconv.FormatFloat(p.AccelerationZ, 'f', 3, 64),
			p.CreatedAt.Format("2006-01-02 15:04:05"),
			p.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

// SavePacketsToJSON exports packets to a JSON file
func (d *Database) SavePacketsToJSON(filename string, limit int) error {
	// Get packets from database
	packets, err := d.GetPackets(limit)
	if err != nil {
		return fmt.Errorf("failed to get packets: %w", err)
	}

	// Create the file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create JSON encoder and write data
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print
	if err := encoder.Encode(packets); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// GenerateExportFilename creates a timestamped filename for exports
func GenerateExportFilename(format string) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("komkomunikacijos_data_%s.%s", timestamp, format)
}
