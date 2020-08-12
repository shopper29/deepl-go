package deepl

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/context"
)

func createTranslateResponse(detectLang string, text string) *TranslateResponse {
	var r = &TranslateResponse{
		[]translation{
			{
				DetectedSourceLanguage: detectLang,
				Text:                   text,
			},
		},
	}
	return r
}

func initTestServer(t *testing.T, mockResponseHeaderFile, mockResponseBodyFile string, expectedMethod, expectedRequestPath, expectedRawQuery string) (*Client, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != expectedMethod {
			t.Fatalf("request method wrong. want=%s, got=%s", expectedMethod, req.Method)
		}
		if req.URL.Path != expectedRequestPath {
			t.Fatalf("request path wrong. want=%s, got=%s", expectedRequestPath, req.URL.Path)
		}
		if req.URL.RawQuery != expectedRawQuery {
			t.Fatalf("request query wrong. want=%s, got=%s", expectedRawQuery, req.URL.RawQuery)
		}

		headerBytes, err := ioutil.ReadFile(mockResponseHeaderFile)
		if err != nil {
			t.Fatalf("failed to read header '%s': %s", mockResponseHeaderFile, err.Error())
		}
		firstLine := strings.Split(string(headerBytes), "\n")[0]
		statusCode, err := strconv.Atoi(strings.Fields(firstLine)[1])
		if err != nil {
			t.Fatalf("failed to extract status code from header: %s", err.Error())
		}
		w.WriteHeader(statusCode)

		bodyBytes, err := ioutil.ReadFile(mockResponseBodyFile)
		if err != nil {
			t.Fatalf("failed to read body '%s': %s", mockResponseBodyFile, err.Error())
		}
		w.Write(bodyBytes)
	}))

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to get mock server URL: %s", err.Error())
	}

	cli := &Client{
		BaseURL:    serverURL,
		HTTPClient: server.Client(),
		Logger:     nil,
	}
	teardown := func() {
		server.Close()
	}
	return cli, teardown
}

