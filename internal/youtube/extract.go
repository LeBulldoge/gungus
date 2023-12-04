package youtube

import (
	"bufio"
	"errors"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/LeBulldoge/gungus/internal/os"
	"golang.org/x/net/context"
)

type YoutubeData struct {
	URL       string
	Title     string
	Thumbnail string
	Length    string
	ID        string
}

func (d YoutubeData) GetShortURL() string {
	return "https://youtu.be/" + d.ID
}

type YoutubeDataResult struct {
	Data  YoutubeData
	Error error
}

func SearchYoutube(ctx context.Context, query string, output chan<- YoutubeDataResult) error {
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

	res := YoutubeDataResult{}
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

			res.Data.URL = parts[0]
			res.Data.Title = parts[1]

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

func GetYoutubeData(ctx context.Context, videoURL string, output chan<- YoutubeDataResult) error {
	ytdlp := exec.Command(
		"yt-dlp",
		videoURL,
		"-f", "ba",
		"--get-title",
		"--get-id",
		"--get-url",
		"--get-thumbnail",
		"--get-duration",
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

		res := YoutubeDataResult{}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			res.Data.Title = scanner.Text()

			if scanner.Scan() {
				res.Data.ID = scanner.Text()
			}

			if scanner.Scan() {
				res.Data.URL = scanner.Text()
			}

			if scanner.Scan() {
				res.Data.Thumbnail = scanner.Text()
			}

			if scanner.Scan() {
				res.Data.Length = scanner.Text()
				if str := strings.Count(res.Data.Length, ":"); str < 1 {
					var sb strings.Builder
					sb.WriteString("00:")
					if len(res.Data.Length) < 2 {
						sb.WriteRune('0')
						sb.WriteString(res.Data.Length)
					}
					res.Data.Length = sb.String()
				}
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
