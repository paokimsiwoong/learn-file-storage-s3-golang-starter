package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

// POST /api/video_upload/{videoID} handler : 전달 받은 비디오 파일을 s3에 저장
func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// request 바디 최대 용량 제한 (1 << 30은 1 * 2^30 즉 1GB)
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
	// 이 용량 제한을 넘으면 내부적으로 MaxBytesError 발생
	// 이 에러는 io.ReadAll(r.Body)나 r.ParseMultipartForm(~)가 반환하면서 서버가 연결을 종료한다

	// r.PathValue(path parameter 이름)로 videoID 가져오고
	// // videoID는 video 메타데이터를 담은 database.Video의 프라이머리 키 id
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

	fmt.Println("uploading video file to s3 for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	// << 20는 bit shift to left를 20번 시행한다는 의미
	//    EX: 00010111 는 10진법으로 23 => bit shift to left를 한번하면
	//    00101110가 된다 10진법으로는 46 ==> *2 와 동일한 결과
	// ====> << 20은 * 2^20 과 동일한 결과
	// https://en.wikipedia.org/wiki/Bitwise_operation#Bit_shifts
	// Bit shifting is a way to multiply by powers of 2. 10 << 20 is the same as 10 * 1024 * 1024, which is 10MB.

	// maxMemory(10MB)까지 메모리에 저장하고 초과분은 임시 파일로 저장
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		// 413 코드
		respondWithError(w, http.StatusRequestEntityTooLarge, "Video file is too big", err)
		return
	}
	// @@@ 해답은 r.ParseMultipartForm 부분 생략
	// @@@ => 생략하면 r.FormFile이 실행되면서 자동으로 ParseMultipartForm를 호출하긴 하지만
	// @@@ maxMemory 는 기본값 32MB이 적용되고 임시 파일로 저장되는 용량 제어도 불가능

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	// `file` is an `io.Reader` that we can read from to get the video data
	defer file.Close()

	// file이 무슨 파일인지 header에서 정보 가져오기 (이 header는 썸네일 파일의 헤더 *multipart.FileHeader)
	// @@@ Content-Type 헤더 안에 MIME type 형태로 데이터가 들어 있으므로 mime.ParseMediaType 사용
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid Content-Type", err)
		return
	}
	// 썸네일이 image/jpeg나 image/png이 아닌 경우 에러 예외 처리
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "video file type must be mp4", err)
		return
	}

	// 임시파일 생성
	tempFile, err := os.CreateTemp("", "tubely-upload_*.mp4")
	// dir은 ""로 두면 시스템 기본 임시파일 폴더에 저장
	// pattern으로 제공된 string 뒤에 임의의 문자열을 붙인 뒤 파일이름으로 사용
	// // pattern에 *이 포함되면 임의의 문자열을 pattern 마지막에 붙이는 대신
	// // 마지막 *을 임의의 문자열로 치환한 것을 파일이름으로 사용
	// @@@ 해답은 tempFile, err := os.CreateTemp("", "tubely-upload.mp4") 만 사용
	// @@@ ==> 파일이 tubely-upload.mp4로 생성되고 이미 존재할경우 tubely-upload.mp41 과 같이 뒤에 숫자를 붙여서 중복을 피한다
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temp file", err)
		return
	}

	// 임시 파일 삭제를 defer 걸어두기
	defer os.Remove(tempFile.Name())
	// 임시 파일 Close defer 해서 메모리 누수 방지
	defer tempFile.Close()
	// @@@ defer는 LIFO
	// // @@@ 따라서 tempFile.Close()가 먼저 실행되고 그 다음에 os.Remove가 실행된다

	// 비디오 데이터를 임시파일로 복사
	_, err = io.Copy(tempFile, file)
	// file multipart.File은 io.Reader 인터페이스를 구현하고
	// tempFile *os.File은 io.Writer 인터페이스를 구현(Write 함수)
	// 복사 후 몇 바이트를 복사했는지 반환하는 것은 필요 없으므로 _ 처리
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy temp file", err)
		return
	}

	// 임시파일을 ffprobe명령어로 살펴보고 화면비를 얻기
	// 반드시 io.Copy 뒤에 있어야 실제로 디스크에 저장된 임시파일의 데이터를 가져올 수 있음
	videoAspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to compute aspect ratio", err)
		return
	}

	// tempFile은 io.Copy로 데이터를 복사받으면서 내부 포인터가 복사받은 데이터 끝부분을 가리키고 있음
	// ==> Seek(0, io.SeekStart)로 데이터 첫부분을 가리키도록 변경
	// // @@@ tempFile을 다시 사용해 s3에 복사하므로 포인터가 데이터 첫부분을 가리켜야 한다
	tempFile.Seek(0, io.SeekStart)
	// 두번째 인자 whence가 0(io.SeekStart)이면 파일 첫부분, 1는 현재 포인터 위치, 2는 파일 끝부분
	// 첫번째 인자 offset은 whence에서 지정한 위치에 offset만큼 포인터 위치 이동
	// @@@ 해답은
	// _, err = tempFile.Seek(0, io.SeekStart)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Could not reset file pointer", err)
	// 	return
	// } 로 에러처리 하고 있음

	// @@@ s3에 파일 업로드 @@@

	// 업로드에 필요한 정보를 담은 s3.PutObjectInput 구조체 생성
	fileName := getS3AssetPath(mediaType, videoAspectRatio)
	// 파일이름은 <prefix>/<randName>.<file_extension> 형태
	putObjectInput := s3.PutObjectInput{
		// Bucket: &cfg.s3Bucket, // bucket 이름은 cfg.s3Bucket 또는 .env에 S3_BUCKET로 저장되어 있음
		// @@@ 해답처럼 aws.String 활용하기
		Bucket: aws.String(cfg.s3Bucket),
		// Key:    &fileName,
		Key:  aws.String(fileName),
		Body: tempFile,
		// tempFile은 io.Reader 인터페이스도 구현(Read함수)
		// ContentType: &mediaType,
		ContentType: aws.String(mediaType),
	}
	// s3.Client의 PutObject 메소드로 s3에 파일 업로드
	_, err = cfg.s3Client.PutObject(r.Context(), &putObjectInput)
	// @@@ 해답은 &s3.PutObjectInput{~~}로 함수 인자 입력안에서 구조체 바로 생성하고 그 포인터 입력
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upload the file to S3", err)
		return
	}

	newVideoURL := cfg.getObjectURL(fileName)

	video.VideoURL = &newVideoURL

	// 갱신된 video를 db에 입력해 db 갱신
	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update the video's metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
