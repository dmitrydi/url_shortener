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

func TestBasicStorage(t *testing.T) {
	prefix := "prefix/"
	initURL := "some_url"
	stor := NewBasicStorage(prefix)
	shortURL, err := stor.Put(initURL)
	require.NoError(t, err, "storage error on Put()")
	assert.Equal(t, len(strings.TrimPrefix(shortURL, prefix)), shortURLLen, "invalid short URL pattern")
	restoredURL, err := stor.Get(stor.RemovePrefix(shortURL))
	require.NoError(t, err, "storage error on Get()")
	assert.Equal(t, restoredURL, initURL, "restored URL differs from initial one")
}

func TestPostHandler(t *testing.T) {
	prefix := "http://localhost:8080/"
	storage := NewBasicStorage(prefix)
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
			name:    "positive_test_#1",
			initURL: "ya.ru",
			method:  http.MethodPost,
			want: want{
				code:        201,
				contentType: "text/plain",
			},
		},
		{
			name:    "bad_method_#1",
			initURL: "ya.ru",
			method:  http.MethodGet,
			want:    want{code: 400},
		},
		{
			name:    "bad_method_#2",
			initURL: "ya.ru",
			method:  http.MethodPut,
			want:    want{code: 400},
		},
		{
			name:    "empty_url",
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
			assert.Equal(t, test.want.code, res.StatusCode, "wrong response status")
			if test.method == http.MethodPost && len(test.initURL) > 0 {
				// получаем и проверяем тело запроса
				defer res.Body.Close()
				resBody, err := io.ReadAll(res.Body)

				require.NoError(t, err, "io.ReadAll error")
				assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"), "invalid content type")
				assert.Equal(t, len(prefix)+storage.GetURLSize(), len(string(resBody)), "invalid body size")
			}

		})
	}
}

func TestGetHandler(t *testing.T) {
	prefix := "http://localhost:8080/"
	stor := NewBasicStorage(prefix)
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
		want      want
	}{
		{
			name:      "put_and_get_ok",
			initURL:   "ya.ru",
			putMethod: http.MethodPost,
			getMethod: http.MethodGet,
			want: want{
				putCode:  201,
				getCode:  307,
				location: "ya.ru",
			},
		}, {
			name:      "put_and_get_fail",
			initURL:   "ya.ru",
			putMethod: http.MethodPut,
			getMethod: http.MethodGet,
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
			assert.Equal(t, test.want.putCode, putRes.StatusCode, "invalid status code")
			if putRes.StatusCode == http.StatusCreated {
				defer putRes.Body.Close()
				resBody, err := io.ReadAll(putRes.Body)
				require.NoError(t, err, "io.ReadAll error")
				// удаляем префикс из результата запроса
				shortPath := "/" + strings.TrimPrefix(string(resBody), prefix)

				// делаем get-запрос к серверу
				getRequest := httptest.NewRequest(test.getMethod, shortPath, nil)
				r := httptest.NewRecorder()
				GetHandler(r, getRequest, stor)
				getRes := r.Result()
				defer getRes.Body.Close()
				// проверки
				assert.Equal(t, test.want.getCode, getRes.StatusCode, "invalid response code")
				assert.Equal(t, test.want.location, getRes.Header.Get("Location"), "wrong redirect")

			} else {
				shortPath := "/" + randomPath

				// делаем get-запрос к серверу
				getRequest := httptest.NewRequest(test.getMethod, shortPath, nil)
				r := httptest.NewRecorder()
				GetHandler(r, getRequest, stor)
				getRes := r.Result()
				defer getRes.Body.Close()
				// проверки
				assert.Equal(t, test.want.getCode, getRes.StatusCode, "invalid response code")
				assert.Empty(t, getRes.Header.Get("Location"), "non-empty redirect on failed request")

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
	require.NoError(t, err, "server error")
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestRouter(t *testing.T) {
	hostPrefix := "http://localhost:8080/"
	initURL := "www.ya.ru"
	tstorage := NewBasicStorage(hostPrefix)
	tserver := httptest.NewServer(MakeRouter(MakeGetHandler(tstorage), MakePostHandler(tstorage)))
	defer tserver.Close()
	//tserver.Start()
	postResp, postBody := testRequest(t, tserver, http.MethodPost, "/", initURL)
	defer postResp.Body.Close()
	assert.Equal(t, postResp.StatusCode, http.StatusCreated, "expected successful creation")
	path := strings.TrimPrefix(postBody, hostPrefix)
	assert.Equal(t, len([]rune(path)), 8, "unexpected URL size")
	getResp, _ := testRequest(t, tserver, http.MethodGet, "/"+path, "")
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, getResp.StatusCode, "invalid response code")
	assert.Equal(t, initURL, getResp.Header.Get("Location"), "invalid redirect")

}
