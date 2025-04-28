package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

// POST /api/thumbnail_upload/{videoID} handler : 전달 받은 썸네일을 db에 저장
func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	// r.PathValue(path parameter 이름)로 videoID 가져오고
	videoIDString := r.PathValue("videoID")
	// string 형태인 uuid를 uuid.Parse함수로 uuid.UUID 타입으로 변환
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// jWT sting이 Authorization header에 저장되어 있는지 확인
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	// JWT 검증
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	// << 20는 bit shift to left를 20번 시행한다는 의미
	//    EX: 00010111 는 10진법으로 23 => bit shift to left를 한번하면
	//    00101110가 된다 10진법으로는 46 ==> *2 와 동일한 결과
	// ====> << 20은 * 2^20 과 동일한 결과
	// https://en.wikipedia.org/wiki/Bitwise_operation#Bit_shifts
	// Bit shifting is a way to multiply by powers of 2. 10 << 20 is the same as 10 * 1024 * 1024, which is 10MB.
	r.ParseMultipartForm(maxMemory)

	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	// `file` is an `io.Reader` that we can read from to get the image data
	defer file.Close()

	// file이 무슨 파일인지 header에서 정보 가져오기 (이 header는 썸네일 파일의 헤더 *multipart.FileHeader)
	mediaType := header.Header.Get("Content-Type")
	// @@@ 해답처럼 헤더가 제대로 설정되지 않은 경우 예외 처리
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
		return
	}

	// io.ReadAll은 io.Reader를 입력으로 받아 []byte, error 출력
	data, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to read file", err)
		return
	}

	// db에서 videoID로 해당 video 메타데이터를 담은 database.Video 불러오기
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get the video's metadata", err)
		return
	}

	// 만약 video 원 업로더 아이디와 현재 아이디가 일치하지 않는 경우 에러
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not the owner of the video", errors.New("not the owner of the video"))
		return
	}

	// @@@ base64 도입 후 글로벌 맵 삭제
	// // 썸네일 데이터를 담는 구조체 생성
	// newThumbnail := thumbnail{
	// 	data:      data,
	// 	mediaType: mediaType,
	// }
	// @@@ base64 도입 후 글로벌 맵 삭제
	// // videoThumbnails는 map[uuid.UUID]thumbnail 인 글로벌 맵
	// videoThumbnails[videoID] = newThumbnail

	// []byte를 base64 encoding
	encoded := base64.StdEncoding.EncodeToString(data)

	// 썸네일 url 생성
	newThumbnailURL := fmt.Sprintf("data:%s;base64,%s", mediaType, encoded)

	// video의 ThumbnailURL 필드 갱신
	video.ThumbnailURL = &newThumbnailURL

	// 갱신된 video를 db에 입력해 db 갱신
	if err := cfg.db.UpdateVideo(video); err != nil {
		// @@@ 해답처럼 map에 이미 추가된 videoID key를 다시 삭제해주어야 한다
		// @@@ (∵ 썸네일 생성이 실패했으므로)
		// delete(videoThumbnails, videoID) // @@@ base64 도입 후 글로벌 맵 삭제
		// @@@
		respondWithError(w, http.StatusInternalServerError, "Unable to update the video's metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
