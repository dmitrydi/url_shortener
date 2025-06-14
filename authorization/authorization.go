package authorization

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type AuthStatus int

const (
	StatusOK            = 0
	StatusInvalidCookie = 1
	StatusUnauthorized  = 2
	StatusNotFound      = 3
)

const (
	AuthCookieName  = "authorization"
	secretKey       = "01234567890"
	cookieDelimiter = "."
)

type UserAuth struct {
	UID    uuid.UUID
	Status AuthStatus
}

func FromCookie(cookie *http.Cookie) UserAuth {
	parts := strings.Split(cookie.Value, cookieDelimiter)
	if len(parts) != 2 {
		log.Println("invalid parts num")
		return UserAuth{UID: uuid.Nil, Status: StatusInvalidCookie}
	}
	decodedValue, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		log.Println("could not decode name")
		return UserAuth{UID: uuid.Nil, Status: StatusInvalidCookie}
	}

	decodedSig, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		log.Println("coud not decode key")
		return UserAuth{UID: uuid.Nil, Status: StatusUnauthorized}
	}

	// Проверяем подпись
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(decodedValue)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(decodedSig, expectedSig) {
		return UserAuth{UID: uuid.Nil, Status: StatusUnauthorized}
	}

	id, err := uuid.Parse(string(decodedValue))
	if err != nil {
		log.Println("coud not parse uid")
		return UserAuth{UID: uuid.Nil, Status: StatusInvalidCookie}
	}
	return UserAuth{UID: id, Status: StatusOK}
}

func EncodeUID(uid uuid.UUID) (string, error) {
	sUID := uid.String()
	if len(sUID) == 0 {
		return "", errors.New("error encoding uid")
	}
	// Создаем подпись
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(sUID))
	signature := mac.Sum(nil)

	// Кодируем значение и подпись в base64 для cookie
	encodedValue := base64.URLEncoding.EncodeToString([]byte(sUID))
	encodedSig := base64.URLEncoding.EncodeToString(signature)

	cookieString := encodedValue + cookieDelimiter + encodedSig
	return cookieString, nil
}
