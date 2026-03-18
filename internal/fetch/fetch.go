package fetch

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Episode struct {
	ID          string
	PageURL     string
	Title       string
	MP3URL      string
	TrackNumber string
	Year        string
	CoverArt    []byte
}

var dayRe = regexp.MustCompile(`day-(\d+)`)

func DeriveIDFromURL(pageURL string) string {
	parts := strings.Split(strings.TrimSuffix(pageURL, "/"), "/")
	if len(parts) == 0 {
		return pageURL
	}
	return parts[len(parts)-1]
}

func extractTrackFromURL(pageURL string) string {
	m := dayRe.FindStringSubmatch(pageURL)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func FetchEpisode(client *http.Client, pageURL string) (*Episode, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "bible_audio_downloader/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(
		firstNonEmpty(
			metaContent(doc, "meta[name=title]"),
			metaContent(doc, "meta[property='og:title']"),
			doc.Find("title").First().Text(),
		),
	)
	if title == "" {
		return nil, errors.New("no title found")
	}

	mp3URL := strings.TrimSpace(
		firstNonEmpty(
			metaContent(doc, "meta[property='og:audio:secure_url']"),
			metaContent(doc, "meta[name='twitter:player:stream']"),
		),
	)
	if mp3URL == "" {
		return nil, errors.New("no mp3 url found")
	}

	cover, _ := fetchCoverArt(doc, client)

	ep := &Episode{
		ID:          DeriveIDFromURL(pageURL),
		PageURL:     pageURL,
		Title:       htmlUnescape(title),
		MP3URL:      mp3URL,
		TrackNumber: extractTrackFromURL(pageURL),
		Year:        deriveYearFromTitle(title),
		CoverArt:    cover,
	}

	return ep, nil
}

func metaContent(doc *goquery.Document, selector string) string {
	return strings.TrimSpace(doc.Find(selector).First().AttrOr("content", ""))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func htmlUnescape(s string) string {
	// Fireside meta titles are already plain text; hook for html.UnescapeString if needed.
	return s
}

func deriveYearFromTitle(title string) string {
	// Very simple heuristic: look for "(2022)" etc.
	for _, y := range []string{"2021", "2022", "2023", "2024", "2025"} {
		if strings.Contains(title, y) {
			return y
		}
	}
	return ""
}

func fetchCoverArt(doc *goquery.Document, client *http.Client) ([]byte, error) {
	imgURL := strings.TrimSpace(
		doc.Find(`meta[property='og:image']`).First().AttrOr("content", ""),
	)
	if imgURL == "" {
		return nil, fmt.Errorf("no og:image found")
	}

	req, err := http.NewRequest("GET", imgURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "bible_audio_downloader/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover status: %s", resp.Status)
	}

	return ioReadAll(resp.Body)
}

// small wrapper to avoid importing io in multiple places if you want to keep it minimal
func ioReadAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	buf := make([]byte, 0, 32*1024)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return buf, err
		}
	}
	return buf, nil
}

func (e *Episode) OutputFilename() string {
	// deterministic, safe filename based on ID
	name := strings.TrimSpace(e.ID)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "?", "")
	name = strings.ReplaceAll(name, "*", "")
	name = strings.ReplaceAll(name, "\"", "'")
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
	name = strings.ReplaceAll(name, "|", "-")
	return name + ".mp3"
}

