package youtube

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/LeBulldoge/gungus/internal/os"
)

type Video struct {
	URL       string
	Title     string
	Thumbnail string
	Length    string
	ID        string
}

func (d Video) GetShortURL() string {
	return "https://youtu.be/" + d.ID
}

type SearchResult struct {
	Video Video
	Error error
}

func SearchYoutube(ctx context.Context, query string, output chan<- SearchResult) error {
	ytdlp := exec.Command(
		"yt-dlp",
		"ytsearch5:"+query,
		"--print", "%(url)s;%(title)s",
		"--flat-playlist",
		"--lazy-playlist",
		"--ies", "youtube:search",
		"--cache-dir", os.CachePath("ytdlp"),
	)

	stdout, err := ytdlp.StdoutPipe()
	if err != nil {
		return err
	}

	if err := ytdlp.Start(); err != nil {
		return err
	}

	res := SearchResult{}
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			parts := strings.Split(scanner.Text(), ";")
			if len(parts) < 2 {
				res.Error = errors.New("failed parsing youtube result: " + scanner.Text())
				select {
				case <-ctx.Done():
					slog.Info("SearchYoutube canceled via context", "query", query)
					close(output)
					return
				case output <- res:
				}
				continue
			}

			res.Video.URL = parts[0]
			res.Video.Title = parts[1]

			select {
			case <-ctx.Done():
				slog.Info("SearchYoutube canceled via context", "query", query)
				close(output)
				return
			case output <- res:
			}
		}
		slog.Info("SearchYoutube finished", "query", query)

		close(output)

		if err := ytdlp.Wait(); err != nil {
			slog.Error("SearchYoutube error on wait", "err", err)
			return
		}
	}()

	return nil
}

func GetYoutubeData(ctx context.Context, videoURL string, output chan<- SearchResult) error {
	ytdlp := exec.Command(
		"yt-dlp",
		videoURL,
		"--get-title",
		"--get-id",
		"--get-duration",
		"--flat-playlist",
		"--no-playlist",
		"--cache-dir", os.CachePath("ytdlp"),
	)

	stdout, err := ytdlp.StdoutPipe()
	if err != nil {
		return err
	}

	if err := ytdlp.Start(); err != nil {
		return err
	}

	go func() {
		defer close(output)

		res := SearchResult{}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			res.Video.Title = scanner.Text()

			if scanner.Scan() {
				res.Video.ID = scanner.Text()
			}

			if scanner.Scan() {
				res.Video.Length = scanner.Text()
			}

			select {
			case <-ctx.Done():
				slog.Info("GetYoutubeData canceled via context", "videoUrl", videoURL)
				return
			case output <- res:
			}
		}
		slog.Info("GetYoutubeData finished", "videoUrl", videoURL)

		if err := ytdlp.Wait(); err != nil {
			res.Error = err
			select {
			case <-ctx.Done():
				slog.Info("GetYoutubeData canceled via context", "videoUrl", videoURL)
				return
			case output <- res:
			}
		}
	}()

	return nil
}
