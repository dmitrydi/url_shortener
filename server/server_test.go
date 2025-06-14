package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/handlers"
	"github.com/dmitrydi/url_shortener/middleware"
	"github.com/dmitrydi/url_shortener/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func MakeTestRouter(getHandler http.HandlerFunc, postHandler http.HandlerFunc, jsonHandler http.HandlerFunc) chi.Router {
	r := chi.NewRouter()
	r.Get(`/{path}`, getHandler)
	r.Post(`/api/shorten`, jsonHandler)
	r.Post(`/`, postHandler)
	return r
}

func TestBasicStorage(t *testing.T) {
	prefix := "prefix/"
	initURL := "some_url"
	pfile := "./dummy.out"
	stor, err := NewBasicStorage(prefix, pfile)
	if err != nil {
		t.Fatal("could not initialize storage ", err.Error())
	}
	defer stor.Close()
	shortURL, err := stor.Put(context.TODO(), initURL, uuid.New())
	require.NoError(t, err, "storage error on Put()")
	assert.Equal(t, len(strings.TrimPrefix(shortURL, prefix)), storage.ShortURLLen, "invalid short URL pattern")
	restoredURL, err := stor.Get(context.TODO(), stor.RemovePrefix(shortURL))
	require.NoError(t, err, "storage error on Get()")
	assert.Equal(t, restoredURL, initURL, "restored URL differs from initial one")
}

func TestPostHandler(t *testing.T) {
	prefix := "http://localhost:8080/"
	pfile := "./dummy.out"
	storage, err := NewBasicStorage(prefix, pfile)
	if err != nil {
		t.Fatal("could not initialize storage ", err.Error())
	}
	defer storage.Close()
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
			handlers.PostHandler(w, request, storage, authorization.UserAuth{UID: uuid.New(), Status: authorization.StatusOK})

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
	pfile := "./dummy.out"
	stor, err := NewBasicStorage(prefix, pfile)
	if err != nil {
		t.Fatal("could not initialize storage ", err.Error())
	}
	defer stor.Close()
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
			handlers.PostHandler(w, putRequest, stor, authorization.UserAuth{UID: uuid.New(), Status: authorization.StatusOK})
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
				handlers.GetHandler(r, getRequest, stor, authorization.UserAuth{UID: uuid.New(), Status: authorization.StatusOK})
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
				handlers.GetHandler(r, getRequest, stor, authorization.UserAuth{UID: uuid.New(), Status: authorization.StatusOK})
				getRes := r.Result()
				defer getRes.Body.Close()
				// проверки
				assert.Equal(t, test.want.getCode, getRes.StatusCode, "invalid response code")
				assert.Empty(t, getRes.Header.Get("Location"), "non-empty redirect on failed request")

			}
		})
	}
}

func TestJSONHandler(t *testing.T) {
	req := makeJSONRequest(t, http.MethodPost, "/api/shorten", "ya.ru")
	req.Header.Set("Content-Type", "application/json")
	prefix := "http://localhost:8080/"
	pfile := "./dummy.out"
	storage, err := NewBasicStorage(prefix, pfile)
	if err != nil {
		t.Fatal("could not initialize storage ", err.Error())
	}
	defer storage.Close()
	w := httptest.NewRecorder()
	handlers.JSONHandler(w, req, storage, authorization.UserAuth{UID: uuid.New(), Status: authorization.StatusOK})
	resp := w.Result()
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"), "invalid content type")
	require.Equal(t, resp.StatusCode, http.StatusCreated, "bad response status")
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	jresp := handlers.JSONResp{}

	err = json.Unmarshal(resBody, &jresp)
	require.NoError(t, err)

	assert.Equal(t, len(prefix)+storage.GetURLSize(), len(jresp.Result), "invalid body size")
}

func TestJSONHandlerBad(t *testing.T) {
	req := makeBadJSONRequest(t, http.MethodPost, "/api/shorten", "ya.ru")
	req.Header.Set("Content-Type", "application/json")
	prefix := "http://localhost:8080/"
	pfile := "./dummy.out"
	storage, err := NewBasicStorage(prefix, pfile)
	if err != nil {
		t.Fatal("could not initialize storage ", err.Error())
	}
	defer storage.Close()
	w := httptest.NewRecorder()
	handlers.JSONHandler(w, req, storage, authorization.UserAuth{UID: uuid.New(), Status: authorization.StatusOK})
	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, http.StatusBadRequest, "unexpected status")
}

