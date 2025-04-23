package main

import (
	"net/http"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
)

// POST /api/refresh handler : valid한 refresh token이 있으면 새로운 1시간짜리 jwt 생성
func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	// refresh token이 Authorization header에 저장되어 있는지 확인
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", err)
		return
	}

	// db에서 refresh token으로 user 찾기 (users.id와 refresh_tokens.user_id INNER JOIN)
	user, err := cfg.db.GetUserByRefreshToken(refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", err)
		return
	} // 현재 쿼리는 토큰 만료, 파기 여부 확인하지 않고 있음

	// 새로 발급할 1시간짜리 JWT 생성
	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		time.Hour,
	)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Token: accessToken,
	})
}

// POST /api/revoke handler : refresh 토큰 revoke
func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	// refresh token이 Authorization header에 저장되어 있는지 확인
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", err)
		return
	}

	// db의 해당 refresh token의 revoked_at 필드 업데이트
	err = cfg.db.RevokeRefreshToken(refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't revoke session", err)
		return
	} // updated_at도 수정할 필요가 있어보이나 현재 쿼리는 updated_at은 수정하지 않고 있음

	w.WriteHeader(http.StatusNoContent)
}
