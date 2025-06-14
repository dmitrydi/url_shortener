package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/dmitrydi/url_shortener/authorization"
	"github.com/dmitrydi/url_shortener/storage"
	"github.com/google/uuid"
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
		var cookieError *CookieNotFound
		log.Println("AuthStatus ", ua.Status)
		if err != nil {
			log.Println("Auth error ", err)
		}
		if errors.As(err, &cookieError) || ua.Status == authorization.StatusNotFound {
			ua.UID = uuid.New()
			cookieString, err := authorization.EncodeUID(ua.UID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cookie := &http.Cookie{Name: authorization.AuthCookieName, Value: cookieString}
			http.SetCookie(w, cookie)
		}
		next(w, r, st, ua)
	}
}
