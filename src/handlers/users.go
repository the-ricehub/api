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
	"strings"

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

func GetUserIdFromRequest(c *gin.Context) *string {
	var userId *string = nil
	tokenStr := strings.TrimSpace(c.GetHeader("Authorization"))
	token, err := utils.ValidateToken(tokenStr)
	if err == nil {
		userId = &token.Subject
	}
	return userId
}

func FetchUsers(c *gin.Context) {
	token := c.MustGet("token").(*utils.AccessToken)

	username := c.Query("username")
	if username == "" && !token.IsAdmin {
		c.Error(errs.UserError("Username query parameter is required", http.StatusBadRequest))
		return
	}

	if username != "" {
		user, err := repository.FindUserByUsername(username)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.Error(errs.UserError("User with provided username does not exist", http.StatusNotFound))
				return
			}

			c.Error(errs.InternalError(err))
			return
		}

		c.JSON(http.StatusOK, user.ToDTO())
	} else {
		limit, err := utils.ParseLimitQuery(c)
		if err != nil {
			c.Error(err)
			return
		}

		users, err := repository.FetchRecentUsers(limit)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		c.JSON(http.StatusOK, models.UsersToDTOs(users))
	}
}

func GetUserById(c *gin.Context) {
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

	// check if request has been sent by logged in user
	userId := GetUserIdFromRequest(c)

	rice, err := repository.FindRiceBySlug(userId, slug, username)
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

func FetchUserRices(c *gin.Context) {
	userId := c.Param("id")

	rices, err := repository.FetchUserRices(userId)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.PartialRicesToDTOs(rices))
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

	// check if display name is blacklisted
	blacklist := append(utils.Config.Blacklist.Words, utils.Config.Blacklist.Usernames[:]...)
	for _, word := range blacklist {
		if strings.Contains(strings.ToLower(body.DisplayName), word) {
			c.Error(errs.BlacklistedDisplayName)
			return
		}
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

	c.JSON(http.StatusCreated, gin.H{"avatarUrl": utils.Config.CDNUrl + avatarPath})
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

	c.JSON(http.StatusOK, gin.H{"avatarUrl": utils.Config.CDNUrl + utils.Config.DefaultAvatar})
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