type BadJSONRequest struct {
	BadURL string `json:"bad_url"`
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

func makeJSONRequest(t *testing.T, method string, path string, initURL string) *http.Request {
	jreq := handlers.JSONReq{URL: initURL}
	bt, err := json.Marshal(jreq)
	if err != nil {
		t.Fatal("makeJSONRequest: json.Marshal")
	}
	req, err := http.NewRequest(method, path, bytes.NewBuffer(bt))
	if err != nil {
		t.Fatal("http.NewRequest ", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

func makeBadJSONRequest(t *testing.T, method string, path string, initURL string) *http.Request {
	jreq := BadJSONRequest{initURL}
	bt, err := json.Marshal(jreq)
	if err != nil {
		t.Fatal("makeJSONRequest: json.Marshal")
	}
	req, err := http.NewRequest(method, path, bytes.NewBuffer(bt))
	if err != nil {
		t.Fatal("http.NewRequest ", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestRouter(t *testing.T) {
	hostPrefix := "http://localhost:8080/"
	initURL := "www.ya.ru"
	tstorage, err := NewBasicStorage(hostPrefix, "./dummy.out")
	if err != nil {
		t.Fatal("could not initialize storage ", err.Error())
	}
	defer tstorage.Close()
	tserver := httptest.NewServer(MakeTestRouter(
		handlers.WithAuthHandlerWrapper(handlers.GetHandler, tstorage),
		handlers.WithAuthHandlerWrapper(handlers.PostHandler, tstorage),
		handlers.WithAuthHandlerWrapper(handlers.JSONHandler, tstorage)))
	defer tserver.Close()
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

func TestRouterJSONApi(t *testing.T) {
	hostPrefix := "http://localhost:8080/"
	initURL := "www.ya.ru"
	tstorage, err := NewBasicStorage(hostPrefix, "./dummy.out")
	if err != nil {
		t.Fatal("could not initialize storage ", err.Error())
	}
	defer tstorage.Close()
	tserver := httptest.NewServer(MakeTestRouter(
		handlers.WithAuthHandlerWrapper(handlers.GetHandler, tstorage),
		handlers.WithAuthHandlerWrapper(handlers.PostHandler, tstorage),
		handlers.WithAuthHandlerWrapper(handlers.JSONHandler, tstorage)))
	defer tserver.Close()
	req := makeJSONRequest(t, http.MethodPost, tserver.URL+"/api/shorten", initURL)
	resp, err := tserver.Client().Do(req)
	require.NoError(t, err, "server error")
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "bad response status")
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"), "bad response content type")

	defer resp.Body.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err, "io.ReadAll error")
	r := handlers.JSONResp{}
	err = json.Unmarshal(buf.Bytes(), &r)
	require.NoError(t, err, "json.Unmarshal")
	assert.Equal(t, len(hostPrefix)+tstorage.GetURLSize(), len(r.Result), "invalid body size")
}

func TestRouterCompress(t *testing.T) {
	hostPrefix := "http://localhost:8080/"
	initURL := "www.ya.ru"
	tstorage, err := NewBasicStorage(hostPrefix, "./dummy.out")
	if err != nil {
		t.Fatal("could not initialize storage ", err.Error())
	}
	defer tstorage.Close()
	writerPool := &sync.Pool{
		New: func() any {
			writer, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			return writer
		},
	}
	getHandler := middleware.CompressHandler(handlers.WithAuthHandlerWrapper(handlers.GetHandler, tstorage), writerPool)
	postHandler := middleware.CompressHandler(handlers.WithAuthHandlerWrapper(handlers.PostHandler, tstorage), writerPool)
	jsonHandler := middleware.CompressHandler(handlers.WithAuthHandlerWrapper(handlers.JSONHandler, tstorage), writerPool)
	tserver := httptest.NewServer(MakeTestRouter(getHandler, postHandler, jsonHandler))
	defer tserver.Close()
	req := makeJSONRequest(t, http.MethodPost, tserver.URL+"/api/shorten", initURL)
	resp, err := tserver.Client().Do(req)
	require.NoError(t, err, "server error")
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "bad response status")
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"), "bad response content type")
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err, "io.ReadAll error")
	r := handlers.JSONResp{}
	err = json.Unmarshal(buf.Bytes(), &r)
	require.NoError(t, err, "json.Unmarshal")
	assert.Equal(t, len(hostPrefix)+tstorage.GetURLSize(), len(r.Result), "invalid body size")
}

func TestRouterCompressDB(t *testing.T) {
	hostPrefix := "http://localhost:8080/"
	initURL := "www.ya.ru"
	tstorage, err := NewBasicStorage(hostPrefix, "./dummy.out")
	if err != nil {
		t.Fatal("could not initialize storage ", err.Error())
	}
	defer tstorage.Close()
	writerPool := &sync.Pool{
		New: func() any {
			writer, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			return writer
		},
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Sync()

	getHandler := handlers.WithAuthHandlerWrapper(handlers.GetHandler, tstorage)
	postHandler := handlers.WithAuthHandlerWrapper(handlers.PostHandler, tstorage)
	jsonHandler := handlers.WithAuthHandlerWrapper(handlers.JSONHandler, tstorage)
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		`localhost`, `user2`, `07512851SqlPass`, `videos`)

	db, err := sql.Open("pgx", ps)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	pingHandler := handlers.MakePingHandler(db)
	batchHandler := handlers.WithAuthHandlerWrapper(handlers.BatchHandler, tstorage)
	userHandler := handlers.WithAuthHandlerWrapper(handlers.GetByUserHandler, tstorage)
	tserver := httptest.NewServer(MakeRouter(getHandler, postHandler, jsonHandler, pingHandler, batchHandler, userHandler, logger, writerPool))
	defer tserver.Close()
	req := makeJSONRequest(t, http.MethodPost, tserver.URL+"/api/shorten", initURL)
	resp, err := tserver.Client().Do(req)
	require.NoError(t, err, "server error")
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "bad response status")
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"), "bad response content type")
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err, "io.ReadAll error")
	r := handlers.JSONResp{}
	err = json.Unmarshal(buf.Bytes(), &r)
	require.NoError(t, err, "json.Unmarshal")
	assert.Equal(t, len(hostPrefix)+tstorage.GetURLSize(), len(r.Result), "invalid body size")
}
