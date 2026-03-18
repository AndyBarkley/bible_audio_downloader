package download

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"bible_audio_downloader/internal/fetch"
	"bible_audio_downloader/internal/state"

	id3v2 "github.com/bogem/id3v2/v2"
)

func DownloadAndTag(client *http.Client, ep *fetch.Episode, destPath string) error {
	tmpPath := destPath + ".part"

	req, err := http.NewRequest("GET", ep.MP3URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "bible_audio_downloader/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mp3 status: %s", resp.Status)
	}

	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(resp.Body); err != nil {
		out.Close()
		return err
	}
	out.Close()

	if err := setID3Tags(tmpPath, ep); err != nil {
		return fmt.Errorf("set id3: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return err
	}

	return nil
}

func setID3Tags(path string, ep *fetch.Episode) error {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()

	tag.SetTitle(ep.Title)
	tag.SetArtist("Fr. Mike Schmitz")
	tag.SetAlbum("The Bible in a Year (with Fr. Mike Schmitz)")

	if ep.TrackNumber != "" {
		tag.AddTextFrame("TRCK", id3v2.TextFrame{
			Encoding: id3v2.EncodingUTF8,
			Text:     ep.TrackNumber,
		})
	}

	if ep.Year != "" {
		tag.AddTextFrame("TDRC", id3v2.TextFrame{
			Encoding: id3v2.EncodingUTF8,
			Text:     ep.Year,
		})
	}

	if len(ep.CoverArt) > 0 {
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    "image/jpeg",
			PictureType: id3v2.PTFrontCover,
			Description: "Cover",
			Picture:     ep.CoverArt,
		})
	}

	tag.AddCommentFrame(id3v2.CommentFrame{
		Encoding:    id3v2.EncodingUTF8,
		Language:    "eng",
		Description: "Downloaded via bible_audio_downloader",
		Text:        fmt.Sprintf("Source: %s", ep.PageURL),
	})

	return tag.Save()
}

func ValidateID3(path string) (state.ID3Validation, error) {
	v := state.ID3Validation{}
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return v, err
	}
	defer tag.Close()

	v.HasTitle = strings.TrimSpace(tag.Title()) != ""
	v.HasArtist = strings.TrimSpace(tag.Artist()) != ""
	v.HasAlbum = strings.TrimSpace(tag.Album()) != ""

	if frame := tag.GetTextFrame("TRCK"); strings.TrimSpace(frame.Text) != "" {
		v.HasTrack = true
	}

	pics := tag.GetFrames(tag.CommonID("Attached picture"))
	v.HasCover = len(pics) > 0

	return v, nil
}
