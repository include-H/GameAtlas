package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDecodeJSONStrictBoundsRequestBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	body := bytes.NewBufferString(`{"content":"`)
	body.Write(bytes.Repeat([]byte{'x'}, int(maxJSONBodyBytes)))
	body.WriteString(`"}`)
	context.Request = httptest.NewRequest(http.MethodPost, "/json", body)

	var request struct {
		Content string `json:"content"`
	}
	err := decodeJSONStrict(context, &request)
	var maxBytesError *http.MaxBytesError
	if !errors.As(err, &maxBytesError) {
		t.Fatalf("decodeJSONStrict error = %v, want http.MaxBytesError", err)
	}
}
