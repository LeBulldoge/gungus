package playback

import (
	"bufio"
	"context"
	"errors"
	"io"
	"iter"
	"log/slog"
	"os/exec"
	"slices"
	"sync"
	"time"

	"github.com/ClintonCollins/dca"
	"github.com/LeBulldoge/gungus/internal/os"
	"github.com/LeBulldoge/gungus/internal/youtube"
	"github.com/bwmarrin/discordgo"
)

var (
	ErrCauseStop       = errors.New("playback stopped")
	ErrCauseTimeout    = errors.New("playback timed out")
	ErrCauseSkip       = errors.New("playback skipped")
	ErrSkipUnavailable = errors.New("queue is empty")
)

type Player struct {
	mu sync.RWMutex

	vc      *discordgo.VoiceConnection
	running bool

	skipFunc context.CancelCauseFunc

	queue []youtube.Video

	logger *slog.Logger
}

func NewPlayer(vc *discordgo.VoiceConnection) *Player {
	return &Player{
		vc:    vc,
		queue: make([]youtube.Video, 0),
		logger: slog.Default().
			WithGroup("player").
			With("guildID", vc.GuildID, "channelID", vc.ChannelID),
	}
}

func (s *Player) Add(video youtube.Video) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return errors.New("playback service isn't running")
	}

	s.queue = append(s.queue, video)

	return nil
}

func (s *Player) Insert(video youtube.Video, index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return errors.New("playback service isn't running")
	}

	if index >= len(s.queue) {
		return errors.New("insert index is higher than queue length")
	}

	s.queue = slices.Insert(s.queue, index, video)

	return nil
}

func (s *Player) getNextVideo() iter.Seq[youtube.Video] {
	return func(yield func(video youtube.Video) bool) {
		for len(s.queue) > 0 {
			s.mu.Lock()
			video := s.queue[0]
			s.queue = s.queue[1:]
			s.mu.Unlock()
			if !yield(video) {
				return
			}
		}
	}
}

func (s *Player) waitForVideos(ctx context.Context) {
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

func (s *Player) Skip(cnt int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.skipFunc == nil {
		return errors.New("nothing to skip")
	}

	s.skipFunc(ErrCauseSkip)
	s.skipFunc = nil

	return nil
}

func (s *Player) Queue() []youtube.Video {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.queue
}

func (s *Player) Run(ctx context.Context, wg *sync.WaitGroup) error {
	if s.IsRunning() {
		return errors.New("player is already running")
	}

	s.setRunning(true)
	defer s.setRunning(false)

	wg.Done()
	s.waitForVideos(ctx)

	for video := range s.getNextVideo() {
		err := s.vc.Speaking(true)
		if err != nil {
			return err
		}

		skipCtx, skipFunc := context.WithCancelCause(ctx)

		s.mu.Lock()
		s.skipFunc = skipFunc
		s.mu.Unlock()

		s.logger.Info("currently playing", "guild", s.vc.GuildID, "video", video.Title)
		err = s.playAudioFromURL(skipCtx, video.URL, s.vc)
		if err != nil && !errors.Is(err, ErrCauseSkip) {
			return err
		}

		err = s.vc.Speaking(false)
		if err != nil {
			return err
		}
		s.logger.Info("done playing", "guild", s.vc.GuildID, "video", video.Title)
	}

	s.logger.Info("queue is empty", "guild", s.vc.GuildID)
	return nil
}

func (s *Player) setRunning(val bool) {
	s.mu.Lock()
	s.running = val
	s.mu.Unlock()
}

func (s *Player) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Player) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = nil
	return s.vc.Disconnect()
}

func (s *Player) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.queue)
}

func (s *Player) ChannelID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vc.ChannelID
}

func (s *Player) playAudioFromURL(ctx context.Context, url string, vc *discordgo.VoiceConnection) error {
	ytdlp := exec.Command(
		"yt-dlp",
		url,
		"--no-part",
		"--buffer-size", "16K",
		"--limit-rate", "50K",
		"-f", "bestaudio",
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
	options.Bitrate = 64
	options.Channels = 2
	options.Application = dca.AudioApplicationAudio
	options.VolumeFloat = -10.0
	options.VBR = true
	options.Threads = 0
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
			s.logger.Info("ytdlp stderr", "output", sc.Text())
		}
		if err := sc.Err(); err != nil {
			s.logger.Error("ytdlp stderr reader error", "err", err)
		}
	}()

	select {
	case <-ctx.Done():
		if err := session.Stop(); err != nil {
			s.logger.Error("failed to stop encoding session", "err", err)
			return err
		}
		if err := ytdlp.Process.Kill(); err != nil {
			s.logger.Error("failed to kill yt-dlp process", "err", err)
			return err
		}
		return context.Cause(ctx)
	case err := <-done:
		if err != nil {
			if err == io.EOF {
				s.logger.Info("playback finished")
				return nil
			}

			errBuf, _ := io.ReadAll(stderr)
			s.logger.Error("error occured while playing audio", "ffmpeg messages", session.FFMPEGMessages(), "ytdlp", errBuf)

			return err
		}
	}

	return err
}
