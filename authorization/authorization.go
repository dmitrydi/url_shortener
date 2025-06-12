package authorization

import (
	"net/http"

	"github.com/google/uuid"
)

type AuthStatus int

const (
	StatusAuthorized = 0
	StatusNotFound   = 1
	StatusBadKey     = 2
)

type IDStatus struct {
	UID    uuid.UUID
	Status AuthStatus
}

var secretKey = []byte("cookie-secret-key")
var cookieName = "shortener-id"

func UUIDFromCookie(c *http.Cookie) (uuid.UUID, error) {
	return uuid.Parse(c.Value)
}

func GetUserIdStatus(r *http.Request) IDStatus {
	return IDStatus{UID: uuid.New(), Status: StatusAuthorized}
}

func GetUserId(r *http.Request) uuid.UUID {
	for _, c := range r.Cookies() {
		if c.Name == cookieName {
			id, err := UUIDFromCookie(c)
			if err != nil {
				return uuid.Nil
			}
			return id
		}
	}
	return uuid.Nil
}

func CreateUserId() uuid.UUID {
	return uuid.New()
}

func SetCookie(w http.ResponseWriter, cookie *http.Cookie) {
	http.SetCookie(w, cookie)
}

func SetID(cookie *http.Cookie, uid uuid.UUID) {

}