func TestClient_TranslateSentence(t *testing.T) {
	tt := []struct {
		name string

		inputText       string
		inputSourceLang string
		inputTargetLang string

		mockResponseHeaderFile string
		mockResponseBodyFile   string

		expectedMethod      string
		expectedRequestPath string
		expectedRawQuery    string
		expectedResponse    *TranslateResponse
		expectedErrMessage  string
	}{
		{
			name: "success",

			inputText:       "hello",
			inputSourceLang: "EN",
			inputTargetLang: "JA",

			mockResponseHeaderFile: "testdata/TranslateText/success-header",
			mockResponseBodyFile:   "testdata/TranslateText/success-body",

			expectedMethod:      http.MethodPost,
			expectedRequestPath: "/v2/translate",
			expectedRawQuery:    fmt.Sprintf("auth_key=%s&source_lang=EN&target_lang=JA&text=hello", os.Getenv("DEEPL_API_KEY")),
			expectedResponse:    createTranslateResponse("EN", "こんにちわ"),
		},
		{
			name: "misssing target_lang",

			inputText:       "hello",
			inputSourceLang: "EN",
			inputTargetLang: "",

			mockResponseHeaderFile: "testdata/TranslateText/missing-target_lang-header",
			mockResponseBodyFile:   "testdata/TranslateText/missing-target_lang-body",

			expectedMethod:      http.MethodPost,
			expectedRequestPath: "/v2/translate",
			expectedRawQuery:    fmt.Sprintf("auth_key=%s&source_lang=EN&target_lang=&text=hello", os.Getenv("DEEPL_API_KEY")),
			expectedErrMessage:  "Bad request.",
		},
		{
			name: "unsuport target_lang",

			inputText:       "hello",
			inputSourceLang: "EN",
			inputTargetLang: "AA",

			mockResponseHeaderFile: "testdata/TranslateText/unsuport-target_lang-header",
			mockResponseBodyFile:   "testdata/TranslateText/unsuport-target_lang-body",

			expectedMethod:      http.MethodPost,
			expectedRequestPath: "/v2/translate",
			expectedRawQuery:    fmt.Sprintf("auth_key=%s&source_lang=EN&target_lang=AA&text=hello", os.Getenv("DEEPL_API_KEY")),
			expectedErrMessage:  "Bad request.",
		},
		{
			name: "wrong api key",

			inputText:       "hello",
			inputSourceLang: "EN",
			inputTargetLang: "JA",

			mockResponseHeaderFile: "testdata/TranslateText/wrong-apikey-header",
			mockResponseBodyFile:   "testdata/TranslateText/wrong-apikey-body",

			expectedMethod:      http.MethodPost,
			expectedRequestPath: "/v2/translate",
			expectedRawQuery:    fmt.Sprintf("auth_key=%s&source_lang=EN&target_lang=JA&text=hello", os.Getenv("DEEPL_API_KEY")),
			expectedErrMessage:  "Authorization failed.",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cli, teardown := initTestServer(t, tc.mockResponseHeaderFile, tc.mockResponseBodyFile, tc.expectedMethod, tc.expectedRequestPath, tc.expectedRawQuery)
			defer teardown()

			correctResponse, err := cli.TranslateSentence(context.Background(), tc.inputText, tc.inputSourceLang, tc.inputTargetLang)
			if tc.expectedErrMessage == "" {
				if err != nil {
					t.Fatalf("response error should be nil. got=%s", err.Error())
				}

				for i, v := range correctResponse.Translations {
					if v.DetectedSourceLanguage != tc.expectedResponse.Translations[i].DetectedSourceLanguage || v.Text != tc.expectedResponse.Translations[i].Text {
						t.Fatalf("response items wrong. want=%+v, got=%+v", tc.expectedResponse, correctResponse)
					}
				}
			} else {
				if err == nil {
					t.Fatalf("response error should not be non-nil. got=nil")
				}
				if !strings.Contains(err.Error(), tc.expectedErrMessage) {
					t.Fatalf("reponse error message wrong. '%s' is expected to contain '%s'", err.Error(), tc.expectedErrMessage)
				}
			}
		})
	}
}

func TestClient_GetAccountStatus(t *testing.T) {
	tt := []struct {
		name string

		mockResponseHeaderFile string
		mockResponseBodyFile   string

		expectedMethod      string
		expectedRequestPath string
		expectedRawQuery    string
		expectedResponse    *AccountStatus
		expectedErrMessage  string
	}{
		{
			name: "success",

			mockResponseHeaderFile: "testdata/GetAccountStatus/success-header",
			mockResponseBodyFile:   "testdata/GetAccountStatus/success-body",

			expectedMethod:      http.MethodPost,
			expectedRequestPath: "/v2/usage",
			expectedRawQuery:    fmt.Sprintf("auth_key=%s", os.Getenv("DEEPL_API_KEY")),
			expectedResponse:    &AccountStatus{CharacterCount: 30315, CharacterLimit: 1000000},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T){
			cli, teardown := initTestServer(t, tc.mockResponseHeaderFile, tc.mockResponseBodyFile, tc.expectedMethod, tc.expectedRequestPath, tc.expectedRawQuery)
			defer teardown()

			correctResponse, err := cli.GetAccountStatus(context.Background())
			if tc.expectedErrMessage  == "" {
				if err != nil {
					t.Fatalf("response error should be nil. got=%s", err.Error())
				}
				if correctResponse.CharacterCount != tc.expectedResponse.CharacterCount || correctResponse.CharacterLimit != tc.expectedResponse.CharacterLimit {
					t.Fatalf("response items wrong. want=%+v, got=%+v", tc.expectedResponse, correctResponse)
				}
			} else {
				if err == nil {
					t.Fatalf("response error should not be non-nil. got=nil")
				}
			}
		})
	}
}