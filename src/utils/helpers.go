package utils

import (
	"net/http"
	"ricehub/src/errs"
	"strconv"

	"github.com/gin-gonic/gin"
)

// I know having such file is not ideal and should be avoided but for now
// I dont want to spend too much time trying to think where to place these things

// Tries to extract "limit" query param, if not available then uses default value of "10". After that it parses the value to int64 and validates it.
func ParseLimitQuery(c *gin.Context) (int64, error) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		return 0, errs.UserError("Invalid limit query argument provided", http.StatusBadRequest)
	}
	if limit <= 0 || limit >= 100 {
		return 0, errs.UserError("Incorrect limit provided! It must be between 1 and 99.", http.StatusBadRequest)
	}
	return limit, nil
}
