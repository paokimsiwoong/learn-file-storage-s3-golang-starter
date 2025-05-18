package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

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
	// @@@ Content-Type 헤더 안에 MIME type 형태로 데이터가 들어 있으므로 mime.ParseMediaType 사용
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid Content-Type", err)
		return
	}
	// 썸네일이 image/jpeg나 image/png이 아닌 경우 에러 예외 처리
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "thumbnail file type must be either jpeg or png", err)
		return
	}

	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
	// @@@ io.Copy 사용하면서 io.ReadAll 사용 안함
	// io.ReadAll은 io.Reader를 입력으로 받아 []byte, error 출력
	// file은 multipart.File 인터페이스를 구현하고 multipart.File의 구현 조건에는 io.Reader 인터페이스 구현 조건이 포함된다
	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Unable to read file", err)
	// 	return
	// }
	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@

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

	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
	// @@@ base64 도입 후 글로벌 맵 삭제
	// // 썸네일 데이터를 담는 구조체 생성
	// newThumbnail := thumbnail{
	// 	data:      data,
	// 	mediaType: mediaType,
	// }
	// @@@ base64 도입 후 글로벌 맵 삭제
	// // videoThumbnails는 map[uuid.UUID]thumbnail 인 글로벌 맵
	// videoThumbnails[videoID] = newThumbnail

	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
	// []byte를 base64 encoding
	// encoded := base64.StdEncoding.EncodeToString(data)
	// // @@@ 느린 base64대신 file system 사용하도록 변경
	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@

	// mediaType은 image/<확장자> 형태이므로 확장자만 가져오기
	// assetExtension := strings.Split(mediaType, "/")[1]
	// @@@ assts.go의 getAssetPath 함수 대신 사용

	// 파일 이름은 <videoID>.<file_extension> @@@ 매번 다른 경로가 되도록 videoID 대신 랜덤 생성값 사용하도록 변경
	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
	// @@@ 해답처럼 random 값 생성을 getAssetPath 함수 안으로 집어 넣도록 수정
	// 32 bytes 슬라이스 생성
	// randBytes := make([]byte, 32)
	// // crypto/rand.Read함수는 입력한 []byte에 랜덤 값 채워주는 함수
	// _, err = rand.Read(randBytes)
	// if err != nil { // @@@ Read fills b with cryptographically secure random bytes. It never returns an error, and always fills b entirely.
	// 	respondWithError(w, http.StatusInternalServerError, "Unable to create random bytes", err)
	// 	return
	// }
	// randName := base64.RawURLEncoding.EncodeToString(randBytes)
	// assetName := getAssetPath(randName, mediaType)
	assetName := getAssetPath(mediaType)
	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@

	// ./assets 과 <videoID>.<file_extension> 를 filepath.join으로 경로 합치기
	// ==> ./assets/<videoID>.<file_extension>
	// assetPath := filepath.Join(cfg.assetsRoot, assetName)
	// @@@ assets.go의 cfg.getAssetDiskPath 대신 사용
	assetPath := cfg.getAssetDiskPath(assetName)

	// asset 경로에 빈 파일 컨테이너 생성
	assetFile, err := os.Create(assetPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create asset file", err)
		return
	}
	// @@@ 해답처럼 *os.File 도 Close를 defer로 걸어두기
	defer assetFile.Close()

	// 썸네일 데이터를 asset 경로에 복사
	_, err = io.Copy(assetFile, file)
	// file multipart.File은 io.Reader 인터페이스를 구현하고
	// assetFile *os.File은 io.Writer 인터페이스를 구현
	// 복사 후 몇 바이트를 복사했는지 반환하는 것은 필요 없으므로 _ 처리
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy asset file", err)
		return
	}

	// 썸네일 url 생성
	// newThumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetName)
	// @@@ assets.go의 cfg.getAssetURL 대신 사용
	newThumbnailURL := cfg.getAssetURL(assetName)

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
