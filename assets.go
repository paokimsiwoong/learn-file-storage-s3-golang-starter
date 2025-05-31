package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// assets_root 경로 디렉토리가 있는지 확인하고 없으면 디렉토리를 생성하는 함수
// ==> 이미 있거나 정상적으로 생성한 경우 nil 반환
func (cfg apiConfig) ensureAssetsDir() error {
	//
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		// os.Mkdir 두번째 인자는 permission
		return os.Mkdir(cfg.assetsRoot, 0755)
		// x는 1 w는 2 r은 4 ===> 0755는 -rwxr-xr-x
	}
	return nil
}

// @@@ 해답 예시 : asset 파일 생성 관련 유틸들

// uuid, Content-Type을 받아 <videoID>.<file_extension> 형태의 string 반환하는 함수
// func getAssetPath(videoID uuid.UUID, mediaType string) string {
// 	ext := mediaTypeToExt(mediaType)
// 	return fmt.Sprintf("%s%s", videoID, ext)
// }

// Content-Type을 받아 랜덤 생성한 파일 이름으로 <randName>.<file_extension> 형태의 string 반환하는 함수
func getAssetPath(mediaType string) string {
	// @@@ 해답 예시처럼 랜덤 값 생성 여기서 하기
	// 32 bytes 슬라이스 생성
	randBytes := make([]byte, 32)
	// crypto/rand.Read함수는 입력한 []byte에 랜덤 값 채워주는 함수
	_, _ = rand.Read(randBytes)
	// @@@ Read fills b with cryptographically secure random bytes. It never returns an error, and always fills b entirely.
	randName := base64.RawURLEncoding.EncodeToString(randBytes)
	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", randName, ext)
}

// s3 버켓에 저장되는 파일이름을 반환하는 함수. landscape, portrait, other 3가지의 prefix 사용해서
// <prefix>/<randName>.<file_extension> 형태의 string 반환
func getS3AssetPath(mediaType string, aspectRatio string) string {
	// @@@ 해답 예시처럼 랜덤 값 생성 여기서 하기
	// 32 bytes 슬라이스 생성
	randBytes := make([]byte, 32)
	// crypto/rand.Read함수는 입력한 []byte에 랜덤 값 채워주는 함수
	_, _ = rand.Read(randBytes)
	// @@@ Read fills b with cryptographically secure random bytes. It never returns an error, and always fills b entirely.
	randName := base64.RawURLEncoding.EncodeToString(randBytes)
	ext := mediaTypeToExt(mediaType)

	return fmt.Sprintf("%s/%s%s", aspectRatio, randName, ext)
}

// cfg.assetsRoot와 getAssetPath 함수로 만든 파일이름을 합쳐 경로 string을 반환하는 apiConfig method
func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

// db에 저장될 썸네일 url 생성하는 apiConfig method
func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

// s3에 저장되는 객체의 url 생성하는 apiConfig method
func (cfg apiConfig) getObjectURL(fileName string) string {
	// https://<bucket-name>.s3.<region>.amazonaws.com/<key> 형태
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fileName)
}

// Content-Type안에 들어 있는 Mime Type이 image/<확장자> 형태이므로 확장자만 가져오는 함수
func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

