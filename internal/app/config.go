package app

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Environment      string
	HTTPAddr         string
	DatabasePath     string
	UploadDir        string
	TokenSecret      string
	ReminderCronHour int
	WeChatAppID      string
	WeChatAppSecret  string
	WeChatTemplateID string
}

func LoadConfig() Config {
	loadDotEnv(".env")
	hour, err := strconv.Atoi(value("REMINDER_CRON_HOUR", "9"))
	if err != nil || hour < 0 || hour > 23 {
		hour = 9
	}
	path := value("DATABASE_PATH", "./data/food-expiry.db")
	_ = os.MkdirAll(filepath.Dir(path), 0o750)
	uploadDir := value("UPLOAD_DIR", "./data/uploads")
	_ = os.MkdirAll(uploadDir, 0o750)
	return Config{
		Environment: value("APP_ENV", "development"), HTTPAddr: value("HTTP_ADDR", ":8080"),
		DatabasePath: path, TokenSecret: value("TOKEN_SECRET", "development-only-secret"),
		UploadDir:        uploadDir,
		ReminderCronHour: hour, WeChatAppID: os.Getenv("WECHAT_APP_ID"),
		WeChatAppSecret: os.Getenv("WECHAT_APP_SECRET"), WeChatTemplateID: os.Getenv("WECHAT_TEMPLATE_ID"),
	}
}

// loadDotEnv keeps local setup dependency-free. Existing environment variables
// always win, which is important for containers and deployment platforms.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || os.Getenv(parts[0]) != "" {
			continue
		}
		_ = os.Setenv(parts[0], strings.Trim(strings.TrimSpace(parts[1]), `"`))
	}
}

func value(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
