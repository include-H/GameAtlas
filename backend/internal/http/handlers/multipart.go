package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Multipart parsing happens before the service can inspect the uploaded file.
// Keep a small allowance for boundaries and form fields while bounding the
// request body to the file limit.
const multipartRequestOverheadBytes int64 = 1 << 20
const multipartMaxMemoryBytes int64 = 32 << 20

func limitMultipartBody(c *gin.Context, maxFileBytes int64) {
	if c == nil || c.Request == nil || c.Request.Body == nil || maxFileBytes <= 0 {
		return
	}
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		maxFileBytes+multipartRequestOverheadBytes,
	)
}

func parseMultipartForm(c *gin.Context) error {
	if c == nil || c.Request == nil {
		return errors.New("missing multipart request")
	}
	return c.Request.ParseMultipartForm(multipartMaxMemoryBytes)
}

func writeMultipartParseError(c *gin.Context, err error, message string) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		writeJSONError(c, http.StatusRequestEntityTooLarge, "上传文件过大")
		return
	}
	writeJSONError(c, http.StatusBadRequest, message)
}
