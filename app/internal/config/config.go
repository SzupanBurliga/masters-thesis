package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	TLS, Auth, Validation, RateLimit, Complexity bool

	RateLimitRPS   float64
	RateLimitBurst int
	MaxBodyBytes   int64
	MaxDepth       int
	MaxCost        int
	MaxAliases     int
	MaxArrayLen    int
	MaxJSONDepth   int

	DBURL       string
	JWTSecret   string
	ListenAddr  string
	MetricsAddr string
	CertFile    string
	KeyFile     string
	PoolSize    int
	HTTP2       bool // HTTP/2 po TLS (REST, GraphQL); domyślnie wyłączone
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	b := func(k string) bool { v, _ := strconv.ParseBool(env(k, "false")); return v }
	f := func(k, d string) float64 { v, _ := strconv.ParseFloat(env(k, d), 64); return v }
	i := func(k, d string) int { v, _ := strconv.Atoi(env(k, d)); return v }
	body, _ := strconv.ParseInt(env("MAX_BODY_BYTES", "65536"), 10, 64)
	return Config{
		TLS: b("TLS_ENABLED"), Auth: b("AUTH_ENABLED"),
		Validation: b("VALIDATION_ENABLED"), RateLimit: b("RATELIMIT_ENABLED"),
		Complexity:     b("COMPLEXITY_ENABLED"),
		RateLimitRPS:   f("RATELIMIT_RPS", "100"),
		RateLimitBurst: i("RATELIMIT_BURST", "200"),
		MaxBodyBytes:   body,
		MaxDepth:       i("MAX_DEPTH", "6"),
		MaxCost:        i("MAX_COST", "400"),
		MaxAliases:     i("MAX_ALIASES", "10"),
		MaxArrayLen:    i("MAX_ARRAY_LEN", "100"),
		MaxJSONDepth:   i("MAX_JSON_DEPTH", "8"),
		DBURL:          env("DB_URL", "postgres://postgres:test@localhost:5432/shop"),
		JWTSecret:      env("JWT_SECRET", "thesis-secret"),
		ListenAddr:     env("LISTEN_ADDR", ":8443"),
		MetricsAddr:    env("METRICS_ADDR", ":9100"),
		CertFile:       env("TLS_CERT", "/certs/server.crt"),
		KeyFile:        env("TLS_KEY", "/certs/server.key"),
		PoolSize:       i("DB_POOL", "20"),
		HTTP2:          b("HTTP2_ENABLED"),
	}
}

// Log wypisuje aktywną konfigurację (używane do weryfikacji przebiegów).
func (c Config) Log(iface string) {
	log.Printf("config iface=%s TLS=%v AUTH=%v VALIDATION=%v RATELIMIT=%v COMPLEXITY=%v rps=%.0f burst=%d maxbody=%d maxdepth=%d maxcost=%d http2=%v",
		iface, c.TLS, c.Auth, c.Validation, c.RateLimit, c.Complexity,
		c.RateLimitRPS, c.RateLimitBurst, c.MaxBodyBytes, c.MaxDepth, c.MaxCost, c.HTTP2)
}
