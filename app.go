package main

import (
	"context"
	"eleven-hacks/internal/elevenlabs"
	"eleven-hacks/internal/mailtm"
	"errors"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"sync"
)

// App struct
type App struct {
	ctx          context.Context
	bridge       string
	savePath     string
	dubbingFiles []DubbingFile
	mx           *sync.RWMutex
}

type DubbingFile struct {
	Path   string
	ApiKey *elevenlabs.ApiKeyResponse
}

type Token struct {
	FilePath string
	Token    string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		mx: &sync.RWMutex{},
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) UpdateBridge(bridge string) {
	a.bridge = bridge
}

func (a *App) GetLanguages() map[string]string {
	return elevenlabs.GetLanguages()
}

func (a *App) SetSavePath() (string, error) {
	savePath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose dubbing save folder",
	})
	if err != nil {
		return "", err
	} else if savePath == "" {
		runtime.EventsEmit(a.ctx, "LOG", "Save path is not specified.")
		return "", errors.New("save path is not specified")
	}

	a.savePath = savePath

	return savePath, nil
}

func (a *App) ChooseFiles() ([]string, error) {
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
		runtime.EventsEmit(a.ctx, "LOG", "File path is not specified.")
		return nil, errors.New("file path is not specified")
	}

	return filePaths, nil
}

func (a *App) AddDubbingFile(token Token) error {
	apiKey, err := a.RegisterAndConfirmAccount(token.Token)
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", fmt.Sprintf("Failed to create an account for dubbing file %s", token.FilePath))
		return err
	}

	a.mx.Lock()
	defer a.mx.Unlock()
	a.dubbingFiles = append(a.dubbingFiles, DubbingFile{token.FilePath, apiKey})
	return nil
}

func (a *App) StartDubbing(srcLang, targetLang string) error {
	if a.savePath == "" {
		err := errors.New("Please select a save folder first.")
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return err
	}

	if len(a.dubbingFiles) == 0 {
		err := errors.New("Please select a video(-s) for dubbing first.")
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return err
	}

	a.mx.RLock()
	dubbingFiles := a.dubbingFiles
	a.mx.RUnlock()

	a.mx.Lock()
	a.dubbingFiles = nil
	a.mx.Unlock()

	for _, dubbingFile := range dubbingFiles {
		go func(dubbingFile DubbingFile) {
			err := elevenlabs.NewElevenLabs().WaitForDubbedFileAndSave(
				a.ctx,
				120,
				10,
				dubbingFile.Path,
				a.savePath,
				a.bridge,
				dubbingFile.ApiKey,
			)
			if err != nil {
				a.mx.Lock()
				defer a.mx.Unlock()
				a.dubbingFiles = append(a.dubbingFiles, dubbingFile)
			}
		}(dubbingFile)
	}

	return nil
}

func (a *App) RegisterAndConfirmAccount(captcha string) (*elevenlabs.ApiKeyResponse, error) {
	runtime.EventsEmit(a.ctx, "LOG", "Trying to create and confirm a new account...")

	mail, err := mailtm.NewMailTM()
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}

	mailAccount, err := mail.NewAccount()
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}
	defer mail.DeleteAccount(mailAccount)

	el := elevenlabs.NewElevenLabs()
	err = el.Register(mailAccount.Address, mailAccount.Password, captcha)
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}

	message, err := mail.WaitForConfirmationEmail(mailAccount, 20)
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}

	url, err := mail.GetConfirmationUrl(message.Html[0])
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}

	confirmationData, err := mail.GetConfirmationData(url)
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}

	fmt.Println(confirmationData)

	err = el.UpdateAccount(confirmationData.OobCode)
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}

	err = el.PrepareInternalVerification(mailAccount.Address, confirmationData.InternalCode)
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}

	signInData, err := el.SignIn(mailAccount.Address, mailAccount.Password)
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}

	apiKey, err := el.CreateApiKey(signInData.Token)
	if err != nil {
		runtime.EventsEmit(a.ctx, "LOG", err.Error())
		return nil, err
	}

	runtime.EventsEmit(a.ctx, "LOG", "Account created successfully. Your API key is: "+apiKey.ApiKey)

	return apiKey, nil
}
