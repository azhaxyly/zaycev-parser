package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"zaycev-parser/internal/logger"
)

type RawTrack struct {
	ID       int
	Title    string
	Artist   string
	Duration string
	CoverURL string
	Slug     string
}

type ApiTrack struct {
	Track           string `json:"track"`
	ArtistName      string `json:"artistName"`
	Duration        string `json:"duration"`
	ImageJpg        string `json:"imageJpg"`
	PlaybackEnabled bool   `json:"playbackEnabled"`
	DownloadEnabled bool   `json:"downloadEnabled"`
}

type ApiResponse struct {
	Page       int                 `json:"page"`
	TrackIds   []int               `json:"trackIds"`
	TracksInfo map[string]ApiTrack `json:"tracksInfo"`
}

func StartFetching(ctx context.Context, total int, period string, out chan<- RawTrack, errCh chan<- error) {
	const pageSize = 50
	client := &http.Client{Timeout: 10 * time.Second}

	pages := (total + pageSize - 1) / pageSize
	logger.Infof("Starting fetcher: %d tracks, %d pages, period=%s", total, pages, period)

	for page := 1; page <= pages; page++ {
		select {
		case <-ctx.Done():
			logger.Info("Fetcher stopped by context")
			return
		default:
			go func(p int) {
				logger.Debugf("Fetching page %d...", p)
				tracks, err := fetchPage(client, p, pageSize, period)
				if err != nil {
					errCh <- fmt.Errorf("page %d: %w", p, err)
					return
				}
				logger.Infof("Fetched %d tracks from page %d", len(tracks), p)
				for _, t := range tracks {
					select {
					case <-ctx.Done():
						return
					case out <- t:
					}
				}
			}(page)
		}
	}
}

func fetchPage(client *http.Client, page, limit int, period string) ([]RawTrack, error) {
	url := fmt.Sprintf("https://zaycev.net/api/external/pages/index/top?page=%d&limit=%d&period=%s&entity=track", page, limit, period)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", "https://zaycev.net")
	req.Header.Set("Referer", "https://zaycev.net/popular/index.html")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result ApiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var rawTracks []RawTrack
	for _, id := range result.TrackIds {
		idStr := strconv.Itoa(id)
		info, ok := result.TracksInfo[idStr]
		if !ok {
			continue
		}
		// if !info.DownloadEnabled {
		// 	continue
		// }

		raw := RawTrack{
			ID:       id,
			Title:    info.Track,
			Artist:   info.ArtistName,
			Duration: info.Duration,
			CoverURL: info.ImageJpg,
			Slug:     fmt.Sprintf("/pages/index/top/track/%d", id), // для filezmeta
		}
		rawTracks = append(rawTracks, raw)
	}

	return rawTracks, nil
}
