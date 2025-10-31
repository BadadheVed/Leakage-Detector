package route

import (
	"context"
	"net/http"
	"time"

	scanner "github.com/BadadheVed/leakage-detector/scanner"
	"github.com/BadadheVed/leakage-detector/setup"
	"github.com/gin-gonic/gin"
)

func ScanRepo(c *gin.Context, cfg *setup.Config) {
	repoURL := c.Param("url")
	if repoURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo URL is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	repoChan := make(chan string, 1)
	resultChan := make(chan scanner.LeakResult, 100)
	errChan := make(chan error, 10)

	repoChan <- repoURL
	close(repoChan)

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
		done <- struct{}{}
	}()

	for err := range errChan {
		errors = append(errors, err.Error())
	}

	<-done

	c.JSON(http.StatusOK, gin.H{
		"repo":    repoURL,
		"results": results,
		"errors":  errors,
		"status":  "scan complete",
	})
}
