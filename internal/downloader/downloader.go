package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"zaycev-parser/internal/logger"
	"zaycev-parser/internal/models"
)

func StartDownloader(ctx context.Context, wg *sync.WaitGroup, in <-chan models.Track, errCh chan<- error) {
	defer wg.Done()

	const maxConcurrentDownloads = 5
	semaphore := make(chan struct{}, maxConcurrentDownloads)
	outputDir := "downloads"
	os.MkdirAll(outputDir, 0755)

	logger.Info("Downloader started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Downloader stopped by context")
			return
		case track, ok := <-in:
			if !ok {
				logger.Info("Downloader input channel closed")
				return
			}

			if track.Mp3URL == "" {
				logger.Warnf("No Mp3URL for track: %s", track.Title)
				continue
			}

			semaphore <- struct{}{}
			go func(t models.Track) {
				defer func() { <-semaphore }()

				logger.Debug("Downloading:", t.Title)

				if err := downloadTrack(t, outputDir); err != nil {
					errCh <- fmt.Errorf("download %s: %w", t.Title, err)
					return
				}

				logger.Info("Downloaded:", t.Title)
			}(track)
		}
	}
}

func downloadTrack(t models.Track, outputDir string) error {
	resp, err := http.Get(t.Mp3URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	filename := sanitizeFilename(fmt.Sprintf("%s - %s.mp3", t.Artist, t.Title))
	path := filepath.Join(outputDir, filename)

	if _, err := os.Stat(path); err == nil {
		logger.Debug("File already exists, skipping:", filename)
		return nil
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/:*?"<>|`, r) {
			return -1
		}
		return r
	}, name)
}
