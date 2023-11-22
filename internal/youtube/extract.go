package youtube

import (
	"bufio"
	"os/exec"

	"golang.org/x/exp/slog"
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

func GetYoutubeData(ctx context.Context, videoUrl string, output chan<- YoutubeDataResult) error {
	ytdlp := exec.Command(
		"yt-dlp",
		videoUrl,
		"-f", "ba",
		"--get-url",
		"--get-title",
		"--get-thumbnail",
		"--get-duration",
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