package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"
	"ricehub/src/utils"

	"github.com/alexedwards/argon2id"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func isUserPermitted(token *utils.AccessToken, userId string) bool {
	return token.Subject == userId || token.IsAdmin
}

func findUser(userId string) (*models.User, error) {
	user, err := repository.FindUserById(userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// theoretically this response is invalid when admin tries to fetch non-existent user
			return nil, errs.UserError("The requested user has been removed", http.StatusGone)
		}

		return nil, errs.InternalError(err)
	}

	return user, nil
}

func preCheck(token *utils.AccessToken, userId string) (user *models.User, err error) {
	if !isUserPermitted(token, userId) {
		return nil, errs.UserError("You can't access this resource", http.StatusForbidden)
	}

	user, err = findUser(userId)
	return
}

// func GetMe(c *gin.Context) {
// 	token := c.MustGet("token").(*utils.AccessToken)

// 	user, err := findUser(token.Subject)
// 	if err != nil {
// 		c.Error(err)
// 		return
// 	}

// 	c.JSON(http.StatusOK, user.ToDTO())
// }

func GetUser(c *gin.Context) {
	userId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	user, err := preCheck(token, userId)
	if err != nil {
		c.Error(err)
		return
	}

	dto := user.ToDTO()
	c.JSON(http.StatusOK, dto)
}

func GetUserRiceBySlug(c *gin.Context) {
	username := c.Param("id") // path param has to be called 'id' because gin is upset otherwise
	slug := c.Param("slug")

	rice, err := repository.FindRiceBySlug(slug, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(errs.RiceNotFound)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, rice.ToDTO())
}

func UpdateDisplayName(c *gin.Context) {
	userId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	_, err := preCheck(token, userId)
	if err != nil {
		c.Error(err)
		return
	}

	var body models.UpdateDisplayNameDTO
	if err := utils.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	err = repository.UpdateUserDisplayName(userId, body.DisplayName)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func UpdatePassword(c *gin.Context) {
	userId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	user, err := preCheck(token, userId)
	if err != nil {
		c.Error(err)
		return
	}

	var body models.UpdatePasswordDTO
	if err := utils.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	if !token.IsAdmin {
		match, err := argon2id.ComparePasswordAndHash(body.OldPassword, user.Password)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		if !match {
			c.Error(errs.UserError("Invalid current password provided", http.StatusForbidden))
			return
		}
	}

	hash, err := argon2id.CreateHash(body.NewPassword, argon2id.DefaultParams)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	if err := repository.UpdateUserPassword(userId, hash); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func UploadAvatar(c *gin.Context) {
	userId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	_, err := preCheck(token, userId)
	if err != nil {
		c.Error(err)
		return
	}

	// check if user uploaded any file
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(errs.MissingFile)
		return
	}

	ext, err := utils.ValidateFileAsImage(file)
	if err != nil {
		c.Error(err)
		return
	}

	// delete old avatar file (if exists)
	oldAvatar, err := repository.FetchUserAvatarPath(userId)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if oldAvatar != nil {
		path := "./public" + *oldAvatar
		if err := os.Remove(path); err != nil {
			zap.L().Warn("Failed to remove old user avatar from CDN", zap.String("path", path))
		}
	}

	// save file to cdn
	avatarPath := fmt.Sprintf("/avatars/%v%v", uuid.New(), ext)
	c.SaveUploadedFile(file, "./public"+avatarPath)

	// update avatar path in database
	repository.UpdateUserAvatarPath(userId, &avatarPath)

	c.Status(http.StatusCreated)
}

func DeleteAvatar(c *gin.Context) {
	userId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	_, err := preCheck(token, userId)
	if err != nil {
		c.Error(err)
		return
	}

	repository.UpdateUserAvatarPath(userId, nil)

	c.Status(http.StatusNoContent)
}

func DeleteUser(c *gin.Context) {
	userId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	user, err := preCheck(token, userId)
	if err != nil {
		c.Error(err)
		return
	}

	var body models.DeleteUserDTO
	if err := utils.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, user.Password)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !match {
		c.Error(errs.UserError("Invalid current password provided", http.StatusForbidden))
		return
	}

	err = repository.DeleteUser(userId)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}
