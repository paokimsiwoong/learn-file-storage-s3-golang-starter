package main

import (
	"os"
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
