package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Env string

	HTTPAddr            string
	HTTPShutdownTimeout time.Duration

	// DatabaseURL is a Postgres DSN, e.g.:
	// postgres://user:pass@localhost:5432/dbname?sslmode=disable
	DatabaseURL string

	// Optional pool tuning.
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime time.Duration
	DBMaxConnIdleTime time.Duration

	// Used by readiness/health DB checks.
	DBHealthTimeout time.Duration

	// ONNX Runtime configuration for embeddings
	ONNXRuntimeLibPath string // Path to libonnxruntime.so
	ONNXModelPath      string // Path to model.onnx file
	ONNXTokenizerPath  string // Path to tokenizer.json file
	EmbeddingWorkers   int

	IngestCron        string // cron expression, e.g. "*/15 * * * *"
	CleanupCron       string // cron expression for DB cleanup, e.g. "0 3 * * *"
	CleanupRetentionDays int // articles older than this many days are deleted

	RedisURL string // e.g. redis://localhost:6379 — empty disables caching

	// AllowedOrigins is a comma-separated list of allowed CORS origins.
	// Empty string means all origins are allowed (wildcard). Replace with your domain in production.
	AllowedOrigins []string
}

func Load() (*Config, error) {
	// Load local .env if present (optional).
	// We intentionally ignore errors so production environments without a .env file work.
	err := godotenv.Load()
	if err != nil {
		log.Printf("failed to load .env file: %v", err)
	}

	c := &Config{
		Env:                 getenvDefault("APP_ENV", "development"),
		HTTPAddr:            getenvDefault("HTTP_ADDR", ":8080"),
		HTTPShutdownTimeout: durationFromEnvDefault("HTTP_SHUTDOWN_TIMEOUT", 5*time.Second),

		DatabaseURL:     os.Getenv("DATABASE_URL"),
		DBHealthTimeout: durationFromEnvDefault("DB_HEALTH_TIMEOUT", 2*time.Second),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	c.DBMaxConns = int32FromEnvDefault("DB_MAX_CONNS", 0)
	c.DBMinConns = int32FromEnvDefault("DB_MIN_CONNS", 0)
	c.DBMaxConnLifetime = durationFromEnvDefault("DB_MAX_CONN_LIFETIME", 0)
	c.DBMaxConnIdleTime = durationFromEnvDefault("DB_MAX_CONN_IDLE_TIME", 0)

	// ONNX Runtime configuration
	c.ONNXRuntimeLibPath = getenvDefault("ONNX_RUNTIME_LIB_PATH", "")
	c.ONNXModelPath = getenvDefault("ONNX_MODEL_PATH", "./models/paraphrase-multilingual-MiniLM-L12-v2/model.onnx")
	c.ONNXTokenizerPath = getenvDefault("ONNX_TOKENIZER_PATH", "./models/paraphrase-multilingual-MiniLM-L12-v2/tokenizer.json")
	c.EmbeddingWorkers = int(int32FromEnvDefault("EMBEDDING_WORKERS", 4))
	if c.EmbeddingWorkers < 1 {
		c.EmbeddingWorkers = 1
	}

	c.IngestCron = getenvDefault("INGEST_CRON", "*/15 * * * *")
	c.CleanupCron = getenvDefault("CLEANUP_CRON", "0 3 * * *")
	c.CleanupRetentionDays = int(int32FromEnvDefault("CLEANUP_RETENTION_DAYS", 3))
	if c.CleanupRetentionDays < 1 {
		c.CleanupRetentionDays = 1
	}
	c.RedisURL = getenvDefault("REDIS_URL", "")

	if raw := os.Getenv("ALLOWED_ORIGINS"); raw != "" {
		for _, o := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				c.AllowedOrigins = append(c.AllowedOrigins, trimmed)
			}
		}
	}

	return c, nil
}

func getenvDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func int32FromEnvDefault(key string, def int32) int32 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return def
	}
	return int32(n)
}

func durationFromEnvDefault(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
