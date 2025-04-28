package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dmitrydi/url_shortener/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostHandler(t *testing.T) {
	prefix := "http://localhost:8080/"
	storage := storage.NewBasicStorage(prefix)
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name    string
		initURL string
		method  string
		want    want
	}{
		{
			name:    "positive test #1",
			initURL: "ya.ru",
			method:  http.MethodPost,
			want: want{
				code:        201,
				contentType: "text/plain",
			},
		},
		{
			name:    "bad method #1",
			initURL: "ya.ru",
			method:  http.MethodGet,
			want:    want{code: 400},
		},
		{
			name:    "bad method #2",
			initURL: "ya.ru",
			method:  http.MethodPut,
			want:    want{code: 400},
		},
		{
			name:    "empty url",
			initURL: "",
			method:  http.MethodPost,
			want:    want{code: 400},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/", bytes.NewBuffer([]byte(test.initURL)))
			// создаём новый Recorder
			w := httptest.NewRecorder()
			PostHandler(w, request, storage)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.code, res.StatusCode)
			if test.method == http.MethodPost && len(test.initURL) > 0 {
				// получаем и проверяем тело запроса
				defer res.Body.Close()
				resBody, err := io.ReadAll(res.Body)

				require.NoError(t, err)
				assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
				assert.Equal(t, len(prefix)+storage.GetURLSize(), len(string(resBody)))
			}

		})
	}
}

func TestGetHandler(t *testing.T) {
	prefix := "http://localhost:8080/"
	stor := storage.NewBasicStorage(prefix)
	randomPath := storage.MakeRandomString(8)
	type want struct {
		putCode  int
		getCode  int
		location string
	}
	tests := []struct {
		name      string
		initURL   string
		putMethod string
		getMethod string
		getNum    int
		want      want
	}{
		{
			name:      "put and get ok",
			initURL:   "ya.ru",
			putMethod: http.MethodPost,
			getMethod: http.MethodGet,
			getNum:    10,
			want: want{
				putCode:  201,
				getCode:  307,
				location: "ya.ru",
			},
		}, {
			name:      "put and get fail",
			initURL:   "ya.ru",
			putMethod: http.MethodPut,
			getMethod: http.MethodGet,
			getNum:    10,
			want: want{
				putCode: 400,
				getCode: 400,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			putRequest := httptest.NewRequest(test.putMethod, "/", bytes.NewBuffer([]byte(test.initURL)))
			w := httptest.NewRecorder()
			PostHandler(w, putRequest, stor)
			putRes := w.Result()
			assert.Equal(t, test.want.putCode, putRes.StatusCode)
			if putRes.StatusCode == http.StatusCreated {
				defer putRes.Body.Close()
				resBody, err := io.ReadAll(putRes.Body)
				require.NoError(t, err)
				// удаляем префикс из результата запроса
				shortPath := "/" + strings.TrimPrefix(string(resBody), prefix)
				for i := 0; i < test.getNum; i++ {
					// делаем get-запрос к серверу
					getRequest := httptest.NewRequest(test.getMethod, shortPath, nil)
					r := httptest.NewRecorder()
					GetHandler(r, getRequest, stor)
					getRes := r.Result()
					defer getRes.Body.Close()
					// проверки
					assert.Equal(t, test.want.getCode, getRes.StatusCode)
					assert.Equal(t, test.want.location, getRes.Header.Get("Location"))
				}

			} else {
				shortPath := "/" + randomPath
				for i := 0; i < test.getNum; i++ {
					// делаем get-запрос к серверу
					getRequest := httptest.NewRequest(test.getMethod, shortPath, nil)
					r := httptest.NewRecorder()
					GetHandler(r, getRequest, stor)
					getRes := r.Result()
					defer getRes.Body.Close()
					// проверки
					assert.Equal(t, test.want.getCode, getRes.StatusCode)
					assert.Empty(t, getRes.Header.Get("Location"))
				}
			}
		})
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method,
	path, body string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	cli := ts.Client()

	cli.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestRouter(t *testing.T) {
	hostPrefix := "http://localhost:8080/"
	initURL := "www.ya.ru"
	tserver := httptest.NewServer(MakeRouter(hostPrefix))
	defer tserver.Close()
	//tserver.Start()
	postResp, postBody := testRequest(t, tserver, http.MethodPost, "/", initURL)
	defer postResp.Body.Close()
	assert.Equal(t, postResp.StatusCode, http.StatusCreated)
	path := strings.TrimPrefix(postBody, hostPrefix)
	assert.Equal(t, len([]rune(path)), 8)
	getResp, _ := testRequest(t, tserver, http.MethodGet, "/"+path, "")
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, getResp.StatusCode)
	assert.Equal(t, initURL, getResp.Header.Get("Location"))

}
