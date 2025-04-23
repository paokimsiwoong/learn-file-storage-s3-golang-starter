package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// .env의 변수들과 db client 담는 구조체
type apiConfig struct {
	db               database.Client
	jwtSecret        string
	platform         string
	filepathRoot     string
	assetsRoot       string
	s3Bucket         string
	s3Region         string
	s3CfDistribution string
	port             string
}

// 썸네일 데이터와 데이터 타입을 담는 구조체
type thumbnail struct {
	data      []byte
	mediaType string
}

// video의 uuid id와 썸네일을 연결하는 글로벌 맵
var videoThumbnails = map[uuid.UUID]thumbnail{}

func main() {
	// @@@ 환경변수, db 초기화 섹션 시작 @@@

	// godotenv.Load(filenames ...string) 함수 (파일이름 입력하지 않으면 기본값 .env 파일 로드)
	godotenv.Load(".env")

	// os.Getenv 함수로 환경변수를 불러올 수 있음
	pathToDB := os.Getenv("DB_PATH")
	if pathToDB == "" {
		log.Fatal("DB_URL must be set")
	}

	// database.NewClient는 *sql.DB를 필드로 가지는 Client 구조체 반환
	db, err := database.NewClient(pathToDB)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM environment variable is not set")
	}

	filepathRoot := os.Getenv("FILEPATH_ROOT")
	if filepathRoot == "" {
		log.Fatal("FILEPATH_ROOT environment variable is not set")
	}

	assetsRoot := os.Getenv("ASSETS_ROOT")
	if assetsRoot == "" {
		log.Fatal("ASSETS_ROOT environment variable is not set")
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		log.Fatal("S3_BUCKET environment variable is not set")
	}

	s3Region := os.Getenv("S3_REGION")
	if s3Region == "" {
		log.Fatal("S3_REGION environment variable is not set")
	}

	s3CfDistribution := os.Getenv("S3_CF_DISTRO")
	if s3CfDistribution == "" {
		log.Fatal("S3_CF_DISTRO environment variable is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}

	// 불러온 환경변수들, db 를 apiConfig 구조체에 저장
	cfg := apiConfig{
		db:               db,
		jwtSecret:        jwtSecret,
		platform:         platform,
		filepathRoot:     filepathRoot,
		assetsRoot:       assetsRoot,
		s3Bucket:         s3Bucket,
		s3Region:         s3Region,
		s3CfDistribution: s3CfDistribution,
		port:             port,
	}

	// cfg.ensureAssetsDir method는 assets_root 경로 디렉토리가 있는지 확인하고 없으면 디렉토리를 생성하는 함수
	err = cfg.ensureAssetsDir()
	if err != nil {
		log.Fatalf("Couldn't create assets directory: %v", err)
	}
	// @@@ 환경변수, db 초기화 섹션 종료 @@@

	// @@@ Routing 섹션 시작 @@@

	// http.NewServeMux 는 http 요청 멀티플렉서(multiplexer) 인스턴스를 생성하는 함수
	// http 요청 멀티플렉서 : 여러개의 URL 경로를 처리한다
	mux := http.NewServeMux()
	// 이 멀티플렉서가 처리할 URL 경로들을 아래에서 계속 추가한다

	// file 서버 핸들러 생성
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	// http.FileServer는 url path 부분과 서버 내부 filepathRoot를 연결
	// ex: url path = /app/test.png ==> filepathRoot/app/test.png
	// http.FileServer 함수의 인자는 http.FileSystem 인터페이스를 구현하는 변수여야 한다
	// filepathRoot는 단순 string이므로 http.FileSystem 인터페이스를 구현하는 http.Dir 타입으로 형변환

	// filepathRoot가 ./app 이므로 url path의 /app부분을 그대로 두면 /app/test.png ==> ./app/app/test.png 로 연결되어 문제발생
	// ====> http.StripPrefix함수는 지정한 prefix를 url path에서 제거한 후 fileserver에 나머지 path를 입력하는 handler를 만들어준다
	// =======> /app/test.png ==> ./app/test.png 로 제대로 연결

	// /app 경로에 file 서버 핸들러 연결
	mux.Handle("/app/", appHandler)

	// 위의 /app file server 생성의 경우와 거의 동일하게 /assets 경로에 file 서버 핸들러 연결하면서
	// 추가로 cacheMiddleware 적용
	assetsHandler := http.StripPrefix("/assets", http.FileServer(http.Dir(assetsRoot)))
	mux.Handle("/assets/", cacheMiddleware(assetsHandler))
	// cacheMiddleware는 response에 캐쉬 관련 헤더를 설정하도록 하는 middleware

	// api 계열 엔드포인트 handler 등록
	mux.HandleFunc("POST /api/login", cfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", cfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", cfg.handlerRevoke)

	mux.HandleFunc("POST /api/users", cfg.handlerUsersCreate)

	mux.HandleFunc("POST /api/videos", cfg.handlerVideoMetaCreate)
	mux.HandleFunc("POST /api/thumbnail_upload/{videoID}", cfg.handlerUploadThumbnail)
	mux.HandleFunc("POST /api/video_upload/{videoID}", cfg.handlerUploadVideo)
	mux.HandleFunc("GET /api/videos", cfg.handlerVideosRetrieve)
	mux.HandleFunc("GET /api/videos/{videoID}", cfg.handlerVideoGet)
	mux.HandleFunc("GET /api/thumbnails/{videoID}", cfg.handlerThumbnailGet)
	mux.HandleFunc("DELETE /api/videos/{videoID}", cfg.handlerVideoMetaDelete)

	mux.HandleFunc("POST /admin/reset", cfg.handlerReset)
	// @@@ Routing 섹션 종료 @@@

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on: http://localhost:%s/app/\n", port)
	log.Fatal(srv.ListenAndServe())
	// @@@ when ListenAndServe() is called, the main function blocks until the server is shut down
	// @@@ ListenAndServe 의 err는 항상 non nil ( After [Server.Shutdown] or [Server.Close], the returned error is [ErrServerClosed].)
}
