package elevenlabs

import (
	"bytes"
	"context"
	"eleven-hacks/internal/config"
	"eleven-hacks/internal/torproxy"
	"eleven-hacks/pkg/multiparthelper"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ElevenLabs struct {
	config *config.Config
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

type CreateDubbingResponse struct {
	DubbingId        string  `json:"dubbing_id"`
	ExpectedDuration float64 `json:"expected_duration_sec"`
}

type GetDubbingDataResponse struct {
	DubbingId       string   `json:"dubbing_id"`
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	TargetLanguages []string `json:"target_languages"`
	Err             string   `json:"error"`
}

type CreateDubbingErrorResponse struct {
	Detail struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"detail"`
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
	CreateDubbingUrl               = "https://api.elevenlabs.io/v1/dubbing"
	GetDubbingDataUrl              = "https://api.elevenlabs.io/v1/dubbing"
	GetDubbedFileUrl               = "https://api.elevenlabs.io/v1/dubbing/%s/audio/%s"
)

var Languages = map[string]string{
	"eng": "English",
	"hi":  "Hindi",
	"pt":  "Portuguese",
	"zh":  "Chinese",
	"es":  "Spanish",
	"fr":  "French",
	"de":  "German",
	"ja":  "Japanese",
	"ar":  "Arabic",
	"ru":  "Russian",
	"ko":  "Korean",
	"id":  "Indonesian",
	"it":  "Italian",
	"nl":  "Dutch",
	"tr":  "Turkish",
	"pl":  "Polish",
	"sv":  "Swedish",
	"fil": "Filipino",
	"ms":  "Malay",
	"ro":  "Romanian",
	"uk":  "Ukrainian",
	"el":  "Greek",
	"cs":  "Czech",
	"da":  "Danish",
	"fi":  "Finnish",
	"bg":  "Bulgarian",
	"hr":  "Croatian",
	"sk":  "Slovak",
	"ta":  "Tamil",
}

var ErrUnusualActivityDetected = errors.New("Unusual activity detected. Change proxy.")

func NewElevenLabs(config *config.Config) *ElevenLabs {
	return &ElevenLabs{
		config: config,
	}
}

func GetLanguages() map[string]string {
	return Languages
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

func (el *ElevenLabs) CreateDubbing(filePath string, apiKey *ApiKeyResponse, proxy *torproxy.TorProxy) (*CreateDubbingResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to open file for create dubbing request")
	}
	defer file.Close()

	buff := make([]byte, 512)
	_, err = file.Read(buff)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to read the file for create dubbing request")
	}
	fileContentType := http.DetectContentType(buff)
	file.Seek(0, io.SeekStart)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("name", "dub-dubbing")
	writer.WriteField("source_lang", "en")
	writer.WriteField("target_lang", "ru")
	writer.WriteField("watermark", "true")
	writer.WriteField("end_time", "220")
	writer.WriteField("use_profanity_filter", "false")
	formFileWriter, _ := multiparthelper.CreateFormFile("file", filepath.Base(file.Name()), fileContentType, writer)
	io.Copy(formFileWriter, file)
	writer.Close()

	dialer, _ := proxy.Tor.Dialer(context.Background(), nil)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}

	req, err := http.NewRequest(http.MethodPost, CreateDubbingUrl, body)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to create new create dubbing request")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Add("xi-api-key", apiKey.ApiKey)

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to execute create dubbing request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// StatusForbidden means that we need to change IP
		if res.StatusCode == http.StatusForbidden {
			return nil, ErrUnusualActivityDetected
		}

		errResponse := &CreateDubbingErrorResponse{}
		err = json.NewDecoder(res.Body).Decode(&errResponse)
		if err == nil && errResponse.Detail.Status == "detected_unusual_activity" {
			return nil, ErrUnusualActivityDetected
		}

		body, _ := io.ReadAll(res.Body)
		return nil, errors.Errorf("Create dubbing request responded with status code %d and body %s", res.StatusCode, string(body))
	}

	resData := &CreateDubbingResponse{}
	err = json.NewDecoder(res.Body).Decode(&resData)
	if err != nil {
		return nil, err
	}

	return resData, nil
}

func (el *ElevenLabs) GetDubbingData(dubbing *CreateDubbingResponse, apiKey *ApiKeyResponse) (*GetDubbingDataResponse, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", GetDubbingDataUrl, dubbing.DubbingId), &bytes.Buffer{})
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to create new get dubbing data request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", apiKey.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to execute get dubbing data request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, errors.Errorf("Get dubbing data request responded with status code %d and body %s", res.StatusCode, string(body))
	}

	resData := &GetDubbingDataResponse{}
	err = json.NewDecoder(res.Body).Decode(&resData)
	if err != nil {
		return nil, err
	}

	return resData, nil
}

func (el *ElevenLabs) SaveDubbedFile(savePath, fileName string, dubbing *GetDubbingDataResponse, apiKey *ApiKeyResponse) error {
	err := os.MkdirAll(savePath, os.ModePerm)
	if err != nil {
		return errors.WithMessagef(err, "Unable to create path %s for save dubbed file request", savePath)
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(GetDubbedFileUrl, dubbing.DubbingId, dubbing.TargetLanguages[0]), &bytes.Buffer{})
	if err != nil {
		return errors.WithMessage(err, "Unable to create new save dubbed file request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", apiKey.ApiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.WithMessage(err, "Unable to execute save dubbed file request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return errors.Errorf("Save dubbed file request responded with status code %d and body %s", res.StatusCode, string(body))
	}

	file, err := os.Create(filepath.Join(savePath, fileName+".mp4"))
	if err != nil {
		return errors.WithMessagef(err, "Unable to create file %s for save dubbed file request", file.Name())
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	return err
}

func (el *ElevenLabs) WaitForDubbedFileAndSave(ctx context.Context, maxAttempts, interval int, filePath, savePath, bridge string, apiKey *ApiKeyResponse) error {
	var err error
	var wg sync.WaitGroup
	var createDubbingRes *CreateDubbingResponse

	runtime.EventsEmit(ctx, "LOG", fmt.Sprintf("Dubbing file: %s", filePath))

	fileName := filepath.Base(filePath)

	proxy, err := torproxy.NewTorProxy(bridge, el.config)
	if err != nil {
		runtime.EventsEmit(ctx, "LOG", "Failed to start Tor proxy (maybe because it is blocked in your country).")
		return err
	}
	defer proxy.Close()

	wg.Add(1)
	maxCreateDubbingAttempts := 100
	attempt := 0
	go func() {
		for {
			attempt += 1

			if attempt > 0 {
				_, err = proxy.SwapChain()
				if err != nil {
					runtime.EventsEmit(ctx, "LOG", fmt.Sprintf("Failed to swap NYM (IP) - %s", fileName))
					continue
				}
			}

			createDubbingRes, err = el.CreateDubbing(filePath, apiKey, proxy)
			if err == nil {
				runtime.EventsEmit(ctx, "LOG", fmt.Sprintf("Dubbing successfully started. - %s", fileName))
				wg.Done()
				return
			}

			if attempt >= maxCreateDubbingAttempts {
				runtime.EventsEmit(ctx, "LOG", "Reached maximum limit of attempts to create dubbing. Try again or use/change bridge.")
				err = errors.New("Reached maximum limit of attempts to create dubbing. Try again or use/change bridge.")
				wg.Done()
				return
			}

			if errors.Is(err, ErrUnusualActivityDetected) {
				runtime.EventsEmit(ctx, "LOG", fmt.Sprintf("Bad proxy IP, trying to create dubbing again. [%d/%d] - %s", attempt, maxCreateDubbingAttempts, fileName))
			} else {
				fmt.Println(err)
			}
		}
	}()
	wg.Wait()

	if createDubbingRes == nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	attempt = 0
	var dubbingData *GetDubbingDataResponse
	wg.Add(1)
	go func() {
		for ; ; <-ticker.C {
			attempt += 1

			dubbingData, err = el.GetDubbingData(createDubbingRes, apiKey)
			if err != nil {
				ticker.Stop()
				wg.Done()
				return
			}

			switch dubbingData.Status {
			case "detected_unusual_activity":
				runtime.EventsEmit(ctx, "LOG", "Unusual activity detected. Try again or use/change bridge.")
				err = ErrUnusualActivityDetected
				ticker.Stop()
				wg.Done()
				return
			case "dubbed":
				runtime.EventsEmit(ctx, "LOG", fmt.Sprintf("Dubbing is ready. Downloading... - %s", fileName))
				ticker.Stop()
				wg.Done()
				return
			case "dubbing":
				runtime.EventsEmit(ctx, "LOG", fmt.Sprintf("Dubbing in progress. - %s", fileName))
			default:
				err = errors.New(dubbingData.Err)
				ticker.Stop()
				wg.Done()
				return
			}

			if attempt >= maxAttempts {
				ticker.Stop()
				wg.Done()
				return
			}
		}
	}()
	wg.Wait()

	if err != nil {
		return err
	}

	err = el.SaveDubbedFile(savePath, fileName, dubbingData, apiKey)
	if err != nil {
		return err
	}

	runtime.EventsEmit(ctx, "LOG", fmt.Sprintf("Dubbing was finished successfully and saved to %s.", savePath))

	return nil
}
