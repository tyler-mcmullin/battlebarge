package routes_test

import (
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

func serve(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
