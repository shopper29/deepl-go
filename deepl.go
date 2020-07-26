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

func decodeBody(bodyBytes []byte, outStruct interface{}) error {
	if err := json.Unmarshal(bodyBytes, outStruct); err != nil {
		err := xerrors.Errorf("Failed to parse Json: %w", err)
		return err
	}
	return nil
}

func (c *Client) TranslateSentence(ctx context.Context, text string, sourceLang string, targetLang string) (*TranslateResponse, error) {
	var transResp TranslateResponse
	var errResp ErrorResponse

	reqURL := *c.BaseURL

	// Set path
	reqURL.Path = path.Join(reqURL.Path, "v2", "translate")

	q := reqURL.Query()

	// need to parepare setting API key in env
	val, ok := os.LookupEnv("DEEPL_API_KEY")
	if !ok {
		err := xerrors.Errorf("Not set API key environment")
		return &transResp, err
	} else if val == "" {
		err := xerrors.Errorf("DEEPL_API_KEY is empty")
		return &transResp, err
	}

	apiKey := os.Getenv("DEEPL_API_KEY")
	q.Add("auth_key", apiKey)
	q.Add("text", text)
	q.Add("target_lang", targetLang)
	q.Add("source_lang", sourceLang)
	reqURL.RawQuery = q.Encode()

	// make new request
	req, err := http.NewRequest(http.MethodPost, reqURL.String(), nil)
	if err != nil {
		err := xerrors.Errorf("Failed to create request: %w", err)
		return &transResp, err
	}

	// set header
	req.Header.Set("User-Agent", "Deepl-Go-Client")

	// set context
	req = req.WithContext(ctx)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		err := xerrors.Errorf("Failed to send http request: %w", err)
		return &transResp, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err := xerrors.Errorf("Failed to read response: %w", err)
		return &transResp, err
	}

	// http response failed and received to error message in json
	var errMessage string
	if resp.StatusCode != http.StatusOK && len(bodyBytes) != 0 {
		err := decodeBody(bodyBytes, &errResp)
		if err != nil {
			return &transResp, xerrors.Errorf("Failed to decode error response: %w", err)
		}
		errMessage = errResp.ErrMessage
	}

	switch resp.StatusCode {
	case http.StatusOK:
		err := decodeBody(bodyBytes, &transResp)
		if err != nil {
			return &transResp, err
		}
		return &transResp, nil
	case http.StatusBadRequest:
		return &transResp, xerrors.Errorf("Bad request. Please check error message and your parameters. Error message is %s", errMessage)
	case http.StatusForbidden:
		return &transResp, xerrors.New("Authorization failed. Please supply a valid auth_key parameter.")
	case http.StatusNotFound:
		return &transResp, xerrors.New("The requested resource clould not be found.")
	case http.StatusRequestEntityTooLarge:
		return &transResp, xerrors.New("The request size exceeds the limit.")
	case http.StatusTooManyRequests:
		return &transResp, xerrors.New("Too many requests. Please wait and resend your request.")
	case 456:
		return &transResp, xerrors.New("Quota exceeded. The character limit has been reached.")
	case http.StatusServiceUnavailable:
		return &transResp, xerrors.New("Resource currently unavailable. Try again later.")
	default:
		// Response status code 5** is internal error but error code "503" is http.StatusServiceUnavailable
		if resp.StatusCode >= 500 {
			return &transResp, xerrors.New("Internal error")
		}
		return &transResp, xerrors.New("Unexpected error")
	}
}
