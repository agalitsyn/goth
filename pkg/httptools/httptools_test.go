package httptools

import "net/http"

func getTestHandlerBlah() http.HandlerFunc {
	fn := func(rw http.ResponseWriter, _ *http.Request) {
		_, _ = rw.Write([]byte("blah"))
	}
	return fn
}
