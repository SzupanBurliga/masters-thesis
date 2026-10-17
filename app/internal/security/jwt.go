package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWT struct{ secret []byte }

func NewJWT(secret string) *JWT { return &JWT{secret: []byte(secret)} }

func (j *JWT) Sign(sub string, ttl time.Duration) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(ttl).Unix(),
	}).SignedString(j.secret)
}

// Verify sprawdza podpis, algorytm (HS256) i exp; zwraca sub.
func (j *JWT) Verify(tok string) (string, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tok, claims, func(*jwt.Token) (any, error) { return j.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return "", err
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", errors.New("missing sub")
	}
	return sub, nil
}
