package elevenlabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

type ElevenLabs struct {
}

type PreSignUpRequest struct {
	AccountMetaData AccountMetaData `json:"account_metadata"`
	Email           string          `json:"email"`
	RecaptchaToken  string          `json:"recaptcha_token"`
}

type AccountMetaData struct {
	AgreesToProductUpdates bool        `json:"agrees_to_product_updates"`
	GeoLocation            GeoLocation `json:"geo_location"`
}

type GeoLocation struct {
	City    string `json:"city"`
	Country string `json:"country"`
	Region  string `json:"region"`
}

type AccountSignUpRequest struct {
	ClientType        string `json:"clientType"`
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type AccountUpdateRequest struct {
	OobCode string `json:"oobCode"`
}

type InternalVerificationRequest struct {
	Email            string `json:"email"`
	VerificationCode string `json:"verification_code"`
}

type EmailVerificationRequest struct {
	Email string `json:"email"`
}

type ApiKeyRequest struct {
	Name string `json:"name"`
}

type ApiKeyResponse struct {
	ApiKey string `json:"xi_api_key"`
}

type SignInRequest struct {
	ClientType        string `json:"clientType"`
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type SignInResponse struct {
	Token string `json:"idToken"`
}

const (
	GoogleApiKey                   = "AIzaSyBSsRE_1Os04-bxpd5JTLIniy3UK4OqKys"
	PreSignUpUrl                   = "https://api.elevenlabs.io/v1/user/pre-sign-up"
	SendVerificationEmailUrl       = "https://api.elevenlabs.io/v1/user/send-verification-email"
	PrepareInternalVerificationUrl = "https://api.elevenlabs.io/v1/user/prepare-internal-verification"
	AccountSignUpUrl               = "https://identitytoolkit.googleapis.com/v1/accounts:signUp"
	AccountSignInUrl               = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword"
	AccountUpdateUrl               = "https://identitytoolkit.googleapis.com/v1/accounts:update"
	CreateApiKeyUrl                = "https://api.elevenlabs.io/v1/user/create-api-key"
)

func NewElevenLabs() *ElevenLabs {
	return &ElevenLabs{}
}

func (el *ElevenLabs) Register(email, password, captcha string) error {
	err := el.PreSignUp(email, captcha)
	if err != nil {
		return err
	}
	err = el.SignUp(email, password)
	if err != nil {
		return err
	}
	err = el.SendVerificationEmail(email)
	if err != nil {
		return err
	}

	return nil
}

func (el *ElevenLabs) PreSignUp(email, captcha string) error {
	data := &PreSignUpRequest{
		AccountMetaData: AccountMetaData{
			AgreesToProductUpdates: false,
			GeoLocation: GeoLocation{
				City:    "?",
				Country: "US",
				Region:  "?",
			},
		},
		Email:          email,
		RecaptchaToken: captcha,
	}
	dataJson, _ := json.Marshal(data)

	req, err := http.NewRequest(http.MethodPost, PreSignUpUrl, bytes.NewBuffer(dataJson))
	if err != nil {
		return errors.WithMessage(err, "Unable to create new pre-sign-up request")
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithMessage(err, "Unable to execute pre-sign-up request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return errors.Errorf("Pre-sign-up request responded with status code %d and body %s", res.StatusCode, string(body))
	}

	return nil
}

func (el *ElevenLabs) SendVerificationEmail(email string) error {
	data := &EmailVerificationRequest{
		Email: email,
	}
	dataJson, _ := json.Marshal(data)

	req, err := http.NewRequest(http.MethodPost, SendVerificationEmailUrl, bytes.NewBuffer(dataJson))
	if err != nil {
		return errors.WithMessage(err, "Unable to create new email verification request")
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithMessage(err, "Unable to execute email verification request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return errors.Errorf("Email verification request responded with status code %d and body %s", res.StatusCode, string(body))
	}

	return nil
}

func (el *ElevenLabs) SignUp(email, password string) error {
	data := &AccountSignUpRequest{
		ClientType:        "CLIENT_TYPE_WEB",
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	}
	dataJson, _ := json.Marshal(data)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s?key=%s", AccountSignUpUrl, GoogleApiKey), bytes.NewBuffer(dataJson))
	if err != nil {
		return errors.WithMessage(err, "Unable to create new account sign-up request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://elevenlabs.io/")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithMessage(err, "Unable to execute account sign-up request")
	}
	defer res.Body.Close()

	// Usually response status code will be 400, because tmp emails can't be verified by Google
	// but ElevenLabs just ignores it
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(res.Body)
		return errors.Errorf(
			"Account sign-up request responded with status code %d and body %s\nURL: %s",
			res.StatusCode,
			string(body),
			req.URL,
		)
	}

	return nil
}

func (el *ElevenLabs) SignIn(email, password string) (*SignInResponse, error) {
	data := &SignInRequest{
		ClientType:        "CLIENT_TYPE_WEB",
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	}
	dataJson, _ := json.Marshal(data)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s?key=%s", AccountSignInUrl, GoogleApiKey), bytes.NewBuffer(dataJson))
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to create new sign-in request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://elevenlabs.io/")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to execute sign-in request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, errors.Errorf("Sign-in request responded with status code %d and body %s", res.StatusCode, string(body))
	}

	resData := &SignInResponse{}
	err = json.NewDecoder(res.Body).Decode(&resData)
	if err != nil {
		return nil, err
	}

	return resData, nil
}

func (el *ElevenLabs) UpdateAccount(oobCode string) error {
	data := &AccountUpdateRequest{
		OobCode: oobCode,
	}
	dataJson, _ := json.Marshal(data)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s?key=%s", AccountUpdateUrl, GoogleApiKey), bytes.NewBuffer(dataJson))
	if err != nil {
		return errors.WithMessage(err, "Unable to create new account update request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://elevenlabs.io/")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithMessage(err, "Unable to execute account update request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return errors.Errorf("Account update request responded with status code %d and body %s", res.StatusCode, string(body))
	}

	return nil
}

func (el *ElevenLabs) PrepareInternalVerification(email, verificationCode string) error {
	data := &InternalVerificationRequest{
		Email:            email,
		VerificationCode: verificationCode,
	}
	dataJson, _ := json.Marshal(data)

	req, err := http.NewRequest(http.MethodPost, PrepareInternalVerificationUrl, bytes.NewBuffer(dataJson))
	if err != nil {
		return errors.WithMessage(err, "Unable to create new prepare internal verification request")
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithMessage(err, "Unable to execute prepare internal verification request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return errors.Errorf("Prepare internal verification request responded with status code %d and body %s", res.StatusCode, string(body))
	}

	return nil
}

func (el *ElevenLabs) CreateApiKey(token string) (*ApiKeyResponse, error) {
	data := &ApiKeyRequest{
		Name: "ApiKey",
	}
	dataJson, _ := json.Marshal(data)

	req, err := http.NewRequest(http.MethodPost, CreateApiKeyUrl, bytes.NewBuffer(dataJson))
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to create new create api key request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to execute create api key request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, errors.Errorf("Create api key request responded with status code %d and body %s", res.StatusCode, string(body))
	}

	resData := &ApiKeyResponse{}
	err = json.NewDecoder(res.Body).Decode(&resData)
	if err != nil {
		return nil, err
	}

	return resData, nil
}
