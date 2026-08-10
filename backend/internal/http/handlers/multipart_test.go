package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLimitMultipartBodyBoundsRequestBeforeParsing(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/upload",
		bytes.NewReader(bytes.Repeat([]byte{'x'}, int(multipartRequestOverheadBytes+5))),
	)

	limitMultipartBody(context, 4)
	_, err := io.ReadAll(context.Request.Body)
	var maxBytesError *http.MaxBytesError
	if !errors.As(err, &maxBytesError) {
		t.Fatalf("ReadAll error = %v, want http.MaxBytesError", err)
	}
}
