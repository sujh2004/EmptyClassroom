package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port            string
	DatabaseDSN     string
	AllowedOrigins  []string
	EnableScheduler bool
	SyncOnStart     bool
	AutoMigrate     bool
	Timezone        string
	BUPT            BUPTConfig
}

type BUPTConfig struct {
	UserNo            string
	Password          string
	EncryptedPassword string
	LoginURL          string
	ClassroomURL      string
	Referer           string
	UserAgent         string
	Timeout           time.Duration
}

func Load(dotEnvPath string) (Config, error) {
	if dotEnvPath != "" {
		if err := loadDotEnv(dotEnvPath); err != nil && !os.IsNotExist(err) {
			return Config{}, err
		}
	}

	cfg := Config{
		Port:            getenv("PORT", "8080"),
		DatabaseDSN:     databaseDSN(),
		AllowedOrigins:  splitCSV(getenv("ALLOWED_ORIGINS", "*")),
		EnableScheduler: getenvBool("ENABLE_SCHEDULER", true),
		SyncOnStart:     getenvBool("RUN_SYNC_ON_START", false),
		AutoMigrate:     getenvBool("AUTO_MIGRATE", true),
		Timezone:        getenv("TZ", "Asia/Shanghai"),
		BUPT: BUPTConfig{
			UserNo:            os.Getenv("BUPT_USER_NO"),
			Password:          os.Getenv("BUPT_PASSWORD"),
			EncryptedPassword: os.Getenv("BUPT_ENCRYPTED_PWD"),
			LoginURL:          getenv("BUPT_LOGIN_URL", "http://jwglweixin.bupt.edu.cn/bjyddx/login"),
			ClassroomURL:      getenv("BUPT_CLASSROOM_URL", "http://jwglweixin.bupt.edu.cn/bjyddx/todayClassrooms"),
			Referer:           getenv("BUPT_REFERER", "https://jwglweixin.bupt.edu.cn/sjd/"),
			UserAgent: getenv("BUPT_USER_AGENT",
				"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"),
			Timeout: time.Duration(getenvInt("BUPT_TIMEOUT_SECONDS", 20)) * time.Second,
		},
	}

	if cfg.DatabaseDSN == "" {
		return Config{}, fmt.Errorf("database dsn is empty")
	}
	return cfg, nil
}

func databaseDSN() string {
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		return dsn
	}
	user := getenv("MYSQL_USER", "empty")
	password := getenv("MYSQL_PASSWORD", "empty")
	host := getenv("MYSQL_HOST", "127.0.0.1")
	port := getenv("MYSQL_PORT", "3306")
	name := getenv("MYSQL_DATABASE", "empty_classroom")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&charset=utf8mb4", user, password, host, port, name)
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, trimEnvValue(value))
	}
	return scanner.Err()
}

func trimEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
