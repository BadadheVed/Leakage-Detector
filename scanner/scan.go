package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

// InventoryItem matches your inventory.json schema
type InventoryItem struct {
	ID         string `json:"id"`
	Provider   string `json:"provider"`
	TokenType  string `json:"token_type"`
	TokenValue string `json:"token_value"`
	Owner      string `json:"owner"`
	Notes      string `json:"notes"`
}

// LeakResult is emitted for each detected leak
type LeakResult struct {
	InventoryID string    `json:"inventory_id"`
	Provider    string    `json:"provider"`
	TokenType   string    `json:"token_type"`
	Matched     string    `json:"matched_value"`
	RepoURL     string    `json:"repo_url"`
	FilePath    string    `json:"file_path"`
	BlobURL     string    `json:"blob_url"`
	Snippet     string    `json:"snippet"`
	Timestamp   time.Time `json:"timestamp"`
}

func StartScanner(
	ctx context.Context,
	githubToken string,
	repoChan <-chan string,
	resultChan chan<- LeakResult,
	errChan chan<- error,

	perRepoTimeout time.Duration,
	fileWorkerCount int,
	inventoryPath string,
) {

	inv, err := LoadInventory(inventoryPath)
	if err != nil {

		errChan <- fmt.Errorf("failed to load inventory: %w", err)
		close(resultChan)
		close(errChan)
		return
	}
	log.Printf("[scanner] loaded %d inventory items\n", len(inv))

	var ghClient *github.Client
	{
		ctx0 := context.Background()
		if strings.TrimSpace(githubToken) != "" {
			ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
			tc := oauth2.NewClient(ctx0, ts)
			ghClient = github.NewClient(tc)
		} else {
			ghClient = github.NewClient(nil)
		}
	}

	var wg sync.WaitGroup

	workerCount := 4
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			log.Printf("[scanner][worker-%d] started\n", workerID)
			for {
				select {
				case <-ctx.Done():

					log.Printf("[scanner][worker-%d] global context canceled: %v\n", workerID, ctx.Err())
					return
				case repoURL, ok := <-repoChan:
					if !ok {

						log.Printf("[scanner][worker-%d] repo channel closed\n", workerID)
						return
					}

					repoCtx, cancel := context.WithTimeout(ctx, perRepoTimeout)

					if cerr := scanRepositoryWithFilePool(repoCtx, ghClient, repoURL, inv, resultChan, errChan, fileWorkerCount); cerr != nil {

						select {
						case errChan <- fmt.Errorf("worker-%d repo=%s error=%w", workerID, repoURL, cerr):
						case <-ctx.Done():
							log.Printf("[scanner][worker-%d] global context canceled while sending error: %v\n", workerID, ctx.Err())
							cancel()
							return
						}
					}
					cancel()
				}
			}
		}(i + 1)
	}

	// Wait for workers then close result and error channels
	go func() {
		wg.Wait()
		// close channels to signal caller no more results
		close(resultChan)
		close(errChan)
		log.Printf("[scanner] all workers finished, closed result and error channels\n")
	}()
}

