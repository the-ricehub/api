package errs

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func logInternalError(logger *zap.Logger, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		logger.Error("Row not found error", zap.Error(err))
		return
	}

	logger.Error("Unexpected error occurred", zap.Error(err))
}

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// check if any error occurred
		errs := c.Errors
		if len(errs) == 0 {
			return
		}

		// take the last error
		err := errs.Last().Err

		var appErr *AppError
		if ok := errors.As(err, &appErr); ok {
			// check for internal error
			if appErr.Err != nil {
				logInternalError(logger, appErr.Err)
			}

			msgs := appErr.Messages
			body := gin.H{"errors": msgs}
			if len(msgs) == 1 {
				body = gin.H{"error": msgs[0]}
			}

			c.JSON(appErr.Code, body)
		} else {
			logger.Error("Unhandled non-app error occurred", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unhandled server error! Please report to service administrator."})
		}
	}
}
