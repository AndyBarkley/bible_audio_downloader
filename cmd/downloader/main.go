package main

import (
	"bufio"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bible_audio_downloader/internal/download"
	"bible_audio_downloader/internal/fetch"
	"bible_audio_downloader/internal/server"
	"bible_audio_downloader/internal/state"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <urls.txt> <output-dir>", os.Args[0])
	}

	urlsFile := os.Args[1]
	outDir := os.Args[2]

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	urls, err := readLines(urlsFile)
	if err != nil {
		logger.Error("read_urls_failed", "error", err.Error())
		os.Exit(1)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	st := state.NewServiceState()
	st.StartedAt = time.Now().UTC()

	// Pre-populate state with IDs/URLs
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || strings.HasPrefix(u, "#") {
			continue
		}
		id := fetch.DeriveIDFromURL(u)
		st.AddEpisode(state.EpisodeStatus{
			ID:      id,
			PageURL: u,
			Status:  "pending",
		})
	}

	// Start status server
	server.StartStatusServer(st, ":8080")

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || strings.HasPrefix(u, "#") {
			continue
		}

		id := fetch.DeriveIDFromURL(u)
		logger.Info("fetch_start", "episode_id", id, "url", u)
		st.UpdateStatus(id, func(es *state.EpisodeStatus) {
			es.Status = "fetching"
			es.UpdatedAt = time.Now().UTC()
		})

		ep, err := fetch.FetchEpisode(client, u)
		if err != nil {
			logger.Error("fetch_failed", "episode_id", id, "error", err.Error())
			st.UpdateStatus(id, func(es *state.EpisodeStatus) {
				es.Status = "error"
				es.Error = err.Error()
				es.UpdatedAt = time.Now().UTC()
			})
			continue
		}

		logger.Info("metadata_extracted",
			"episode_id", ep.ID,
			"title", ep.Title,
			"mp3_url", ep.MP3URL,
			"track", ep.TrackNumber,
		)

		st.UpdateStatus(ep.ID, func(es *state.EpisodeStatus) {
			es.Title = ep.Title
			es.Status = "downloading"
			es.UpdatedAt = time.Now().UTC()
		})

		destPath := filepath.Join(outDir, ep.OutputFilename())
		if _, err := os.Stat(destPath); err == nil {
			logger.Info("file_exists_skip", "episode_id", ep.ID, "path", destPath)
			st.UpdateStatus(ep.ID, func(es *state.EpisodeStatus) {
				es.Status = "downloaded"
				es.FilePath = destPath
				es.UpdatedAt = time.Now().UTC()
			})
			continue
		}

		if err := download.DownloadAndTag(client, ep, destPath); err != nil {
			logger.Error("download_or_tag_failed", "episode_id", ep.ID, "error", err.Error())
			st.UpdateStatus(ep.ID, func(es *state.EpisodeStatus) {
				es.Status = "error"
				es.Error = err.Error()
				es.UpdatedAt = time.Now().UTC()
			})
			continue
		}

		id3v, err := download.ValidateID3(destPath)
		if err != nil {
			logger.Error("id3_validation_failed", "episode_id", ep.ID, "error", err.Error())
		}

		logger.Info("download_complete",
			"episode_id", ep.ID,
			"path", destPath,
			"id3_title", id3v.HasTitle,
			"id3_artist", id3v.HasArtist,
			"id3_album", id3v.HasAlbum,
			"id3_track", id3v.HasTrack,
			"id3_cover", id3v.HasCover,
		)

		st.UpdateStatus(ep.ID, func(es *state.EpisodeStatus) {
			es.Status = "downloaded"
			es.FilePath = destPath
			es.ID3 = id3v
			es.UpdatedAt = time.Now().UTC()
		})

		st.LastRun = time.Now().UTC()
	}

	logger.Info("run_complete")
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}
