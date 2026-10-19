package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadEnvFile loads KEY=VALUE pairs from a file into os.Environ.
// Already-set variables are NOT overwritten (env > file).
// Missing file is not an error.
func LoadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return sc.Err()
}

// ForceReloadEnvFile clears known keys and re-reads .env.
// Called from the SIGHUP handler.
func ForceReloadEnvFile(path string) error {
	for _, k := range []string{
		"ENV", "ADDR", "DATABASE_URL", "BASE_URL",
		"READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT", "HEADER_TIMEOUT",
		"TRUSTED_PROXIES", "TLS_CERT_FILE", "TLS_KEY_FILE",
		"LOGIN_RATE_PER_MIN", "LOGIN_BURST",
	} {
		_ = os.Unsetenv(k)
	}
	return LoadEnvFile(path)
}
