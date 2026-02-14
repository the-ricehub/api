package handlers

import (
	"errors"
	"math"
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"
	"ricehub/src/utils"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

var invalidCredentials = errs.UserError("Invalid credentials provided", http.StatusUnauthorized)

func Register(c *gin.Context) {
	var credentials models.RegisterDTO
	if err := utils.ValidateJSON(c, &credentials); err != nil {
		c.Error(err)
		return
	}

	// check if username or display name contain blacklisted words
	blacklist := append(utils.Config.Blacklist.Words, utils.Config.Blacklist.Usernames[:]...)
	for _, word := range blacklist {
		if strings.Contains(strings.ToLower(credentials.Username), word) {
			c.Error(errs.UserError("You can't use this username! Please try again with a different one.", http.StatusUnprocessableEntity))
			return
		}

		if strings.Contains(strings.ToLower(credentials.DisplayName), word) {
			c.Error(errs.BlacklistedDisplayName)
			return
		}
	}

	// check if username is taken
	usernameTaken, err := repository.IsUsernameTaken(credentials.Username)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if usernameTaken {
		c.Error(errs.UserError("Username is already taken", http.StatusConflict))
		return
	}

	// hash password
	pass, err := argon2id.CreateHash(credentials.Password, argon2id.DefaultParams)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// insert new user
	err = repository.InsertUser(credentials.Username, credentials.DisplayName, pass)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusCreated)
}

func Login(c *gin.Context) {
	var credentials models.LoginDTO
	if err := utils.ValidateJSON(c, &credentials); err != nil {
		c.Error(err)
		return
	}

	// find user by username
	user, err := repository.FindUserByUsername(credentials.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(invalidCredentials)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	// check if password matches
	match, err := argon2id.ComparePasswordAndHash(credentials.Password, user.Password)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	if !match {
		c.Error(invalidCredentials)
		return
	}

	// create tokens
	refresh, err := utils.NewRefreshToken(user.Id)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	access, err := utils.NewAccessToken(user.Id, user.IsAdmin)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	maxAge := int(math.Round(utils.Config.JWT.RefreshExpiration.Seconds()))
	// TODO: use config value for the host
	c.SetCookie("refresh_token", refresh, maxAge, "/", "127.0.0.1", false, false)
	c.JSON(http.StatusOK, gin.H{"accessToken": access, "user": user.ToDTO()})
}

func RefreshToken(c *gin.Context) {
	// try to extract refresh token from cookies
	tokenStr, err := c.Cookie("refresh_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			c.Error(errs.UserError("Refresh token is required to generate a new access token!", http.StatusBadRequest))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	// validate refresh claims
	refresh, err := utils.DecodeRefreshToken(tokenStr)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			c.Error(errs.UserError("Refresh token is expired! Please authenticate again.", http.StatusForbidden))
			return
		}

		c.Error(errs.UserError(err.Error(), http.StatusForbidden))
		return
	}

	// check user data from database
	user, err := repository.FindUserById(refresh.Subject)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// generate access token
	access, err := utils.NewAccessToken(user.Id, user.IsAdmin)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	// return new token in response body
	c.JSON(http.StatusOK, gin.H{"accessToken": access})
}

func LogOut(c *gin.Context) {
	c.SetCookie("refresh_token", "", -10, "/", "127.0.0.1", false, false)
	c.Status(http.StatusOK)
}
