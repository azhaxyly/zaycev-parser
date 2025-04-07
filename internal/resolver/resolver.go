package resolver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"zaycev-parser/internal/fetcher"
	"zaycev-parser/internal/logger"
	"zaycev-parser/internal/models"
)

type requestPayload struct {
	URL string `json:"url"`
}

type filezMetaResponse struct {
	URL   string `json:"url"`
	Track struct {
		File string `json:"file"`
	} `json:"track"`
}

func ResolveMp3URL(ctx context.Context, in <-chan fetcher.RawTrack, out chan<- models.Track, errCh chan<- error) {
	client := &http.Client{Timeout: 10 * time.Second}
	logger.Info("Resolver started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Resolver stopped by context")
			return
		case raw, ok := <-in:
			if !ok {
				logger.Info("Resolver input channel closed")
				return
			}

			go func(rt fetcher.RawTrack) {
				logger.Debugf("Resolving mp3 for: %s", rt.Title)

				mp3url, err := getMp3URL(client, rt.Slug)
				if err != nil {
					errCh <- fmt.Errorf("mp3 for %s: %w", rt.Title, err)
					return
				}

				if mp3url == "" {
					logger.Warnf("No mp3 URL for: %s", rt.Title)
				}

				logger.Debugf("Resolved mp3: %s → %s", rt.Title, mp3url)

				track := models.Track{
					Title:    rt.Title,
					Artist:   rt.Artist,
					Duration: rt.Duration,
					CoverURL: rt.CoverURL,
					Mp3URL:   mp3url,
					PageURL:  "https://zaycev.net" + rt.Slug,
				}

				select {
				case <-ctx.Done():
					return
				case out <- track:
				}
			}(raw)
		}
	}
}

func getMp3URL(client *http.Client, slug string) (string, error) {
	payload := requestPayload{URL: slug}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://zaycev.net/api/external/track/filezmeta", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://zaycev.net")
	req.Header.Set("Referer", "https://zaycev.net/popular/index.html")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// [] — значит, нет ссылки
	if string(body) == "[]" {
		return "", nil
	}

	// пробуем как объект
	var single filezMetaResponse
	if err := json.Unmarshal(body, &single); err == nil && single.URL != "" {
		return single.URL, nil
	}

	// пробуем как массив объектов
	var arr []filezMetaResponse
	if err := json.Unmarshal(body, &arr); err == nil && len(arr) > 0 && arr[0].URL != "" {
		return arr[0].URL, nil
	}

	return "", fmt.Errorf("unexpected JSON format: %s", string(body))
}
