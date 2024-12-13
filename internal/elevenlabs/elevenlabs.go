package elevenlabs

import (
	"bytes"
	"context"
	"eleven-hacks/internal/app"
	"eleven-hacks/internal/event"
	multiparthelper "eleven-hacks/internal/helper/multipart-helper"
	"eleven-hacks/internal/torproxy"
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
	Proxy  *torproxy.TorProxy
	client *http.Client
	ctx    context.Context
}

func New(ctx context.Context, startProxy bool) (*ElevenLabs, error) {
	var err error
	el := &ElevenLabs{
		ctx: ctx,
	}

	el.client = http.DefaultClient
	config := app.GetConfig(ctx)

	if startProxy {
		el.Proxy, err = torproxy.New(config)
		if err != nil {
			return nil, err
		}

		dialer, _ := el.Proxy.Tor.Dialer(ctx, nil)
		el.client = &http.Client{
			Transport: &http.Transport{
				DialContext: dialer.DialContext,
			},
		}
	}

	return el, nil
}

func (el *ElevenLabs) doRequestWithRetries(req *http.Request, maxRetries int, expectedResponse []int) (*http.Response, *[]byte, error) {
	var err error
	var res *http.Response
	var body []byte

	defer func() {
		if res != nil {
			body, _ = io.ReadAll(res.Body)
			res.Body.Close()
		}
	}()

	var originalBody []byte
	if req.Body != nil {
		originalBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(originalBody))
	}

	attempt := 1

