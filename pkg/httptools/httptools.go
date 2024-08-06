package httptools

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

func FileServerHandlerFunc(embedFiles embed.FS, staticFolder string) http.HandlerFunc {
	staticFS, err := fs.Sub(embedFiles, staticFolder) // error is always nil
	if err != nil {
		panic(err) // should never happen we load from embedded FS
	}
	return func(w http.ResponseWriter, r *http.Request) {
		webFS := http.StripPrefix(fmt.Sprintf("/%s/", staticFolder), http.FileServer(http.FS(staticFS)))
		webFS.ServeHTTP(w, r)
	}
}
