package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const minSecretKeySize = 32

var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

type Payload struct {
	ID        uuid.UUID `json:"id"`
	UserID    int64     `json:"user_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

func NewPayload(userID int64, duration time.Duration) (*Payload, error) {
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	payload := &Payload{
		ID:        tokenID,
		UserID:    userID,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}
	return payload, nil
}

func (p *Payload) Valid() error {
	if time.Now().After(p.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}

type Maker interface {
	CreateToken(userID int64, duration time.Duration) (string, error)
	VerifyToken(token string) (*Payload, error)
}

type JWTMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, errors.New("invalid key size: must be at least 32 characters")
	}
	return &JWTMaker{secretKey}, nil
}

func (m *JWTMaker) CreateToken(userID int64, duration time.Duration) (string, error) {
	payload, err := NewPayload(userID, duration)
	if err != nil {
		return "", err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return jwtToken.SignedString([]byte(m.secretKey))
}

func (m *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (any, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}
		return []byte(m.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrInvalidToken
	}

	return payload, nil
}

func (p *Payload) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(p.ExpiredAt), nil
}
func (p *Payload) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(p.IssuedAt), nil
}
func (p *Payload) GetNotBefore() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(p.IssuedAt), nil
}
func (p *Payload) GetIssuer() (string, error) {
	return "", nil
}
func (p *Payload) GetSubject() (string, error) {
	return "", nil
}
func (p *Payload) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil
}
