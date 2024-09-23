package main

import (
	"context"
	"eleven-hacks/internal/elevenlabs"
	"eleven-hacks/internal/mailtm"
	"errors"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	Bridge string
	ApiKey *elevenlabs.ApiKeyResponse
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) UpdateBridge(bridge string) {
	a.Bridge = bridge
}

func (a *App) RegisterAndConfirmAccount(captcha string) error {
	mail, err := mailtm.NewMailTM()
	if err != nil {
		fmt.Println(err)
		return err
	}

	mailAccount, err := mail.NewAccount()
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer mail.DeleteAccount(mailAccount)

	el := elevenlabs.NewElevenLabs()
	err = el.Register(mailAccount.Address, mailAccount.Password, captcha)
	if err != nil {
		fmt.Println(err)
		return err
	}

	message, err := mail.WaitForConfirmationEmail(mailAccount, 20)
	if err != nil {
		fmt.Println(err)
		return err
	}

	url, err := mail.GetConfirmationUrl(message.Html[0])
	if err != nil {
		fmt.Println(err)
		return err
	}

	confirmationData, err := mail.GetConfirmationData(url)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println(confirmationData)

	err = el.UpdateAccount(confirmationData.OobCode)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = el.PrepareInternalVerification(mailAccount.Address, confirmationData.InternalCode)
	if err != nil {
		fmt.Println(err)
		return err
	}

	signInData, err := el.SignIn(mailAccount.Address, mailAccount.Password)
	if err != nil {
		fmt.Println(err)
		return err
	}

	apiKey, err := el.CreateApiKey(signInData.Token)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println(apiKey)
	a.ApiKey = apiKey

	return nil
}

func (a *App) CreateDubbing() error {
	if a.ApiKey == nil {
		runtime.EventsEmit(a.ctx, "LOG", "You need to create an account first before dubbing.")
		return errors.New("you need to create an account first before dubbing")
	}

	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose file for dubbing",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Videos (*.mp4,*.webm)",
				Pattern:     "*.mp4;*.webm",
			},
		},
	})
	if err != nil {
		return err
	} else if filePath == "" {
		runtime.EventsEmit(a.ctx, "LOG", "File path is not specified.")
		return errors.New("file path is not specified")
	}

	savePath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose dubbing save folder",
	})
	if err != nil {
		return err
	} else if savePath == "" {
		runtime.EventsEmit(a.ctx, "LOG", "Save path is not specified.")
		return errors.New("save path is not specified")
	}

	err = elevenlabs.NewElevenLabs().WaitForDubbedFileAndSave(
		a.ctx,
		120,
		10,
		filePath,
		savePath,
		a.Bridge,
		a.ApiKey,
	)
	if err != nil {
		return err
	}

	a.ApiKey = nil

	return nil
}
