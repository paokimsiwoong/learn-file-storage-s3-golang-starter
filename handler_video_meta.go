package main

import (
	"encoding/json"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerVideoMetaCreate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		database.CreateVideoParams
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}
	params.UserID = userID

	video, err := cfg.db.CreateVideo(params.CreateVideoParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create video", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, video)
}

func (cfg *apiConfig) handlerVideoMetaDelete(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusForbidden, "You can't delete this video", err)
		return
	}

	err = cfg.db.DeleteVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't delete video", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerVideoGet(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't get video", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)

	// @@@ cloud front 사용하면서 signed url 미사용
	// // dbVideoToSignedVideo 메소드는
	// // VideoURL 필드에 변환된 presigned URL을 담은 새 database.Video 구조체를 반환
	// // @@@ video.VideoURL은 "<bucketName>,<fileName>" 형태
	// presignedVideo, err := cfg.dbVideoToSignedVideo(video)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Unable to create presigned url", err)
	// 	return
	// }

	// // 반환되는 presignedVideo는 db와 다르게 VideoURL 필드가 presigned URL
	// respondWithJSON(w, http.StatusOK, presignedVideo)
}

func (cfg *apiConfig) handlerVideosRetrieve(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	videos, err := cfg.db.GetVideos(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve videos", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videos)

	// @@@ cloud front 사용하면서 signed url 미사용
	// // 기존의 "<bucketName>,<fileName>" 형태를 VideoURL 필드에 가진 video들의 슬라이스 videos 대신
	// // VideoURL 필드에 presigned url을 저장한 presigned video를 담을 새로운 슬라이스
	// presignedVideos := make([]database.Video, len(videos))

	// for _, video := range videos {
	// 	presignedVideo, err := cfg.dbVideoToSignedVideo(video)
	// 	if err != nil {
	// 		respondWithError(w, http.StatusInternalServerError, "Unable to create presigned url", err)
	// 		return
	// 	}

	// 	presignedVideos = append(presignedVideos, presignedVideo)
	// }

	// respondWithJSON(w, http.StatusOK, presignedVideos)
}
