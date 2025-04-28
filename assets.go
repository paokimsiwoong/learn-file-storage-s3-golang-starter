package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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
func getAssetPath(videoID uuid.UUID, mediaType string) string {
	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s%s", videoID, ext)
}

// cfg.assetsRoot와 getAssetPath 함수로 만든 파일이름을 합쳐 경로 string을 반환하는 apiConfig method
func (cfg apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

// db에 저장될 썸네일 url 생성하는 apiConfig method
func (cfg apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

// Content-Type안에 들어 있는 Mime Type이 image/<확장자> 형태이므로 확장자만 가져오는 함수
func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

// @@@ 해답 예시 : asset 파일 생성 관련 유틸들
