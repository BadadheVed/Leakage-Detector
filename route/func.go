package route

import (
	"context"
	"net/http"
	"time"

	scanner "github.com/BadadheVed/leakage-detector/scanner"
	"github.com/BadadheVed/leakage-detector/setup"
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

	// Collect errors while results are being processed
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	<-done

	c.JSON(http.StatusOK, gin.H{
		"repo":    req.URL,
		"results": results,
		"errors":  errors,
		"status":  "scan complete",
	})
}
