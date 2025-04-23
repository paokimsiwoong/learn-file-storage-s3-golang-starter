package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TokenTypeAccess(TokenType)은 MakeJWT, ValidateJW의 jwt.Claims 인터페이스를 구현하는 구조체 Issuer 필드에 사용됨
type TokenType string

const (
	TokenTypeAccess TokenType = "tubely-access"
)

var ErrNoAuthHeaderIncluded = errors.New("no auth header included in request")

// 암호를 받아서 hash로 변환해주는 함수
func HashPassword(password string) (string, error) {
	dat, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	// 두번째 인자 cost가 높을 수록 해쉬 처리 단계가 늘어나 뚫기 어려워진다 @@@ 기본값 10은 bcrypt.DefaultCost 로 입력 가능
	if err != nil {
		return "", err
	}
	return string(dat), nil
}

// hash 와 입력된 암호를 비교하는 함수 nil이면 암호일치, nil이 아니면 불일치
func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// JWT(JSON Web Token) 생성함수
func MakeJWT(
	userID uuid.UUID,
	tokenSecret string,
	expiresIn time.Duration,
) (string, error) {
	signingKey := []byte(tokenSecret)

	// JWT 토큰 생성
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    string(TokenTypeAccess),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()), // jwt.NewNumericDate 함수는 time.Time을 담는 jwt.NumericDate 구조체의 포인터를 반환
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	})

	// 토큰 생성 시에 유저가 제공한 tokenSecret을 같이 사용해서 생성한다.
	return token.SignedString(signingKey)
	// HS256은 key에 []byte 타입 입력해야한다
	// https://golang-jwt.github.io/jwt/usage/signing_methods/#signing-methods-and-key-types
}

// JWT 검증함수, 검증 후 userID(uuid.UUID) 반환
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	// MakeJWT 함수에서 사용한 jwt.Claims 구현 타입을 그대로 사용
	claimsStruct := jwt.RegisteredClaims{}
	// ??? jwt.NewWithClaims로 생성된 token은 Claims 필드에 함수 인자로 제공된 claim이 저장되고
	// jwt.ParseWithClaims는 tokenString으로부터 token(*jwt.Token)을 다시 얻어내는 과정에서 token의 필드 Claims도 복원되는데
	// 이때 복호화(decode)된 데이터들을 다시 담을 claim은 생성 당시 claim과 동일한 구조체여야 한다
	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
	// claims := jwt.MapClaims{} // type MapClaims map[string]interface{}
	// MapClaims is a claims type that uses the map[string]interface{} for JSON decoding
	// @@@ 만약 토큰 생성시의 claim 구조체의 구조(JSON 구조)를 모를 경우
	// @@@ 어떠한 형태건 받아들일 수 있는 map[string]interface{} 사용가능
	// @@@ map[json_key]json_value 형태, 임의의 모든 타입은 interface{} 구현
	// @@@ jwt.MapClaims{}을 쓸경우 ParseWithClaims에 인자로 입력할 때 & 있거나 없거나 문제없음
	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@

	// tokenString decode해서 token(*jwt.Token) 반환
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(token *jwt.Token) (interface{}, error) { return []byte(tokenSecret), nil },
	)
	// @@@ 두번째인자 claims는 jwt.MapClaims{}를 쓰지않는 경우 pointer를 입력해야 에러가 안난다.
	// 3번째 인자 keyFunc는 tokenSecret 처리에 쓰이는 함수로
	// 그냥 원본 그대로 사용시에는 함수 시그니처 만족하면서 []byte(tokenSecret) 반환하는 함수를 인자로 입력하면 된다.
	if err != nil { // 토큰이 invalid하거나 expired일 경우 err != nil
		return uuid.Nil, err
	}

	// 토큰에 저장된 userID는 Claims의 Subject 필드에 저장되어 있음 (string 상태)
	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	// 부정토큰(토큰 정규발급자가 아닌 자가 위조한 토큰)을 걸러내기 - Issuer 비교
	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, err
	}
	if issuer != string(TokenTypeAccess) {
		return uuid.Nil, errors.New("invalid issuer")
	}

	// string화된 uuid를 uuid.UUID로 변환
	id, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
	}
	return id, nil
}

// Authorization header에 들어있는 인증 정보에서 tokenString만 추출해서 반환하는 함수
func GetBearerToken(headers http.Header) (string, error) {
	// http.Header에서 Get으로 정보 불러오기
	authHeader := headers.Get("Authorization")
	// authorization 헤더 내용은 "Bearer <tokenString>" 형태
	if authHeader == "" {
		return "", ErrNoAuthHeaderIncluded
	}
	splitAuth := strings.Split(authHeader, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "Bearer" {
		return "", errors.New("malformed authorization header")
	}

	return splitAuth[1], nil
}

// refresh token 생성 함수 (access 토큰인 JWT와 다름)
func MakeRefreshToken() (string, error) {
	token := make([]byte, 32)
	// crypto/rand.Read함수는 입력한 []byte에 랜덤 값 채워주는 함수
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
	// hex.EncodeToString함수는 input을 hexadecimal encoding 한 후 반환
}

// Authorization header에 들어있는 인증 정보에서 APIKey만 추출해서 반환하는 함수
func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	// authorization 헤더 내용은 " ApiKey <THE_KEY_HERE>" 형태
	if authHeader == "" {
		return "", ErrNoAuthHeaderIncluded
	}
	splitAuth := strings.Split(authHeader, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "ApiKey" {
		return "", errors.New("malformed authorization header")
	}

	return splitAuth[1], nil
}
