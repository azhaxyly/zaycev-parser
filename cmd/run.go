package cmd

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"zaycev-parser/config"
	"zaycev-parser/internal/downloader"
	"zaycev-parser/internal/fetcher"
	"zaycev-parser/internal/logger"
	"zaycev-parser/internal/models"
	"zaycev-parser/internal/resolver"
	"zaycev-parser/internal/writer"
)

func Run() {
	logger.Init()

	cfg := config.ParseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalCh
		cancel()
	}()

	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	rawTrackChan := make(chan fetcher.RawTrack, 100)
	resolvedChan := make(chan models.Track, 100)
	writerChan := make(chan models.Track, 100)
	downloadChan := make(chan models.Track, 100)

	logger.Info("Starting fetcher...")
	go fetcher.StartFetching(ctx, cfg.Limit, cfg.Period, rawTrackChan, errCh)

	logger.Info("Starting resolver...")
	go resolver.ResolveMp3URL(ctx, rawTrackChan, resolvedChan, errCh)

	go func() {
		defer close(writerChan)
		defer close(downloadChan)
		for t := range resolvedChan {
			if t.Mp3URL == "" {
				logger.Warnf("Skipping track without Mp3URL: %s", t.Title)
			}

			writerChan <- t
			if cfg.Download && t.Mp3URL != "" {
				downloadChan <- t
			}
		}
	}()

	logger.Info("Starting writer...")
	wg.Add(1)
	logger.Debug("Starting fanout goroutine")
	go writer.StartWriter(ctx, &wg, cfg.Output, writerChan, errCh)

	if cfg.Download {
		logger.Info("Starting downloader...")
		wg.Add(1)
		go downloader.StartDownloader(ctx, &wg, downloadChan, errCh)
	}

	go func() {
		for err := range errCh {
			logger.Error(err)
		}
	}()

	wg.Wait()
}
