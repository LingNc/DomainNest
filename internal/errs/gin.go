package errs

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// JSONError writes a coded error response.
func JSONError(c *gin.Context, err error) {
	if ce, ok := IsCoded(err); ok {
		c.JSON(HTTPStatus(ce.Code), gin.H{
			"code":       HTTPStatus(ce.Code),
			"error_code": ce.Code,
			"params":     ce.Params,
			"message":    ce.Msg,
		})
		return
	}

	// Fallback for non-coded errors
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":       500,
		"error_code": InternalError,
		"message":    err.Error(),
	})
}

// JSONErrorCode writes a response with just a code (no underlying error).
func JSONErrorCode(c *gin.Context, code Code, params ...interface{}) {
	c.JSON(HTTPStatus(code), gin.H{
		"code":       HTTPStatus(code),
		"error_code": code,
		"params":     params,
		"message":    string(code),
	})
}

// JSONOK writes a success response (no error_code field).
func JSONOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data":    data,
	})
}