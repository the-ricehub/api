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
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

var invalidUserID = errs.UserError("Invalid user ID provided. It must be a valid UUID.", http.StatusBadRequest)
var userNotFound = errs.UserError("User not found", http.StatusNotFound)

func isUserPermitted(token *utils.AccessToken, userID string) bool {
	return token.Subject == userID || token.IsAdmin
}

func findUser(userID string) (*models.User, error) {
	user, err := repository.FindUserById(userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, userNotFound
		}

		return nil, errs.InternalError(err)
	}

	return user, nil
}

func preCheck(token *utils.AccessToken, userID string) (user *models.User, err error) {
	if !isUserPermitted(token, userID) {
		return nil, errs.UserError("You can't access this resource", http.StatusForbidden)
	}

	user, err = findUser(userID)
	return
}

// I chose to not implement custom validator for expiration/duration
// because in the end I need to parse the string again in the handler
// to get time.Duration therefore Imma just write a helper func instead
func computeExpiration(duration *string) (*time.Time, error) {
	if duration != nil {
		parsed, err := time.ParseDuration(*duration)
		if err != nil {
			return nil, errs.UserError("Failed to parse duration", http.StatusBadRequest)
		}

		if parsed.Seconds() < 0 {
			return nil, errs.UserError("Duration must be a non-negative value", http.StatusBadRequest)
		}

		temp := time.Now().Add(parsed)
		return &temp, nil
	}

	return nil, nil
}

func GetUserIdFromRequest(c *gin.Context) *string {
	var userID *string = nil
	tokenStr := strings.TrimSpace(c.GetHeader("Authorization"))
	token, err := utils.ValidateToken(tokenStr)
	if err == nil {
		userID = &token.Subject
	}
	return userID
}

func FetchUsers(c *gin.Context) {
	username := c.Query("username")
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
		return
	}

	// check if user is an admin
	tokenStr := c.Request.Header.Get("Authorization")
	tokenStr = strings.TrimSpace(tokenStr)

	token, err := utils.ValidateToken(tokenStr)
	if err != nil {
		c.Error(err)
		return
	}

	if !token.IsAdmin {
		c.Error(errs.UserError("Username query parameter is required", http.StatusBadRequest))
		return
	}

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

type PathParams struct {
	UserID string `uri:"id" binding:"required,uuid"`
}

func GetUserById(c *gin.Context) {
	var path PathParams
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidUserID)
		return
	}

	token := c.MustGet("token").(*utils.AccessToken)

	user, err := preCheck(token, path.UserID)
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
	userID := GetUserIdFromRequest(c)

	// check if rice's author exists
	exists, err := repository.DoesUserExistsByUsername(username)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !exists {
		c.Error(userNotFound)
		return
	}

	rice, err := repository.FindRiceBySlug(userID, slug, username)
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
	var path PathParams
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidUserID)
		return
	}

	rices, err := repository.FetchUserRices(path.UserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.PartialRicesToDTOs(rices))
}

func UpdateDisplayName(c *gin.Context) {
	var path PathParams
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidUserID)
		return
	}

	token := c.MustGet("token").(*utils.AccessToken)

	_, err := preCheck(token, path.UserID)
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

	err = repository.UpdateUserDisplayName(path.UserID, body.DisplayName)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func UpdatePassword(c *gin.Context) {
	var path PathParams
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidUserID)
		return
	}

	token := c.MustGet("token").(*utils.AccessToken)

	user, err := preCheck(token, path.UserID)
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

	if err := repository.UpdateUserPassword(path.UserID, hash); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func UploadAvatar(c *gin.Context) {
	var path PathParams
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidUserID)
		return
	}

	token := c.MustGet("token").(*utils.AccessToken)

	_, err := preCheck(token, path.UserID)
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
	oldAvatar, err := repository.FetchUserAvatarPath(path.UserID)
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
	repository.UpdateUserAvatarPath(path.UserID, &avatarPath)

	c.JSON(http.StatusCreated, gin.H{"avatarUrl": utils.Config.CDNUrl + avatarPath})
}

func BanUser(c *gin.Context) {
	var path PathParams
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidUserID)
		return
	}

	token := c.MustGet("token").(*utils.AccessToken)

	// 1. validate DTO
	var ban *models.BanUserDTO
	if err := utils.ValidateJSON(c, &ban); err != nil {
		c.Error(err)
		return
	}

	// 2. compute expiration
	expiresAt, err := computeExpiration(ban.Duration)
	if err != nil {
		c.Error(err)
		return
	}

	// 3. check if user exists AND is not already banned
	state, err := repository.IsUserBanned(path.UserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !state.UserExists {
		c.Error(userNotFound)
		return
	}
	if state.UserBanned {
		c.Error(errs.UserError("User is already banned", http.StatusConflict))
		return
	}

	// 4. insert ban into the database
	userBan, err := repository.InsertBan(path.UserID, token.Subject, ban.Reason, expiresAt)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// 5. return 201 with ban id in json
	c.JSON(http.StatusCreated, userBan.ToDTO())
}

func UnbanUser(c *gin.Context) {
	var path PathParams
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidUserID)
		return
	}

	// 1. check if user is banned
	state, err := repository.IsUserBanned(path.UserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !state.UserExists {
		c.Error(userNotFound)
		return
	}
	if !state.UserBanned {
		c.Error(errs.UserError("User is not banned", http.StatusConflict))
		return
	}

	// 2. revoke ban in the database
	if err := repository.RevokeBan(path.UserID); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// 3. return 204
	c.Status(http.StatusNoContent)
}

func DeleteAvatar(c *gin.Context) {
	var path PathParams
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidUserID)
		return
	}

	token := c.MustGet("token").(*utils.AccessToken)

	_, err := preCheck(token, path.UserID)
	if err != nil {
		c.Error(err)
		return
	}

	repository.UpdateUserAvatarPath(path.UserID, nil)

	c.JSON(http.StatusOK, gin.H{"avatarUrl": utils.Config.CDNUrl + utils.Config.DefaultAvatar})
}

func DeleteUser(c *gin.Context) {
	var path PathParams
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidUserID)
		return
	}

	token := c.MustGet("token").(*utils.AccessToken)

	user, err := preCheck(token, path.UserID)
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

	err = repository.DeleteUser(path.UserID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}
