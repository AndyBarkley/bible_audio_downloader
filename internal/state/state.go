package state

import (
	"sync"
	"time"
)

type ID3Validation struct {
	HasTitle  bool `json:"has_title"`
	HasArtist bool `json:"has_artist"`
	HasAlbum  bool `json:"has_album"`
	HasCover  bool `json:"has_cover"`
	HasTrack  bool `json:"has_track"`
}

type EpisodeStatus struct {
	ID        string        `json:"id"`
	PageURL   string        `json:"page_url"`
	Title     string        `json:"title"`
	Status    string        `json:"status"`
	UpdatedAt time.Time     `json:"updated_at"`
	Error     string        `json:"error,omitempty"`
	ID3       ID3Validation `json:"id3"`
	FilePath  string        `json:"file_path"`
}

type ServiceState struct {
	mu       sync.RWMutex
	StartedAt time.Time        `json:"started_at"`
	LastRun   time.Time        `json:"last_run"`
	Episodes  []EpisodeStatus  `json:"episodes"`
}

func NewServiceState() *ServiceState {
	return &ServiceState{
		Episodes: make([]EpisodeStatus, 0),
	}
}

func (s *ServiceState) snapshot() ServiceState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp := ServiceState{
		StartedAt: s.StartedAt,
		LastRun:   s.LastRun,
		Episodes:  make([]EpisodeStatus, len(s.Episodes)),
	}
	copy(cp.Episodes, s.Episodes)
	return cp
}

func (s *ServiceState) AddEpisode(es EpisodeStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Episodes = append(s.Episodes, es)
}

func (s *ServiceState) UpdateStatus(id string, fn func(*EpisodeStatus)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Episodes {
		if s.Episodes[i].ID == id {
			fn(&s.Episodes[i])
			return
		}
	}
}
