package main

import (
	"context"
	"eleven-hacks/internal/elevenlabs"
	"eleven-hacks/internal/mailtm"
	"fmt"
)

// App struct
type App struct {
	ctx    context.Context
	Bridge string
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

	return nil
}
