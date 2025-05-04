package elevenlabs

import "errors"

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
	RemoveDubbingUrl               = "https://api.elevenlabs.io/v1/dubbing"
	GetDubbingDataUrl              = "https://api.elevenlabs.io/v1/dubbing"
	GetDubbedFileUrl               = "https://api.elevenlabs.io/v1/dubbing/%s/audio/%s"
)

var ErrUnusualActivityDetected = errors.New("Unusual activity detected. Change Proxy.")

var Languages = map[string]string{
	"en":  "English",
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
