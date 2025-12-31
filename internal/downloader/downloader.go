package downloader

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// FileType represents a type of data file to download
type FileType string

const (
	FileCharity                 FileType = "charity"
	FileCharityTrustee          FileType = "charity_trustee"
	FileCharityAnnualReturnA    FileType = "charity_annual_return_parta"
	FileCharityAnnualReturnB    FileType = "charity_annual_return_partb"
	FileCharityAnnualReturnHist FileType = "charity_annual_return_history"
)

// baseURL is the Azure blob storage URL for Charity Commission data
const baseURL = "https://ccewuksprdoneregsadata1.blob.core.windows.net/data/json/publicextract.%s.zip"

// DownloadedFile represents a file that has been downloaded and extracted in memory
type DownloadedFile struct {
	Type     FileType
	FileName string
	Data     []byte
	Size     int64
}

// Downloader manages downloading and extracting Charity Commission data files
type Downloader struct {
	httpClient      *http.Client
	maxRetries      int
	retryDelay      time.Duration
	progressHandler func(fileType FileType, bytesDownloaded, totalBytes int64)
}

// Config holds configuration for the downloader
type Config struct {
	Timeout         time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	ProgressHandler func(fileType FileType, bytesDownloaded, totalBytes int64)
}

// NewDownloader creates a new downloader with the given configuration
func NewDownloader(config Config) *Downloader {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	return &Downloader{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		maxRetries:      config.MaxRetries,
		retryDelay:      config.RetryDelay,
		progressHandler: config.ProgressHandler,
	}
}

// DownloadFile downloads and extracts a single file in memory
func (d *Downloader) DownloadFile(ctx context.Context, fileType FileType) (*DownloadedFile, error) {
	url := fmt.Sprintf(baseURL, string(fileType))
	log.Printf("Downloading %s from %s", fileType, url)

	// Download the ZIP file with retries
	zipData, err := d.downloadWithRetry(ctx, url, fileType)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", fileType, err)
	}

	log.Printf("Download complete for %s (%d bytes), extracting...", fileType, len(zipData))

	// Extract the JSON file from the ZIP
	jsonData, fileName, err := extractJSONFromZip(zipData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract %s: %w", fileType, err)
	}

	log.Printf("Extraction complete for %s: %s (%d bytes)", fileType, fileName, len(jsonData))

	return &DownloadedFile{
		Type:     fileType,
		FileName: fileName,
		Data:     jsonData,
		Size:     int64(len(jsonData)),
	}, nil
}

// DownloadFiles downloads multiple files in parallel and returns them in memory
func (d *Downloader) DownloadFiles(ctx context.Context, fileTypes []FileType) (map[FileType]*DownloadedFile, error) {
	results := make(map[FileType]*DownloadedFile)
	errors := make(map[FileType]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, fileType := range fileTypes {
		wg.Add(1)
		go func(ft FileType) {
			defer wg.Done()

			file, err := d.DownloadFile(ctx, ft)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors[ft] = err
			} else {
				results[ft] = file
			}
		}(fileType)
	}

	wg.Wait()

	// Check if any critical errors occurred
	if len(errors) > 0 {
		var errMsg string
		for ft, err := range errors {
			errMsg += fmt.Sprintf("%s: %v; ", ft, err)
		}
		return results, fmt.Errorf("some downloads failed: %s", errMsg)
	}

	return results, nil
}

// downloadWithRetry downloads data from a URL with retry logic
func (d *Downloader) downloadWithRetry(ctx context.Context, url string, fileType FileType) ([]byte, error) {
	var lastErr error

	for attempt := 1; attempt <= d.maxRetries; attempt++ {
		if attempt > 1 {
			log.Printf("Retrying %s (attempt %d/%d)...", fileType, attempt, d.maxRetries)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(d.retryDelay):
			}
		}

		data, err := d.download(ctx, url, fileType)
		if err == nil {
			return data, nil
		}

		lastErr = err
		log.Printf("Download attempt %d failed for %s: %v", attempt, fileType, err)
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", d.maxRetries, lastErr)
}

// download performs a single download operation
func (d *Downloader) download(ctx context.Context, url string, fileType FileType) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read with progress tracking
	var buf bytes.Buffer
	totalBytes := resp.ContentLength
	var bytesRead int64

	// Create a buffer for efficient copying
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			buf.Write(buffer[:n])
			bytesRead += int64(n)

			// Report progress if handler is set
			if d.progressHandler != nil && totalBytes > 0 {
				d.progressHandler(fileType, bytesRead, totalBytes)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// extractJSONFromZip extracts the first JSON file from a ZIP archive
func extractJSONFromZip(zipData []byte) ([]byte, string, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read ZIP: %w", err)
	}

	// Find the first JSON file in the archive
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Open the file
		rc, err := file.Open()
		if err != nil {
			return nil, "", fmt.Errorf("failed to open file %s in ZIP: %w", file.Name, err)
		}
		defer rc.Close()

		// Read all content
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read file %s from ZIP: %w", file.Name, err)
		}

		return data, file.Name, nil
	}

	return nil, "", fmt.Errorf("no files found in ZIP archive")
}

// GetReader returns an io.Reader for a downloaded file
func (f *DownloadedFile) GetReader() io.Reader {
	return bytes.NewReader(f.Data)
}

// DefaultFileSet returns the standard set of files needed for a complete import
func DefaultFileSet() []FileType {
	return []FileType{
		FileCharity,
		FileCharityTrustee,
		FileCharityAnnualReturnA,
		FileCharityAnnualReturnB,
		FileCharityAnnualReturnHist,
	}
}
