package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	Env             string
	BaseURL         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	HeaderTimeout   time.Duration
	TrustedProxies  []string
	TLS             *TLSConfig
	LoginRatePerMin int
	LoginBurst      int
}

type TLSConfig struct {
	CertFile string
	KeyFile  string
}

func Load() (*Config, error) {
	cfg := &Config{
		Addr:            getEnv("ADDR", ":8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "data.db"),
		Env:             getEnv("ENV", "dev"),
		BaseURL:         strings.TrimRight(getEnv("BASE_URL", "http://localhost:8080"), "/"),
		ReadTimeout:     getDurationEnv("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getDurationEnv("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getDurationEnv("IDLE_TIMEOUT", 120*time.Second),
		HeaderTimeout:   getDurationEnv("HEADER_TIMEOUT", 5*time.Second),
		LoginRatePerMin: getIntEnv("LOGIN_RATE_PER_MIN", 30),
		LoginBurst:      getIntEnv("LOGIN_BURST", 5),
	}

	cert, key := os.Getenv("TLS_CERT_FILE"), os.Getenv("TLS_KEY_FILE")
	switch {
	case cert != "" && key != "":
		cfg.TLS = &TLSConfig{CertFile: cert, KeyFile: key}
	case cert != "" || key != "":
		return nil, fmt.Errorf("both TLS_CERT_FILE and TLS_KEY_FILE must be set, or neither")
	}

	if raw := os.Getenv("TRUSTED_PROXIES"); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				cfg.TrustedProxies = append(cfg.TrustedProxies, p)
			}
		}
	}

	if cfg.LoginRatePerMin <= 0 || cfg.LoginBurst <= 0 {
		return nil, fmt.Errorf("LOGIN_RATE_PER_MIN and LOGIN_BURST must be > 0")
	}
	return cfg, nil
}

// Diff returns the list of fields that differ between two configs.
// The SIGHUP handler uses it: some fields are applied live, others require
// a full restart (see §5).
func (c *Config) Diff(next *Config) []string {
	var d []string
	if c.Addr != next.Addr {
		d = append(d, "ADDR")
	}
	if c.DatabaseURL != next.DatabaseURL {
		d = append(d, "DATABASE_URL")
	}
	if c.ReadTimeout != next.ReadTimeout {
		d = append(d, "READ_TIMEOUT")
	}
	if c.WriteTimeout != next.WriteTimeout {
		d = append(d, "WRITE_TIMEOUT")
	}
	if c.IdleTimeout != next.IdleTimeout {
		d = append(d, "IDLE_TIMEOUT")
	}
	if c.HeaderTimeout != next.HeaderTimeout {
		d = append(d, "HEADER_TIMEOUT")
	}
	if c.BaseURL != next.BaseURL {
		d = append(d, "BASE_URL")
	}
	if (c.TLS == nil) != (next.TLS == nil) {
		d = append(d, "TLS")
	}
	if c.LoginRatePerMin != next.LoginRatePerMin {
		d = append(d, "LOGIN_RATE_PER_MIN")
	}
	if c.LoginBurst != next.LoginBurst {
		d = append(d, "LOGIN_BURST")
	}
	if c.Env != next.Env {
		d = append(d, "ENV")
	}
	if fmt.Sprint(c.TrustedProxies) != fmt.Sprint(next.TrustedProxies) {
		d = append(d, "TRUSTED_PROXIES")
	}
	return d
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDurationEnv(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getIntEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}
