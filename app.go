package main

import (
	"context"
	"eleven-hacks/internal/config"
	"eleven-hacks/internal/elevenlabs"
	"eleven-hacks/internal/event"
	ffmpeghelper "eleven-hacks/internal/helper/ffmpeg-helper"
	"eleven-hacks/internal/mailtm"
	"fmt"
	"github.com/pkg/errors"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/exp/maps"
	"hash/crc32"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// App struct
type App struct {
	ctx          context.Context
	cancel       context.CancelFunc
	log          *slog.Logger
	logFile      *os.File
	dubbingFiles map[uint32]*elevenlabs.DubbingFile
	mx           *sync.RWMutex
	wg           sync.WaitGroup
	config       *config.Config
}

// NewApp creates a new App application struct
func NewApp() *App {
	config := config.NewConfig()
	config.Load()

	ffmpeg.GlobalCommandOptions = append(ffmpeg.GlobalCommandOptions, func(cmd *exec.Cmd) {
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		cmd.SysProcAttr.HideWindow = true
	})

	logFilePath := "log.txt"
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	return &App{
		log: slog.New(
			slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelError}),
		),
		logFile:      file,
		mx:           &sync.RWMutex{},
		config:       config,
		dubbingFiles: make(map[uint32]*elevenlabs.DubbingFile),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.ctx = context.WithValue(a.ctx, "assets", &assets)
	a.ctx = context.WithValue(a.ctx, "config", a.config)

	runtime.EventsOn(a.ctx, event.Error, func(optionalData ...interface{}) {
		if len(optionalData) > 0 {
			a.log.Error(optionalData[0].(string))
		}
	})
}

func (a *App) onBeforeClose(ctx context.Context) bool {
	a.cancel()
	a.wg.Wait()

	runtime.EventsOff(a.ctx, event.Error)

	a.logFile.Close()

	os.RemoveAll("tmp")

	return false
}

func (a *App) UpdateBridge(bridge string) {
	a.config.SetField("Bridge", bridge)
}

func (a *App) GetTorPath() string {
	if a.config.TorPath == "" {
		return ""
	}

	return strings.TrimSuffix(a.config.TorPath, "Browser/TorBrowser/Tor/tor.exe")
}

func (a *App) GetDubbingFiles() []*elevenlabs.DubbingFile {
	a.mx.RLock()
	defer a.mx.RUnlock()
	return maps.Values(a.dubbingFiles)
}

func (a *App) SetTorPath() (string, error) {
	var err error

	defer func() {
		if err != nil {
			runtime.EventsEmit(a.ctx, event.Error, err.Error())
		} else {
			runtime.EventsEmit(a.ctx, event.Info, "Successfully found the Tor browser.")
		}
	}()

	torPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose Tor browser folder",
	})
	if err != nil {
		return "", err
	} else if torPath == "" {
		err = errors.New("Tor path is not specified.")
		return "", err
	}

	torExePath := filepath.Join(torPath, "Browser/TorBrowser/Tor/tor.exe")

	_, err = os.Open(torExePath)
	if os.IsNotExist(err) {
		return "", errors.New("tor.exe not found.")
	}

	lyrebirdExePath := filepath.Join(torPath, "Browser/TorBrowser/Tor/PluggableTransports/lyrebird.exe")

	_, err = os.Open(lyrebirdExePath)
	if os.IsNotExist(err) {
		return "", errors.New("lyrebird.exe not found.")
	}

	a.config.SetField("TorPath", torExePath)
	a.config.SetField("LyrebirdPath", lyrebirdExePath)

	return torPath, nil
}

func (a *App) GetLanguages() map[string]string {
	return elevenlabs.GetLanguages()
}

func (a *App) GetSavePath() string {
	return a.config.DubbingSavePath
}

func (a *App) SetSavePath() (string, error) {
	var err error

	defer func() {
		if err != nil {
			runtime.EventsEmit(a.ctx, event.Error, err.Error())
		}
	}()

	savePath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose dubbing save folder",
	})
	if err != nil {
		return "", err
	} else if savePath == "" {
		err = errors.New("Save path is not specified.")
		return "", err
	}

	a.config.SetField("DubbingSavePath", savePath)

	return savePath, nil
}

func (a *App) ChooseFiles() ([]string, error) {
	var err error

	defer func() {
		if err != nil {
			runtime.EventsEmit(a.ctx, event.Error, err.Error())
		}
	}()

	filePaths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose files for dubbing",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Videos (*.mp4,*.webm)",
				Pattern:     "*.mp4;*.webm",
			},
		},
	})
	if err != nil {
		return nil, err
	} else if len(filePaths) == 0 {
		err = errors.New("File(-s) path is not specified")
		return nil, err
	}

	return filePaths, nil
}

