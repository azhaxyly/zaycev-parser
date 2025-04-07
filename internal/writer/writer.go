package writer

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"zaycev-parser/internal/logger"
	"zaycev-parser/internal/models"
)

func StartWriter(ctx context.Context, wg *sync.WaitGroup, format string, in <-chan models.Track, errCh chan<- error) {
	defer wg.Done()

	var (
		file     *os.File
		writerFn func([]models.Track) error
		buffer   []models.Track
	)

	outputDir := "output"
	os.MkdirAll(outputDir, 0755)

	filename := filepath.Join(outputDir, "tracks."+format)
	f, err := os.Create(filename)
	if err != nil {
		errCh <- fmt.Errorf("writer: %w", err)
		return
	}
	file = f
	defer file.Close()

	switch format {
	case "json":
		writerFn = func(tracks []models.Track) error {
			enc := json.NewEncoder(file)
			enc.SetIndent("", "  ")
			return enc.Encode(tracks)
		}
	case "csv":
		writerFn = func(tracks []models.Track) error {
			w := csv.NewWriter(file)
			defer w.Flush()

			if err := w.Write([]string{"Title", "Artist", "Duration", "CoverURL", "Mp3URL", "PageURL"}); err != nil {
				return err
			}
			for _, t := range tracks {
				err := w.Write([]string{
					t.Title, t.Artist, t.Duration, t.CoverURL, t.Mp3URL, t.PageURL,
				})
				if err != nil {
					return err
				}
			}
			return nil
		}
	default:
		errCh <- fmt.Errorf("unsupported output format: %s", format)
		return
	}

	logger.Infof("Writer started with format: %s", format)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Writer stopped by context")
			goto save
		case t, ok := <-in:
			if !ok {
				logger.Info("Writer input channel closed")
				goto save
			}
			buffer = append(buffer, t)
		}
	}

save:
	logger.Infof("Saving %d tracks to %s", len(buffer), filename)
	for _, track := range buffer {
		logger.Debugf("Track to write: %s → %s", track.Title, track.Mp3URL)
	}
	if err := writerFn(buffer); err != nil {
		errCh <- fmt.Errorf("writer error: %w", err)
		return
	}
	logger.Infof("Tracks successfully saved to %s", filename)
}