RequestLoop:
	for {
		select {
		case <-el.ctx.Done():
			return nil, nil, el.ctx.Err()
		default:
			res, err = el.client.Do(req)
			if err == nil {
				for _, code := range expectedResponse {
					if res.StatusCode == code {
						return res, &body, nil
					}
				}
			}

			if res != nil {
				res.Body.Close()
			}

			attempt++
			if attempt == maxRetries {
				break RequestLoop
			}

			if req.Body != nil {
				req.Body = io.NopCloser(bytes.NewReader(originalBody))
			}

			if el.Proxy != nil {
				el.Proxy.NewNym()
			}
		}
	}

	return res, &body, err
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

	res, body, err := el.doRequestWithRetries(req, 50, []int{http.StatusOK})
	if err != nil {
		if res != nil && body != nil {
			return errors.Errorf("Pre-sign-up request responded with status code %d and body %s", res.StatusCode, string(*body))
		}
		return errors.WithMessage(err, "Unable to execute pre-sign-up request")
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

	res, body, err := el.doRequestWithRetries(req, 10, []int{http.StatusOK})
	if err != nil {
		if res != nil && body != nil {
			return errors.Errorf("Email verification request responded with status code %d and body %s", res.StatusCode, string(*body))
		}
		return errors.WithMessage(err, "Unable to execute email verification request")
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

	// Usually response status code will be 400, because tmp emails can't be verified by Google
	// but ElevenLabs just ignores it
	res, body, err := el.doRequestWithRetries(req, 10, []int{http.StatusOK, http.StatusBadRequest})
	if err != nil {
		if res != nil && body != nil {
			return errors.Errorf(
				"Account sign-up request responded with status code %d and body %s\nURL: %s",
				res.StatusCode,
				string(*body),
				req.URL,
			)
		}
		return errors.WithMessage(err, "Unable to execute account sign-up request")
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

	res, body, err := el.doRequestWithRetries(req, 10, []int{http.StatusOK})
	if err != nil {
		if res != nil && body != nil {
			return nil, errors.Errorf("Sign-in request responded with status code %d and body %s", res.StatusCode, string(*body))
		}
		return nil, errors.WithMessage(err, "Unable to execute sign-in request")
	}

	resData := &SignInResponse{}
	err = json.NewDecoder(bytes.NewReader(*body)).Decode(&resData)
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

	res, body, err := el.doRequestWithRetries(req, 10, []int{http.StatusOK})
	if err != nil {
		if res != nil && body != nil {
			return errors.Errorf("Account update request responded with status code %d and body %s", res.StatusCode, string(*body))
		}
		return errors.WithMessage(err, "Unable to execute account update request")
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

	res, body, err := el.doRequestWithRetries(req, 10, []int{http.StatusOK})
	if err != nil {
		if res != nil && body != nil {
			return errors.Errorf("Prepare internal verification request responded with status code %d and body %s", res.StatusCode, string(*body))
		}
		return errors.WithMessage(err, "Unable to execute prepare internal verification request")
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

	res, body, err := el.doRequestWithRetries(req, 10, []int{http.StatusOK})
	if err != nil {
		if res != nil && body != nil {
			return nil, errors.Errorf("Create api key request responded with status code %d and body %s", res.StatusCode, string(*body))
		}
		return nil, errors.WithMessage(err, "Unable to execute create api key request")
	}

	resData := &ApiKeyResponse{}
	err = json.NewDecoder(bytes.NewReader(*body)).Decode(&resData)
	if err != nil {
		return nil, err
	}

	return resData, nil
}

func (el *ElevenLabs) CreateDubbing(ctx context.Context, reader io.Reader, fileName, sourceLang, targetLang string, apiKey *ApiKeyResponse) (*CreateDubbingResponse, error) {
	seeker, ok := reader.(io.Seeker)
	if !ok {
		return nil, errors.New("Reader does not support seeker")
	}

	buff := make([]byte, 512)
	_, err := reader.Read(buff)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("Unable to read file %s", fileName))
	}
	fileContentType := http.DetectContentType(buff)
	seeker.Seek(0, io.SeekStart)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("name", "dub-dubbing")
	writer.WriteField("source_lang", sourceLang)
	writer.WriteField("target_lang", targetLang)
	writer.WriteField("watermark", "true")
	writer.WriteField("end_time", "220")
	writer.WriteField("use_profanity_filter", "false")
	formFileWriter, _ := multiparthelper.CreateFormFile("file", fileName, fileContentType, writer)
	io.Copy(formFileWriter, reader)
	seeker.Seek(0, io.SeekStart)
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, CreateDubbingUrl, body)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to create new create dubbing request")
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Add("xi-api-key", apiKey.ApiKey)

	res, resBody, err := el.doRequestWithRetries(req, 1, []int{http.StatusOK})
	if err != nil {
		if res != nil {
			// StatusForbidden means that we need to change IP
			if res.StatusCode == http.StatusForbidden {
				return nil, ErrUnusualActivityDetected
			}

			if resBody != nil {
				errResponse := &CreateDubbingErrorResponse{}
				err = json.NewDecoder(bytes.NewReader(*resBody)).Decode(&errResponse)
				if err == nil && errResponse.Detail.Status == "detected_unusual_activity" {
					return nil, ErrUnusualActivityDetected
				}

				return nil, errors.Errorf("Create dubbing request responded with status code %d and body %s", res.StatusCode, string(*resBody))
			}
		}
		return nil, errors.WithMessage(err, "Unable to execute create dubbing request")
	}

	resData := &CreateDubbingResponse{}
	err = json.NewDecoder(bytes.NewReader(*resBody)).Decode(&resData)
	if err != nil {
		return nil, err
	}

	return resData, nil
}

func (el *ElevenLabs) RemoveDubbing(dubbingId string, apiKey *ApiKeyResponse) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", RemoveDubbingUrl, dubbingId), &bytes.Buffer{})
	if err != nil {
		return errors.WithMessage(err, "Unable to create new remove dubbing request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", apiKey.ApiKey)

	res, body, err := el.doRequestWithRetries(req, 1, []int{http.StatusOK})
	if err != nil {
		if res != nil && body != nil {
			return errors.Errorf("Remove dubbing request responded with status code %d and body %s", res.StatusCode, string(*body))
		}
		return errors.WithMessage(err, "Unable to execute remove dubbing request")
	}

	resData := &GetDubbingDataResponse{}
	err = json.NewDecoder(bytes.NewReader(*body)).Decode(&resData)
	if err != nil {
		return err
	}

	return nil
}

func (el *ElevenLabs) GetDubbingData(dubbing *CreateDubbingResponse, apiKey *ApiKeyResponse) (*GetDubbingDataResponse, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", GetDubbingDataUrl, dubbing.DubbingId), &bytes.Buffer{})
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to create new get dubbing data request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", apiKey.ApiKey)

	res, body, err := el.doRequestWithRetries(req, 1, []int{http.StatusOK})
	if err != nil {
		if res != nil && body != nil {
			return nil, errors.Errorf("Get dubbing data request responded with status code %d and body %s", res.StatusCode, string(*body))
		}
		return nil, errors.WithMessage(err, "Unable to execute get dubbing data request")
	}

	resData := &GetDubbingDataResponse{}
	err = json.NewDecoder(bytes.NewReader(*body)).Decode(&resData)
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

	res, body, err := el.doRequestWithRetries(req, 1, []int{http.StatusOK})
	if err != nil {
		if res != nil && body != nil {
			return errors.Errorf("Save dubbed file request responded with status code %d and body %s", res.StatusCode, string(*body))
		}
		return errors.WithMessage(err, "Unable to execute save dubbed file request")
	}

	filePath := filepath.Join(savePath, fileName+".mp4")
	file, err := os.Create(filePath)
	if err != nil {
		return errors.WithMessagef(err, "Unable to create file %s for save dubbed file request", filePath)
	}
	defer file.Close()

	_, err = io.Copy(file, bytes.NewReader(*body))
	return err
}

type DubbingFile struct {
	Status  DubbingFileStatus `json:"status"`
	Path    string            `json:"path"`
	Name    string            `json:"name"`
	Attempt int               `json:"attempt"`
	ApiKey  *ApiKeyResponse   `json:"apiKey"`
}

type DubbingFileStatus string

const (
	StatusAdded       DubbingFileStatus = "Added"
	StatusAccount     DubbingFileStatus = "Creating an account"
	StatusTryDubbing  DubbingFileStatus = "Trying to dub"
	StatusDubbing     DubbingFileStatus = "Dubbing!"
	StatusDownloading DubbingFileStatus = "Downloading..."
	StatusError       DubbingFileStatus = "Error"
)

type DubbingParams struct {
	MaxTry     int
	Interval   int
	SavePath   string
	Bridge     string
	SourceLang string
	TargetLang string
}

func WaitForDubbedFileAndSave(ctx context.Context, df *DubbingFile, dp *DubbingParams) error {
	var err error
	var mx sync.Mutex
	var createDubbingRes *CreateDubbingResponse

	el, err := New(ctx, false)
	if err != nil {
		return err
	}

	config := app.GetConfig(ctx)
	assets := app.GetAssets(ctx)

	defer func() {
		if el.Proxy != nil {
			el.Proxy.Close()
		}

		if err != nil {
			mx.Lock()
			df.Status = StatusError
			mx.Unlock()

			runtime.EventsEmit(ctx, event.DubbingUpdate)
		}
	}()

	mx.Lock()
	df.Status = StatusTryDubbing
	mx.Unlock()

	runtime.EventsEmit(ctx, event.DubbingUpdate)

	wormFile, err := assets.Open("frontend/src/assets/videos/worm.mp4")
	if err != nil {
		err = errors.WithMessage(err, fmt.Sprintf("Unable to open worm file"))
		return err
	}
	defer wormFile.Close()

	wormFileBuff := &bytes.Buffer{}
	_, err = wormFileBuff.ReadFrom(wormFile)
	if err != nil {
		err = errors.WithMessage(err, fmt.Sprintf("Unable to read worm file - %s", err.Error()))
		return err
	}
	wormFileReader := bytes.NewReader(wormFileBuff.Bytes())

	file, err := os.Open(df.Path)
	if err != nil {
		err = errors.WithMessage(err, fmt.Sprintf("Unable to open file %s", df.Path))
		return err
	}
	defer file.Close()

	df.Attempt = 0
	try := 0

TryingLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if try >= dp.MaxTry {
				try = 0
				el.Proxy.Close()
				el.Proxy = nil
			}

			if el.Proxy == nil {
				el.Proxy, err = torproxy.New(config)
				if err != nil {
					continue
				}

				dialer, _ := el.Proxy.Tor.Dialer(ctx, nil)
				el.client = &http.Client{
					Transport: &http.Transport{
						DialContext: dialer.DialContext,
					},
				}
			}

			mx.Lock()
			df.Attempt += 1
			mx.Unlock()
			try += 1

			runtime.EventsEmit(ctx, event.DubbingUpdate)

			if df.Attempt > 0 {
				_, err = el.Proxy.NewNym()
				if err != nil {
					continue
				}
			}

			createDubbingRes, err = el.CreateDubbing(ctx, wormFileReader, df.Name, dp.SourceLang, dp.TargetLang, df.ApiKey)
			if err == nil {
				el.RemoveDubbing(createDubbingRes.DubbingId, df.ApiKey)

				createDubbingRes, err = el.CreateDubbing(ctx, file, df.Name, dp.SourceLang, dp.TargetLang, df.ApiKey)
				if err == nil {
					break TryingLoop
				}
			}
		}
	}

	if createDubbingRes == nil {
		return err
	}

	mx.Lock()
	df.Status = StatusDubbing
	mx.Unlock()

	runtime.EventsEmit(ctx, event.DubbingUpdate)

	ticker := time.NewTicker(time.Duration(dp.Interval) * time.Second)
	var dubbingData *GetDubbingDataResponse

DubbingLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			dubbingData, err = el.GetDubbingData(createDubbingRes, df.ApiKey)
			if err != nil {
				_, err = el.Proxy.NewNym()
				if err != nil {
					continue
				}
			}

			switch dubbingData.Status {
			case "detected_unusual_activity":
				err = ErrUnusualActivityDetected
				ticker.Stop()
				break
			case "dubbed":
				ticker.Stop()
				break DubbingLoop
			case "dubbing":
				//
			default:
				err = errors.New(dubbingData.Err)
				ticker.Stop()
				break
			}
		}
	}

	if err != nil {
		return err
	}

	df.Status = StatusDownloading
	runtime.EventsEmit(ctx, event.DubbingUpdate)

SaveDubbingLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err = el.SaveDubbedFile(dp.SavePath, df.Name, dubbingData, df.ApiKey)
			if err != nil {
				_, err = el.Proxy.NewNym()
				if err != nil {
					continue
				}
			}

			break SaveDubbingLoop
		}
	}

	return nil
}