// LoadInventory loads inventory.json (or any JSON file with the same schema)
func LoadInventory(path string) ([]InventoryItem, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var inv []InventoryItem
	if err := jsonUnmarshal(b, &inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// jsonUnmarshal is a tiny wrapper so we can change unmarshalling behavior easily later.
// Using encoding/json directly here keeps it simple.
func jsonUnmarshal(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

func scanRepositoryWithFilePool(
	ctx context.Context,
	client *github.Client,
	repoURL string,
	inv []InventoryItem,
	resultChan chan<- LeakResult,
	errChan chan<- error,
	fileWorkerCount int,
) error {
	// Validate and parse repo URL (expecting forms like "owner/repo" or "https://github.com/owner/repo")
	owner, repo, err := parseOwnerRepo(repoURL)
	if err != nil {
		return fmt.Errorf("parse repo URL: %w", err)
	}
	log.Printf("[scanRepo] starting %s/%s\n", owner, repo)

	// fileTasks channel for file paths discovered in repo traversal
	fileTasks := make(chan string, 128)
	var fileWG sync.WaitGroup

	// Start file worker pool for this repo
	for i := 0; i < fileWorkerCount; i++ {
		fileWG.Add(1)
		go func(worker int) {
			defer fileWG.Done()
			for {
				select {
				case <-ctx.Done():

					return
				case path, ok := <-fileTasks:
					if !ok {
						return
					}

					if cerr := scanFile(ctx, client, owner, repo, path, inv, resultChan); cerr != nil {

						select {
						case errChan <- fmt.Errorf("scanFile error repo=%s/%s path=%s: %w", owner, repo, path, cerr):
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}(i + 1)
	}

	// Recursively walk repository contents and push file paths into fileTasks
	err = walkRepoPaths(ctx, client, owner, repo, "", fileTasks)
	// Close the fileTasks channel to signal workers no more files
	close(fileTasks)
	// Wait for workers to finish
	fileWG.Wait()

	if err != nil {
		// returning error to caller (worker). Caller will send into errChan.
		return fmt.Errorf("walk repo error: %w", err)
	}

	log.Printf("[scanRepo] finished %s/%s\n", owner, repo)
	return nil
}

func walkRepoPaths(ctx context.Context, client *github.Client, owner, repo, path string, fileTasks chan<- string) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fileContent, dirContent, _, err := client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {

		return err
	}

	if dirContent != nil {
		for _, item := range dirContent {

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if item == nil || item.Type == nil || item.Path == nil {
				continue
			}
			itemType := item.GetType()
			itemPath := item.GetPath()
			if itemType == "dir" {

				if err := walkRepoPaths(ctx, client, owner, repo, itemPath, fileTasks); err != nil {

					log.Printf("[walkRepoPaths] error walking subdir %s: %v", itemPath, err)
				}
			} else if itemType == "file" {

				select {
				case <-ctx.Done():
					return ctx.Err()
				case fileTasks <- itemPath:
				}
			}
		}
	} else if fileContent != nil {

		select {
		case <-ctx.Done():
			return ctx.Err()
		case fileTasks <- fileContent.GetPath():
		}
	}

	return nil
}

func scanFile(ctx context.Context, client *github.Client, owner, repo, filePath string, inv []InventoryItem, resultChan chan<- LeakResult) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, filePath, nil)
	if err != nil {
		return fmt.Errorf("get contents: %w", err)
	}
	if fileContent == nil {
		return fmt.Errorf("empty file content for %s", filePath)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return fmt.Errorf("get content decode: %w", err)
	}

	for _, it := range inv {
		// skip empty tokens
		if strings.TrimSpace(it.TokenValue) == "" {
			continue
		}
		if strings.Contains(content, it.TokenValue) {

			snippet := extractSnippet(content, it.TokenValue, 120)

			blobURL := buildBlobURL(owner, repo, filePath)

			lr := LeakResult{
				InventoryID: it.ID,
				Provider:    it.Provider,
				TokenType:   it.TokenType,
				Matched:     it.TokenValue,
				RepoURL:     fmt.Sprintf("https://github.com/%s/%s", owner, repo),
				FilePath:    filePath,
				BlobURL:     blobURL,
				Snippet:     snippet,
				Timestamp:   time.Now().UTC(),
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case resultChan <- lr:

			}
		}
	}

	return nil
}

func extractSnippet(content, token string, maxLen int) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, token) {
			if len(line) > maxLen {

				idx := strings.Index(line, token)
				start := idx - (maxLen / 2)
				if start < 0 {
					start = 0
				}
				end := start + maxLen
				if end > len(line) {
					end = len(line)
				}
				return strings.TrimSpace(line[start:end]) + "..."
			}
			return strings.TrimSpace(line)
		}
	}
	// fallback: return a trimmed prefix
	if len(content) > maxLen {
		return strings.TrimSpace(content[:maxLen]) + "..."
	}
	return strings.TrimSpace(content)
}

func parseOwnerRepo(input string) (string, string, error) {
	trim := strings.TrimSpace(input)
	// strip protocol if present
	trim = strings.TrimPrefix(trim, "https://")
	trim = strings.TrimPrefix(trim, "http://")
	trim = strings.TrimPrefix(trim, "github.com/")
	trim = strings.TrimPrefix(trim, "www.github.com/")
	parts := strings.Split(strings.Trim(trim, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repo identifier: %s", input)
	}
	return parts[0], parts[1], nil
}

func buildBlobURL(owner, repo, path string) string {
	return fmt.Sprintf("https://github.com/%s/%s/blob/main/%s", owner, repo, path)
}
