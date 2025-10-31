package setup

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all environment-level settings
type Config struct {
	GitHubToken string

	SMTPServer    string
	SMTPPort      string
	SMTPUser      string
	SMTPPassword  string
	InventoryPath string
}

// Setup initializes configuration from .env or environment variables
func Setup() *Config {
	// Load .env if available
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] .env file not found — using system environment variables.")
	}

	cfg := &Config{
		GitHubToken:   os.Getenv("GITHUB_TOKEN"),
		SMTPServer:    os.Getenv("SMTP_SERVER"),
		SMTPPort:      os.Getenv("SMTP_PORT"),
		SMTPUser:      os.Getenv("SMTP_USER"),
		SMTPPassword:  os.Getenv("SMTP_PASSWORD"),
		InventoryPath: os.Getenv("INVENTORY_PATH"),
	}

	validateConfig(cfg)
	return cfg
}

// validateConfig ensures required fields have defaults or warnings
func validateConfig(cfg *Config) {
	if cfg.InventoryPath == "" {
		cfg.InventoryPath = "inventory.json"
		log.Println("[INFO] Defaulting INVENTORY_PATH to inventory.json")
	}
	if cfg.GitHubToken == "" {
		log.Println("[WARNING] No GitHub token provided — GitHub API rate limits will be low.")
	}
}
