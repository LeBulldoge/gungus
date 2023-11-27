package youtube

import (
	"bufio"
	"log/slog"
	"os/exec"

	"github.com/LeBulldoge/gungus/internal/os"
	"golang.org/x/net/context"
)

type YoutubeData struct {
	Url       string
	Title     string
	Thumbnail string
	Length    string
}

type YoutubeDataResult struct {
	Data  YoutubeData
	Error error
}

func SearchYoutube(ctx context.Context, query string, output chan<- YoutubeDataResult) error {
	ytdlp := exec.Command(
		"yt-dlp",
		"ytsearch5:"+query,
		"--get-url",
		"--get-title",
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

	go func() {
		res := YoutubeDataResult{}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			res.Data.Title = scanner.Text()

			if scanner.Scan() {
				res.Data.Url = scanner.Text()
			}

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

func GetYoutubeData(ctx context.Context, videoUrl string, output chan<- YoutubeDataResult) error {
	ytdlp := exec.Command(
		"yt-dlp",
		videoUrl,
		"-f", "ba",
		"--get-url",
		"--get-title",
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
				res.Data.Url = scanner.Text()
			}

			if scanner.Scan() {
				res.Data.Thumbnail = scanner.Text()
			}

			if scanner.Scan() {
				res.Data.Length = scanner.Text()
			}

			select {
			case <-ctx.Done():
				slog.Info("GetYoutubeData canceled via context", "videoUrl", videoUrl)
				return
			case output <- res:
			}
		}
		slog.Info("GetYoutubeData finished", "videoUrl", videoUrl)

		if err := ytdlp.Wait(); err != nil {
			res.Error = err
			select {
			case <-ctx.Done():
				slog.Info("GetYoutubeData canceled via context", "videoUrl", videoUrl)
				return
			case output <- res:
			}
		}
	}()

	return nil
}
