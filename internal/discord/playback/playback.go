package playback

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"

	"github.com/ClintonCollins/dca"
	"github.com/LeBulldoge/gungus/internal/youtube"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/exp/slog"
)

type PlaybackServiceManager struct {
	sync.RWMutex

	services map[string]*PlaybackService
}

func NewManager() PlaybackServiceManager {
	return PlaybackServiceManager{
		services: make(map[string]*PlaybackService),
	}
}

func (m *PlaybackServiceManager) Get(guildId string) *PlaybackService {
	m.RLock()
	defer m.RUnlock()

	var res *PlaybackService
	if ps, ok := m.services[guildId]; ok {
		res = ps
	}

	return res
}

func (m *PlaybackServiceManager) Add(guildId string, ps *PlaybackService) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.services[guildId]; ok {
		return errors.New("playback service already exists")
	}

	m.services[guildId] = ps

	return nil
}

func (m *PlaybackServiceManager) Delete(guildId string) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.services[guildId]; !ok {
		return errors.New("playback service does not exist")
	} else {
		delete(m.services, guildId)
	}

	return nil
}

type PlaybackService struct {
	sync.RWMutex

	queue   chan youtube.YoutubeData
	vc      *discordgo.VoiceConnection
	running bool
}

func NewPlaybackService(vc *discordgo.VoiceConnection) *PlaybackService {
	return &PlaybackService{
		vc:    vc,
		queue: make(chan youtube.YoutubeData, 50),
	}
}

func (s *PlaybackService) EnqueueVideo(video youtube.YoutubeData) error {
	if !s.IsRunning() {
		return errors.New("playback service isn't running")
	}
	s.queue <- video
	return nil
}

func (s *PlaybackService) Run(ctx context.Context, wg *sync.WaitGroup) error {
	if s.IsRunning() {
		return errors.New("PlaybackService: already running")
	}

	s.setRunning(true)
	defer s.setRunning(false)

	wg.Done()

	for video := range s.queue {
		slog.Info("PlaybackService: received video", "guild", s.vc.GuildID, "video", video.Title)

		s.Lock()
		err := s.vc.Speaking(true)
		s.Unlock()
		if err != nil {
			return err
		}

		slog.Info("PlaybackService: currently speaking", "guild", s.vc.GuildID, "video", video.Title)
		err = playAudioFromUrl(ctx, video.Url, s.vc)
		if err != nil {
			return err
		}

		s.Lock()
		err = s.vc.Speaking(false)
		s.Unlock()
		if err != nil {
			return err
		}
		slog.Info("PlaybackService: done streaming", "guild", s.vc.GuildID, "video", video.Title)

		if len(s.queue) == 0 {
			slog.Info("PlaybackService: queue is empty", "guild", s.vc.GuildID)
			return nil
		}
	}

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
	// wait for channel to be clear and close
	if len(s.queue) > 0 {
		<-s.queue
	}
	close(s.queue)

	return s.vc.Disconnect()
}

func (s *PlaybackService) Count() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.queue)
}

func playAudioFromUrl(ctx context.Context, url string, vc *discordgo.VoiceConnection) error {
	ytdlp := exec.Command(
		"yt-dlp",
		url,
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
		return errors.New("playback canceled")
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
