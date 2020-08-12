package deepl

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"golang.org/x/xerrors"
)

type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	Logger     *log.Logger
}

func New(rawBaseURL string, logger *log.Logger) (*Client, error) {
	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		err := xerrors.Errorf("Failed to parse URL")
		return nil, err
	}

	if logger == nil {
		logger = log.New(os.Stderr, "[Log]", log.LstdFlags)
	}

	return &Client{
		BaseURL:    baseURL,
		HTTPClient: http.DefaultClient,
		Logger:     logger,
	}, nil
}

type TranslateResponse struct {
	Translations []translation `json:"translations"`
}

type translation struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
}

type ErrorResponse struct {
	ErrMessage string `json:"message"`
}

type AccountStatus struct {
	CharacterCount int `json:"character_count"`
	CharacterLimit int `json:"character_limit"`
}

func getAPIKey() (string, error) {
	// need to parepare setting API key in env
	val, ok := os.LookupEnv("DEEPL_API_KEY")
	if !ok {
		err := xerrors.New("Not set API key environment")
		return "", err
	} else if val == "" {
		err := xerrors.New("DEEPL_API_KEY is empty")
		return "", err
	}

	apiKey := os.Getenv("DEEPL_API_KEY")
	return apiKey, nil
}

func decodeBody(bodyBytes []byte, outStruct interface{}) error {
	if err := json.Unmarshal(bodyBytes, outStruct); err != nil {
		return err
	}
	return nil
}

func responseParse(resp *http.Response, outStruct interface{}) error {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err := xerrors.Errorf("Failed to read response: %w", err)
		return err
	}

	// http response failed and received to error message in json
	var errResp ErrorResponse
	var errMessage string

	if resp.StatusCode != http.StatusOK && len(bodyBytes) != 0 {
		err := decodeBody(bodyBytes, &errResp)
		if err != nil {
			return xerrors.Errorf("Failed to decode error response: %w", err)
		}
		errMessage = errResp.ErrMessage
	}

	switch resp.StatusCode {
	case http.StatusOK:
		err := decodeBody(bodyBytes, &outStruct)
		if err != nil {
			return xerrors.Errorf("Failed to parse Json: %w", err)
		}
		return nil
	case http.StatusBadRequest:
		return xerrors.Errorf("Bad request. Please check error message and your parameters. Error message is %s", errMessage)
	case http.StatusForbidden:
		return xerrors.New("Authorization failed. Please supply a valid auth_key parameter.")
	case http.StatusNotFound:
		return xerrors.New("The requested resource clould not be found.")
	case http.StatusRequestEntityTooLarge:
		return xerrors.New("The request size exceeds the limit.")
	case http.StatusTooManyRequests:
		return xerrors.New("Too many requests. Please wait and resend your request.")
	case 456:
		return xerrors.New("Quota exceeded. The character limit has been reached.")
	case http.StatusServiceUnavailable:
		return xerrors.New("Resource currently unavailable. Try again later.")
	default:
		// Response status code 5** is internal error but error code "503" is http.StatusServiceUnavailable
		if resp.StatusCode >= 500 {
			return xerrors.New("Internal error")
		}
		return xerrors.New("Unexpected error")
	}
}

func (c *Client) GetAccountStatus(ctx context.Context) (*AccountStatus, error) {
	var accountStatusResp AccountStatus

	reqURL := *c.BaseURL

	// Set path
	reqURL.Path = path.Join(reqURL.Path, "v2", "usage")

	q := reqURL.Query()

	apiKey, err := getAPIKey()
	if err != nil {
		return nil, err
	}

	q.Add("auth_key", apiKey)
	reqURL.RawQuery = q.Encode()

	// make new request
	req, err := http.NewRequest(http.MethodPost, reqURL.String(), nil)
	if err != nil {
		err := xerrors.Errorf("Failed to create request: %w", err)
		return nil, err
	}

	// set header
	req.Header.Set("User-Agent", "Deepl-Go-Client")

	// set context
	req = req.WithContext(ctx)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		err := xerrors.Errorf("Failed to send http request: %w", err)
		return nil, err
	}
	defer resp.Body.Close()

	if err := responseParse(resp, &accountStatusResp); err != nil {
		return nil, err
	}
	return &accountStatusResp, nil
}

func (c *Client) TranslateSentence(ctx context.Context, text string, sourceLang string, targetLang string) (*TranslateResponse, error) {
	var transResp TranslateResponse

	reqURL := *c.BaseURL

	// Set path
	reqURL.Path = path.Join(reqURL.Path, "v2", "translate")

	q := reqURL.Query()

	apiKey, err := getAPIKey()
	if err != nil {
		return nil, err
	}

	q.Add("auth_key", apiKey)
	q.Add("text", text)
	q.Add("target_lang", targetLang)
	q.Add("source_lang", sourceLang)
	reqURL.RawQuery = q.Encode()

	// make new request
	req, err := http.NewRequest(http.MethodPost, reqURL.String(), nil)
	if err != nil {
		err := xerrors.Errorf("Failed to create request: %w", err)
		return nil, err
	}

	// set header
	req.Header.Set("User-Agent", "Deepl-Go-Client")

	// set context
	req = req.WithContext(ctx)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		err := xerrors.Errorf("Failed to send http request: %w", err)
		return nil, err
	}
	defer resp.Body.Close()

	if err := responseParse(resp, &transResp); err != nil {
		return nil, err
	}

	return &transResp, nil
}
