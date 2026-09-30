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

type Config struct {
	Env      string
	HTTP     HTTPConfig
	DB       DBConfig
	ONNX     ONNXConfig
	Jobs     JobsConfig
	Cache    CacheConfig
	Security SecurityConfig
	Push     PushConfig
}

type DBConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthTimeout   time.Duration
}

type HTTPConfig struct {
	Addr            string
	ShutdownTimeout time.Duration
}

type ONNXConfig struct {
	RuntimeLibPath string
	ModelPath      string
	TokenizerPath  string
	Workers        int
}

type JobsConfig struct {
	IngestCron           string
	CleanupCron          string
	CleanupRetentionDays int
}

// PushConfig holds Web Push (VAPID) settings. Push is optional: when the keys
// are empty the daily send job and the subscribe endpoints are disabled,
// mirroring the graceful-degradation approach used for ONNX embeddings.
type PushConfig struct {
	VAPIDPublic  string
	VAPIDPrivate string
	VAPIDSubject string
	Cron         string
	Timezone     string
	MinSources   int
	FreshHours   int // daily story must have a new article within this many hours
}

// Enabled reports whether Web Push is configured (both VAPID keys present).
func (p PushConfig) Enabled() bool {
	return p.VAPIDPublic != "" && p.VAPIDPrivate != ""
}

type CacheConfig struct {
	RedisURL string
}

type SecurityConfig struct {
	AllowedOrigins []string
}

func NewConfig() (*Config, error) {
	// .env is optional — only log real parse errors, not "file not found".
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: .env parse error: %v", err)
	}

	c := &Config{
		Env: getenvDefault("APP_ENV", "development"),
		HTTP: HTTPConfig{
			Addr:            getenvDefault("HTTP_ADDR", ":8080"),
			ShutdownTimeout: durationFromEnvDefault("HTTP_SHUTDOWN_TIMEOUT", 5*time.Second),
		},
		DB: DBConfig{
			URL:             os.Getenv("DATABASE_URL"),
			HealthTimeout:   durationFromEnvDefault("DB_HEALTH_TIMEOUT", 2*time.Second),
			MaxConns:        int32FromEnvDefault("DB_MAX_CONNS", 20),
			MinConns:        int32FromEnvDefault("DB_MIN_CONNS", 2),
			MaxConnLifetime: durationFromEnvDefault("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime: durationFromEnvDefault("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		},
		ONNX: ONNXConfig{
			RuntimeLibPath: getenvDefault("ONNX_RUNTIME_LIB_PATH", ""),
			ModelPath:      getenvDefault("ONNX_MODEL_PATH", "./models/paraphrase-multilingual-MiniLM-L12-v2/model.onnx"),
			TokenizerPath:  getenvDefault("ONNX_TOKENIZER_PATH", "./models/paraphrase-multilingual-MiniLM-L12-v2/tokenizer.json"),
			Workers:        int(int32FromEnvDefault("EMBEDDING_WORKERS", 4)),
		},
		Jobs: JobsConfig{
			IngestCron:           getenvDefault("INGEST_CRON", "*/15 * * * *"),
			CleanupCron:          getenvDefault("CLEANUP_CRON", "0 3 * * *"),
			CleanupRetentionDays: int(int32FromEnvDefault("CLEANUP_RETENTION_DAYS", 3)),
		},
		Cache: CacheConfig{
			RedisURL: getenvDefault("REDIS_URL", ""),
		},
		Security: SecurityConfig{
			AllowedOrigins: splitCommaEnv("ALLOWED_ORIGINS"),
		},
		Push: PushConfig{
			VAPIDPublic:  os.Getenv("VAPID_PUBLIC_KEY"),
			VAPIDPrivate: os.Getenv("VAPID_PRIVATE_KEY"),
			VAPIDSubject: os.Getenv("VAPID_SUBJECT"),
			Cron:         getenvDefault("PUSH_CRON", "0 9 * * *"),
			Timezone:     getenvDefault("PUSH_TIMEZONE", "Europe/Skopje"),
			MinSources:   int(int32FromEnvDefault("PUSH_MIN_SOURCES", 3)),
			FreshHours:   int(int32FromEnvDefault("PUSH_FRESH_HOURS", 16)),
		},
	}

	if c.DB.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if c.ONNX.Workers < 1 {
		c.ONNX.Workers = 1
	}

	if c.Jobs.CleanupRetentionDays < 1 {
		c.Jobs.CleanupRetentionDays = 1
	}

	if c.Push.MinSources < 1 {
		c.Push.MinSources = 1
	}

	if c.Push.FreshHours < 1 {
		c.Push.FreshHours = 16
	}

	if c.Push.Enabled() && c.Push.VAPIDSubject == "" {
		return nil, fmt.Errorf("VAPID_SUBJECT is required when VAPID keys are set (e.g. mailto:you@example.com)")
	}

	if c.Env == "production" && len(c.Security.AllowedOrigins) == 0 {
		log.Printf("warning: ALLOWED_ORIGINS is empty — CORS allows any origin; set it to your site's origin(s)")
	}

	if c.Env == "production" && !c.Push.Enabled() {
		log.Printf("warning: VAPID keys not set — Web Push notifications disabled")
	}

	return c, nil
}

func splitCommaEnv(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	var out []string
	for o := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