func (a *App) AddDubbingFile(captchaToken string, filePath string) error {
	timestamp := time.Now().Unix()
	hashString := filePath + strconv.FormatInt(timestamp, 10)
	hash := crc32.ChecksumIEEE([]byte(hashString))

	a.mx.Lock()
	a.dubbingFiles[hash] = &elevenlabs.DubbingFile{
		Status: elevenlabs.StatusAccount,
		Path:   filePath,
		Name:   filepath.Base(filePath),
	}
	a.mx.Unlock()
	runtime.EventsEmit(a.ctx, event.DubbingUpdate)

	a.wg.Add(1)
	defer a.wg.Done()
	apiKey, err := a.RegisterAndConfirmAccount(captchaToken)
	if err != nil {
		a.mx.Lock()
		delete(a.dubbingFiles, hash)
		a.mx.Unlock()

		runtime.EventsEmit(a.ctx, event.DubbingUpdate)
		runtime.EventsEmit(a.ctx, event.Error, fmt.Sprintf("Failed to create an account for dubbing file %s", filePath))

		return err
	}

	a.mx.Lock()
	a.dubbingFiles[hash].Status = elevenlabs.StatusAdded
	a.dubbingFiles[hash].ApiKey = apiKey
	a.mx.Unlock()

	runtime.EventsEmit(a.ctx, event.DubbingUpdate)

	return nil
}

func (a *App) StartDubbing(srcLang, targetLang string) error {
	var err error

	defer func() {
		if err != nil {
			runtime.EventsEmit(a.ctx, event.Error, err.Error())
		}
	}()

	if a.config.TorPath == "" {
		err = errors.New("Please specify the Tor path first.")
		return err
	}

	if a.config.DubbingSavePath == "" {
		err = errors.New("Please specify a save folder first.")
		return err
	}

	if len(a.dubbingFiles) == 0 {
		err = errors.New("Please select a video(-s) for dubbing first.")
		return err
	}

	dp := &elevenlabs.DubbingParams{
		MaxTry:     10,
		Interval:   10,
		SavePath:   a.config.DubbingSavePath,
		Bridge:     a.config.Bridge,
		SourceLang: srcLang,
		TargetLang: targetLang,
	}

	for key, dubbingFile := range a.dubbingFiles {
		if dubbingFile.Status != elevenlabs.StatusAdded && dubbingFile.Status != elevenlabs.StatusError {
			continue
		}

		a.wg.Add(1)

		go func(key uint32) {
			defer a.wg.Done()
			err := elevenlabs.WaitForDubbedFileAndSave(a.ctx, a.dubbingFiles[key], dp)
			if err == nil {
				a.mx.Lock()
				delete(a.dubbingFiles, key)
				a.mx.Unlock()
			} else {
				runtime.EventsEmit(a.ctx, event.Error, err.Error())
			}

			runtime.EventsEmit(a.ctx, event.DubbingUpdate)
		}(key)
	}

	return nil
}

func (a *App) RegisterAndConfirmAccount(captcha string) (*elevenlabs.ApiKeyResponse, error) {
	var err error

	defer func() {
		if err != nil {
			runtime.EventsEmit(a.ctx, event.Error, err.Error())
		}
	}()

	mail, err := mailtm.NewMailTM()
	if err != nil {
		return nil, err
	}

	mailAccount, err := mail.NewAccount()
	if err != nil {
		return nil, err
	}
	defer mail.DeleteAccount(mailAccount)

	el, err := elevenlabs.New(a.ctx, true)
	if err != nil {
		return nil, err
	}
	defer el.Proxy.Close()

	err = el.Register(mailAccount.Address, mailAccount.Password, captcha)
	if err != nil {
		return nil, err
	}

	message, err := mail.WaitForConfirmationEmail(mailAccount, 20)
	if err != nil {
		return nil, err
	}

	url, err := mail.GetConfirmationUrl(message.Html[0])
	if err != nil {
		return nil, err
	}

	confirmationData, err := mail.GetConfirmationData(url)
	if err != nil {
		return nil, err
	}

	err = el.UpdateAccount(confirmationData.OobCode)
	if err != nil {
		return nil, err
	}

	err = el.PrepareInternalVerification(mailAccount.Address, confirmationData.InternalCode)
	if err != nil {
		return nil, err
	}

	signInData, err := el.SignIn(mailAccount.Address, mailAccount.Password)
	if err != nil {
		return nil, err
	}

	apiKey, err := el.CreateApiKey(signInData.Token)
	if err != nil {
		return nil, err
	}

	return apiKey, nil
}

func (a *App) SplitVideo(duration int) error {
	var err error

	defer func() {
		if err != nil {
			runtime.EventsEmit(a.ctx, event.Error, err.Error())
		}
	}()

	if ok := ffmpeghelper.IsFfmpegAvailable(); !ok {
		return errors.New("Ffmpeg not found in environment.")
	}

	if ok := ffmpeghelper.IsFfprobeAvailable(); !ok {
		return errors.New("Ffprobe not found in environment.")
	}

	savePath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose video save folder",
	})
	if err != nil {
		return err
	} else if savePath == "" {
		err = errors.New("Save path is not specified.")
		return err
	}

	filePaths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose files to split",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Videos (*.mp4,*.webm)",
				Pattern:     "*.mp4;*.webm",
			},
		},
	})
	if err != nil {
		return err
	} else if len(filePaths) == 0 {
		err = errors.New("File(-s) path is not specified.")
		return err
	}

	for _, filePath := range filePaths {
		go func(filePath string) {
			err := ffmpeghelper.SplitVideo(a.ctx, filePath, savePath, duration)
			if err != nil {
				runtime.EventsEmit(a.ctx, event.Error, err.Error())
			}
		}(filePath)
	}

	return nil
}
