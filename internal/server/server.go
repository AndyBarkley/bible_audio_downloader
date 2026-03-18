package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"bible_audio_downloader/internal/state"
	"log/slog"
)

func StartStatusServer(st *state.ServiceState, addr string) {
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		snap := stSnapshot(st)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		snap := stSnapshot(st)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<html><body>")
		fmt.Fprintf(w, "<h1>Bible Audio Downloader</h1>")
		fmt.Fprintf(w, "<p>Started: %s</p>", snap.StartedAt.Format(time.RFC3339))
		fmt.Fprintf(w, "<p>Last Run: %s</p>", snap.LastRun.Format(time.RFC3339))
		fmt.Fprintf(w, "<table border='1' cellpadding='6'>")
		fmt.Fprintf(w, "<tr><th>ID</th><th>Title</th><th>Status</th><th>ID3</th><th>Error</th></tr>")
		for _, ep := range snap.Episodes {
			fmt.Fprintf(w,
				"<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
				ep.ID,
				ep.Title,
				ep.Status,
				fmt.Sprintf("T:%v A:%v Alb:%v Tr:%v Cov:%v",
					ep.ID3.HasTitle,
					ep.ID3.HasArtist,
					ep.ID3.HasAlbum,
					ep.ID3.HasTrack,
					ep.ID3.HasCover,
				),
				ep.Error,
			)
		}
		fmt.Fprintf(w, "</table></body></html>")
	})

	go func() {
		slog.Info("status_server_started", "addr", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			slog.Error("status_server_error", "error", err.Error())
		}
	}()
}

func stSnapshot(st *state.ServiceState) state.ServiceState {
	return stSnapshotImpl(st)
}

// small wrapper to keep snapshot logic in one place
func stSnapshotImpl(st *state.ServiceState) state.ServiceState {
	return st.snapshot()
}
