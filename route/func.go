package route

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	scanner "github.com/BadadheVed/leakage-detector/scanner"
	"github.com/BadadheVed/leakage-detector/setup"
	"github.com/BadadheVed/leakage-detector/utils"
	"github.com/gin-gonic/gin"
)

// Struct for JSON request body
type ScanRequest struct {
	URL string `json:"url" binding:"required"`
}

func ScanRepo(c *gin.Context, cfg *setup.Config) {
	var req ScanRequest

	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo URL is required in JSON body"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	repoChan := make(chan string, 1)
	resultChan := make(chan scanner.LeakResult, 100)
	errChan := make(chan error, 10)

	repoChan <- req.URL
	close(repoChan)

	// Start the scanner
	go scanner.StartScanner(
		ctx,
		cfg.GitHubToken,
		repoChan,
		resultChan,
		errChan,
		1*time.Minute, // per-repo timeout
		4,             // file worker count
		cfg.InventoryPath,
	)

	var (
		results []scanner.LeakResult
		errors  []string
	)

	done := make(chan struct{})
	go func() {
		for r := range resultChan {
			results = append(results, r)
		}
		close(done)
	}()

	for err := range errChan {
		errors = append(errors, err.Error())
	}

	<-done

	inventoryData, _ := os.ReadFile(cfg.InventoryPath)
	var inventory []scanner.InventoryItem
	_ = json.Unmarshal(inventoryData, &inventory)

	if len(results) == 0 {
		log.Printf("✅ No leaks found in repo: %s", req.URL)
	} else {
		log.Printf("🚨 %d potential leaks detected in repo: %s", len(results), req.URL)
	}

	for _, result := range results {
		for _, item := range inventory {
			if result.Matched == item.TokenValue {
				log.Printf("Leak matched for key: %s (owner: %s)", item.TokenType, item.Owner)
				go func(i scanner.InventoryItem) {
					if err := utils.SendLeakAlertMail(
						cfg.SMTPHost,
						cfg.SMTPPort,
						cfg.SMTPUser,
						cfg.SMTPPass,
						i.Owner,
						i.TokenType,
						i.TokenValue,
						req.URL,
					); err != nil {
						log.Printf("❌ Error sending email to %s: %v", i.Owner, err)
					}
				}(item)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"repo":    req.URL,
		"results": results,
		"errors":  errors,
		"status":  "scan complete",
	})
}
