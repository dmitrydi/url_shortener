package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/mocks"
	"github.com/dmitrydi/url_shortener/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockURLStorage(ctrl)

	type want struct {
		getCode  int
		location string
	}
	tests := []struct {
		name      string
		shortURL  string
		getMethod string
		bad       bool
		want      want
	}{
		{
			name:      "put_and_get_ok",
			shortURL:  "short_url",
			getMethod: http.MethodGet,
			bad:       false,
			want: want{
				getCode:  307,
				location: "ya.ru",
			},
		},
		{
			name:      "put_and_get_fail",
			shortURL:  "short_url",
			getMethod: http.MethodGet,
			bad:       true,
			want: want{
				getCode:  400,
				location: "ya.ru",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			getRequest := httptest.NewRequest(test.getMethod, "/"+test.shortURL, nil)
			if !test.bad {
				m.EXPECT().Get(gomock.Any(), test.shortURL).Return(test.want.location, nil)
			} else {
				m.EXPECT().Get(gomock.Any(), test.shortURL).Return("", errors.New("storage error"))
			}

			r := httptest.NewRecorder()
			GetHandler(r, getRequest, m)
			getRes := r.Result()
			defer getRes.Body.Close()
			assert.Equal(t, test.want.getCode, getRes.StatusCode, "invalid response code")
			if !test.bad {
				assert.Equal(t, test.want.location, getRes.Header.Get("Location"), "wrong redirect")
			}

		})
	}
}

func TestBatchHandlerOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reqData := []storage.OriginalData{
		{CorrelationID: "corr1", OriginalURL: "url1"},
		{CorrelationID: "corr2", OriginalURL: "url2"},
	}

	putResult := storage.ShortenedBatch{
		{CorrelationID: "corr1", ShortURL: "short1"},
		{CorrelationID: "corr2", ShortURL: "short2"},
	}

	m := mocks.NewMockURLStorage(ctrl)
	m.EXPECT().PutMany(gomock.Any(), reqData, gomock.Any()).Return(putResult, nil)

	bt, err := json.Marshal(reqData)
	require.NoError(t, err, "json error")
	req, err := http.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(bt))
	require.NoError(t, err, "http request error")
	r := httptest.NewRecorder()
	BatchHandler(r, req, m, authorization.UserAuth{})
	res := r.Result()

	// базовая проверка
	assert.Equal(t, res.StatusCode, http.StatusCreated, "bad response status")
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"), "bad response content type")

	// проверка тела ответа
	defer res.Body.Close()
	readResult, err := io.ReadAll(res.Body)
	require.NoError(t, err, "response body read error")
	handlerRet := storage.ShortenedBatch{}

	err = json.Unmarshal(readResult, &handlerRet)
	require.NoError(t, err, "body unmarshal error")
	assert.Equal(t, putResult, handlerRet, "unexpected handler result")
}