// 영상파일 파일경로를 받아 화면비를(16:9, 9:16, other 중 하나) 반환하는 함수
// @@@ 문제 지시사항과 다르게 바로 landscape, portrait, other를 반환하도록 변경
func getVideoAspectRatio(filePath string) (string, error) {
	// @@@@@@ ffprobeResult 구조체 정리하기
	// ffprobe 출력 결과(json)를 담을 구조체
	type ffprobeResult struct {
		Streams []struct {
			Index              int    `json:"index"`
			CodecName          string `json:"codec_name,omitempty"`
			CodecLongName      string `json:"codec_long_name,omitempty"`
			Profile            string `json:"profile,omitempty"`
			CodecType          string `json:"codec_type"`
			CodecTagString     string `json:"codec_tag_string"`
			CodecTag           string `json:"codec_tag"`
			Width              int    `json:"width,omitempty"`
			Height             int    `json:"height,omitempty"`
			CodedWidth         int    `json:"coded_width,omitempty"`
			CodedHeight        int    `json:"coded_height,omitempty"`
			ClosedCaptions     int    `json:"closed_captions,omitempty"`
			FilmGrain          int    `json:"film_grain,omitempty"`
			HasBFrames         int    `json:"has_b_frames,omitempty"`
			SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
			DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
			PixFmt             string `json:"pix_fmt,omitempty"`
			Level              int    `json:"level,omitempty"`
			ColorRange         string `json:"color_range,omitempty"`
			ColorSpace         string `json:"color_space,omitempty"`
			ColorTransfer      string `json:"color_transfer,omitempty"`
			ColorPrimaries     string `json:"color_primaries,omitempty"`
			ChromaLocation     string `json:"chroma_location,omitempty"`
			FieldOrder         string `json:"field_order,omitempty"`
			Refs               int    `json:"refs,omitempty"`
			IsAvc              string `json:"is_avc,omitempty"`
			NalLengthSize      string `json:"nal_length_size,omitempty"`
			ID                 string `json:"id"`
			RFrameRate         string `json:"r_frame_rate"`
			AvgFrameRate       string `json:"avg_frame_rate"`
			TimeBase           string `json:"time_base"`
			StartPts           int    `json:"start_pts"`
			StartTime          string `json:"start_time"`
			DurationTs         int    `json:"duration_ts"`
			Duration           string `json:"duration"`
			BitRate            string `json:"bit_rate,omitempty"`
			BitsPerRawSample   string `json:"bits_per_raw_sample,omitempty"`
			NbFrames           string `json:"nb_frames"`
			ExtradataSize      int    `json:"extradata_size"`
			SampleFmt          string `json:"sample_fmt,omitempty"`
			SampleRate         string `json:"sample_rate,omitempty"`
			Channels           int    `json:"channels,omitempty"`
			ChannelLayout      string `json:"channel_layout,omitempty"`
			BitsPerSample      int    `json:"bits_per_sample,omitempty"`
			InitialPadding     int    `json:"initial_padding,omitempty"`
			Disposition        struct {
				Default         int `json:"default"`
				Dub             int `json:"dub"`
				Original        int `json:"original"`
				Comment         int `json:"comment"`
				Lyrics          int `json:"lyrics"`
				Karaoke         int `json:"karaoke"`
				Forced          int `json:"forced"`
				HearingImpaired int `json:"hearing_impaired"`
				VisualImpaired  int `json:"visual_impaired"`
				CleanEffects    int `json:"clean_effects"`
				AttachedPic     int `json:"attached_pic"`
				TimedThumbnails int `json:"timed_thumbnails"`
				NonDiegetic     int `json:"non_diegetic"`
				Captions        int `json:"captions"`
				Descriptions    int `json:"descriptions"`
				Metadata        int `json:"metadata"`
				Dependent       int `json:"dependent"`
				StillImage      int `json:"still_image"`
			} `json:"disposition"`
			Tags struct {
				Language    string `json:"language"`
				HandlerName string `json:"handler_name"`
				VendorID    string `json:"vendor_id"`
				Encoder     string `json:"encoder"`
				Timecode    string `json:"timecode"`
			} `json:"tags,omitempty"`
		} `json:"streams"`
	}

	// exec.Command는 명령어 이름과 args를 저장하는 exec.Cmd 구조체의 포인터를 반환
	// 명령어 실행은 반환된 Cmd의 Run 메소드를 사용해야 한다
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	// cmd의 Stdout 필드에는 출력 결과를 담을 컨테이너의 포인터를 저장
	var out bytes.Buffer
	cmd.Stdout = &out

	// 명령어 실행
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running ffprobe command: %w", err)
	}
	// 실행 완료 후에는 out에 출력 결과가 저장된다

	// 출력 결과를 구조체에 decoding
	var result ffprobeResult
	// @@@ bytes.Buffer의 Read 메소드는 func (b *bytes.Buffer) Read(p []byte) (n int, err error)로
	// @@@ 포인터 리시버(b *bytes.Buffer)를 가진다 ==> 따라서 io.Reader를 구현하는 것은 bytes.Buffer가 아니라
	// @@@ *bytes.Buffer이다 ==> 함수에서 io.Reader 인터페이스 구현한 타입인지 체크할 때는 반드시 *bytes.Buffer를 입력
	// // @@@ 포인터가 아닌 밸류(bytes.Buffer)여도 Read 메소드 실행은 가능하지만
	// // @@@ 이 때는 Go 컴파일러가 자동으로 out.Read(p)를 (&out).Read(p)로 변환해주기 때문
	if err := json.NewDecoder(&out).Decode(&result); err != nil {
		return "", fmt.Errorf("error decoding ffprobe's stdout: %w", err)
	}

	// 화면비 계산
	width := result.Streams[0].Width
	height := result.Streams[0].Height

	standardRatio := 16.0 / 9.0

	if width >= height {
		ratio := float64(width) / float64(height)
		if ratio >= standardRatio-0.05 && ratio <= standardRatio+0.05 {
			// return "16:9", nil
			return "landscape", nil
			// getS3AssetPath에서 16:9대신 landscape를 prefix로 사용하므로 바로 landscape를 저장하도록 변경
		}
	} else {
		ratio := float64(height) / float64(width)
		if ratio >= standardRatio-0.05 && ratio <= standardRatio+0.05 {
			// return "9:16", nil
			return "portrait", nil
			// getS3AssetPath에서 9:16대신 portrait를 prefix로 사용하므로 바로 landscape를 저장하도록 변경
		}
	}

	return "other", nil
}
