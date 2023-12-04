package playback

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/ClintonCollins/dca"
	"github.com/LeBulldoge/gungus/internal/os"
	"github.com/LeBulldoge/gungus/internal/youtube"
	"github.com/bwmarrin/discordgo"
)

var (
	ErrCauseStop       = errors.New("playback stopped")
	ErrCauseSkip       = errors.New("playback skipped")
	ErrSkipUnavailable = errors.New("queue is empty")
)

type PlaybackService struct {
	sync.RWMutex

	vc      *discordgo.VoiceConnection
	running bool

	skipFunc context.CancelCauseFunc

	head  int
	queue []youtube.YoutubeData
}

func NewPlaybackService(vc *discordgo.VoiceConnection) *PlaybackService {
	return &PlaybackService{
		vc:    vc,
		queue: make([]youtube.YoutubeData, 0),
		head:  -1,
	}
}

func (s *PlaybackService) EnqueueVideo(video youtube.YoutubeData) error {
	s.Lock()
	defer s.Unlock()
	if !s.running {
		return errors.New("playback service isn't running")
	}

	s.queue = append(s.queue, video)

	return nil
}

func (s *PlaybackService) getNextVideo() youtube.YoutubeData {
	s.Lock()
	defer s.Unlock()

	video := s.queue[s.head]

	return video
}

func (s *PlaybackService) nextVideo() bool {
	s.Lock()
	defer s.Unlock()

	s.head++
	return s.head < len(s.queue)
}

func (s *PlaybackService) waitForVideos(ctx context.Context) {
	for {
		if s.Count() > 0 {
			return
		}

		t := time.After(time.Second)
		select {
		case <-ctx.Done():
			return
		case <-t:
		}
	}
}

func (s *PlaybackService) Skip(cnt int) error {
	s.Lock()
	defer s.Unlock()
	if s.skipFunc == nil {
		return errors.New("nothing to skip")
	}

	s.skipFunc(ErrCauseSkip)
	s.skipFunc = nil

	s.head += (cnt - 1)

	return nil
}

func (s *PlaybackService) Queue() []youtube.YoutubeData {
	s.RLock()
	defer s.RUnlock()

	return s.queue[s.head:]
}

func (s *PlaybackService) Run(ctx context.Context, wg *sync.WaitGroup) error {
	if s.IsRunning() {
		return errors.New("playback service is already running")
	}

	s.setRunning(true)
	defer s.setRunning(false)

	wg.Done()
	s.waitForVideos(ctx)

	for s.nextVideo() {
		video := s.getNextVideo()

		s.Lock()
		err := s.vc.Speaking(true)
		s.Unlock()
		if err != nil {
			return err
		}

		skipCtx, skipFunc := context.WithCancelCause(ctx)

		s.Lock()
		s.skipFunc = skipFunc
		s.Unlock()

		slog.Info("PlaybackService: currently playing", "guild", s.vc.GuildID, "video", video.Title)
		err = playAudioFromURL(skipCtx, video.URL, s.vc)
		if err != nil && !errors.Is(err, ErrCauseSkip) {
			return err
		}

		s.Lock()
		err = s.vc.Speaking(false)
		s.Unlock()
		if err != nil {
			return err
		}
		slog.Info("PlaybackService: done playing", "guild", s.vc.GuildID, "video", video.Title)
	}

	slog.Info("PlaybackService: queue is empty", "guild", s.vc.GuildID)
	return nil
}

func (s *PlaybackService) setRunning(val bool) {
	s.Lock()
	s.running = val
	s.Unlock()
}

func (s *PlaybackService) IsRunning() bool {
	s.RLock()
	defer s.RUnlock()
	return s.running
}

func (s *PlaybackService) Cleanup() error {
	s.Lock()
	defer s.Unlock()
	return s.vc.Disconnect()
}

func (s *PlaybackService) Count() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.queue)
}

func (s *PlaybackService) ChannelID() string {
	s.RLock()
	defer s.RUnlock()
	return s.vc.ChannelID
}

func playAudioFromURL(ctx context.Context, url string, vc *discordgo.VoiceConnection) error {
	ytdlp := exec.Command(
		"yt-dlp",
		url,
		"--cache-dir", os.CachePath("ytdlp"),
		"-o", "-",
	)

	stdout, err := ytdlp.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := ytdlp.StderrPipe()
	if err != nil {
		return err
	}

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 128
	options.Channels = 2
	options.Application = dca.AudioApplicationLowDelay
	options.VolumeFloat = -10.0
	options.VBR = true
	options.Threads = 0
	options.BufferedFrames = 512
	options.PacketLoss = 0

	session, err := dca.EncodeMem(stdout, options)
	if err != nil {
		return err
	}
	defer session.Cleanup()

	done := make(chan error)
	dca.NewStream(session, vc, done)

	err = ytdlp.Start()
	if err != nil {
		return err
	}
	defer ytdlp.Wait()

	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			slog.Info("ytdlp stderr", "output", sc.Text())
		}
		if err := sc.Err(); err != nil {
			slog.Error("ytdlp stderr reader error", "err", err)
		}
	}()

	select {
	case <-ctx.Done():
		if err := session.Stop(); err != nil {
			slog.Error("PlaybackService: failed to stop encoding session", "err", err)
			return err
		}
		if err := ytdlp.Process.Kill(); err != nil {
			slog.Error("PlaybackService: failed to kill yt-dlp process", "err", err)
			return err
		}
		return context.Cause(ctx)
	case err := <-done:
		if err != nil {
			if err == io.EOF {
				slog.Info("PlaybackService: playback finished")
				return nil
			}

			errBuf, _ := io.ReadAll(stderr)
			slog.Error("PlaybackService: error occured while playing audio", "ffmpeg messages", session.FFMPEGMessages(), "ytdlp", errBuf)

			return err
		}
	}

	return err
}
