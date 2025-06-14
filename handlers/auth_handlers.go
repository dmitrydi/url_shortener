package handlers

import (
	"net/http"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/storage"
)

type CookieNotFound struct {
	Text string
}

func NewCookieNotFoundError(requiredCookieName string) *CookieNotFound {
	return &CookieNotFound{Text: requiredCookieName}
}

func (e *CookieNotFound) Error() string {
	return e.Text
}

func GetUserAuthorization(r *http.Request, st storage.URLStorage) (authorization.UserAuth, error) {
	cookie, err := r.Cookie(authorization.AuthCookieName)
	if err != nil {
		return authorization.UserAuth{}, NewCookieNotFoundError(authorization.AuthCookieName)
	}
	au := authorization.FromCookie(cookie)
	if au.Status == authorization.StatusOK {
		found, err := st.Contains(r.Context(), au.UID)
		if err != nil {
			return authorization.UserAuth{}, err
		}
		if !found {
			return authorization.UserAuth{UID: au.UID, Status: authorization.StatusNotFound}, nil
		}
	}
	return au, nil
}

type WithAuthHandler func(http.ResponseWriter, *http.Request, storage.URLStorage, authorization.UserAuth)

func WithAuthHandlerWrapper(next WithAuthHandler, st storage.URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ua, err := GetUserAuthorization(r, st)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		next(w, r, st, ua)
	}
}
