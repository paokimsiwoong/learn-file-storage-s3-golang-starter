package main

import "net/http"

// 이 middleware로 http.Handler를 감싸는 새로운 http.Handler 반환
// 이 새 http.Handler는 원본 http.Handler의 ServeHTTP메소드를 그대로 호출하면서
// 추가로 원본 호출 직전에 http.ResponseWriter의 cache관련 header를 설정한다
func noCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// w.Header().Set("Cache-Control", "max-age=3600")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
	// @@@@@ type HandlerFunc func(ResponseWriter, *Request)은 일반 함수를 http.Handler로 취급할 수 있게 해주는 일종의 툴
	// @@@@@ http.HandlerFunc(함수)로 함수를 HandlerFunc 타입으로 형변환을 하면 이 HandlerFunc 타입은 ServeHTTP 메소드를 가지고 있으므로 http.Handler 인터페이스를 구현한다
	// @@@@@ 단 함수 시그니처가 func(ResponseWriter, *Request)여야 한다
	// @@@@@@@@ HandlerFunc 타입의 ServeHTTP 메소드 : func (f http.HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) { f(w,r) }
	// @@@@@@@@ ===> 단순히 HandlerFunc 타입인 함수 자기 자신을 호출
	// https://pkg.go.dev/net/http#HandlerFunc
}
