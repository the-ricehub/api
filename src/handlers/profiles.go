package handlers

import (
	"errors"
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type profilesPath struct {
	Username string `uri:"username" binding:"required,alphanum"`
}

func GetUserProfile(c *gin.Context) {
	var path profilesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(errs.UserError("Invalid username path parameter. It must be an alphanumeric string.", http.StatusBadRequest))
		return
	}

	// I could create a new repo function that executes these two statements
	// in one query using `WITH ___ AS () [...]` but im tooo lazyyyyyyyyyy

	// fetch user data
	user, err := repository.FindUserByUsername(path.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(errs.UserNotFound)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	// fetch user rices
	rices, err := repository.FetchUserRices(user.ID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  user.ToDTO(),
		"rices": models.PartialRicesToDTOs(rices),
	})
}
