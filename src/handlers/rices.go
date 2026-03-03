package handlers

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"
	"ricehub/src/security"
	"ricehub/src/utils"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type ricesPath struct {
	RiceID string `uri:"id" binding:"required,uuid"`
}

var availableSorts = []string{"trending", "recent", "mostDownloads", "mostStars"}

var invalidRiceID = errs.UserError("Invalid rice ID path parameter. It must be a valid UUID.", http.StatusBadRequest)
var blacklistedTitle = errs.UserError("Title contains blacklisted words!", http.StatusUnprocessableEntity)
var blacklistedDescription = errs.UserError("Description contains blacklisted words!", http.StatusUnprocessableEntity)

func checkCanUserModifyRice(token *security.AccessToken, riceID string) error {
	if token.IsAdmin {
		return nil
	}

	isAuthor, err := repository.HasUserRiceWithId(riceID, token.Subject)
	if err != nil || !isAuthor {
		return errs.NoAccess
	}

	return nil
}

func fetchWaitingRices(c *gin.Context) {
	rices, err := repository.FetchWaitingRices()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.PartialRicesToDTO(rices))
}

func FetchRices(c *gin.Context) {
	token := GetTokenFromRequest(c)
	isAdmin := token != nil && token.IsAdmin

	// TODO: use ShouldBindQuery instead of manually validating each param

	state := c.Query("state")
	// check if user is an admin and can filter by state
	if state != "" && isAdmin {
		fetchWaitingRices(c)
		return
	}

	sort := c.DefaultQuery("sort", "trending")
	if !slices.Contains(availableSorts, sort) {
		c.Error(errs.UserError("Unsupported sorting method requested!", http.StatusBadRequest))
		return
	}

	lastID := c.Query("lastId")
	lastCreatedAt := c.Query("lastCreatedAt")
	lastDownloads := c.Query("lastDownloads")
	lastStars := c.Query("lastStars")
	lastScore := c.Query("lastScore")
	reverseStr := c.DefaultQuery("reverse", "false")

	var pag repository.Pagination
	useDefault := lastID == "" && lastCreatedAt == "" && lastDownloads == "" && lastStars == "" && lastScore == ""

	if lastID != "" {
		id, err := uuid.Parse(lastID)
		if err != nil {
			c.Error(errs.UserError("Failed to parse last id", http.StatusBadRequest))
			return
		}

		pag.LastID = &id
	}

	if lastCreatedAt != "" {
		ts, err := time.Parse(time.RFC3339, lastCreatedAt)
		if err != nil {
			c.Error(errs.UserError("Failed to parse last created timestamp", http.StatusBadRequest))
			return
		}

		pag.LastCreatedAt = ts
	}

	if lastDownloads != "" {
		downloads, err := strconv.Atoi(lastDownloads)
		if err != nil {
			c.Error(errs.UserError("Failed to parse last downloads query parameter", http.StatusBadRequest))
			return
		}

		pag.LastDownloads = downloads
	}

	if lastStars != "" {
		v, err := strconv.Atoi(lastStars)
		if err != nil {
			c.Error(errs.UserError("Failed to parse last stars query parameter", http.StatusBadRequest))
			return
		}

		pag.LastStars = v
	}

	if lastScore != "" {
		v, err := strconv.ParseFloat(lastScore, 32)
		if err != nil {
			c.Error(errs.UserError("Failed to parse last score query parameter", http.StatusBadRequest))
			return
		}

		pag.LastScore = float32(v)
	}

	if useDefault {
		pag.LastID = nil
		pag.LastDownloads = -1
		pag.LastStars = -1
		pag.LastScore = -1
	}

	pag.Reverse = false
	if reverseStr != "false" {
		pag.Reverse = true
	}

	rices := []models.PartialRice{}
	var err error

	var userID *string = nil
	if token != nil {
		userID = &token.Subject
	}

	switch sort {
	case "trending":
		rices, err = repository.FetchTrendingRices(&pag, userID)
	case "recent":
		rices, err = repository.FetchRecentRices(&pag, userID)
	case "mostDownloads":
		rices, err = repository.FetchMostDownloadedRices(&pag, userID)
	case "mostStars":
		rices, err = repository.FetchMostStarredRices(&pag, userID)
	}

	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	pages, err := repository.FetchPageCount()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	if pag.Reverse {
		// reverse rice array
		for i, j := 0, len(rices)-1; i < j; i, j = i+1, j-1 {
			rices[i], rices[j] = rices[j], rices[i]
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"pageCount": pages,
		"rices":     models.PartialRicesToDTO(rices),
	})
}

func GetRiceById(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	token := GetTokenFromRequest(c)
	var userID *string = nil
	if token != nil {
		userID = &token.Subject
	}

	rice, err := repository.FindRiceById(userID, path.RiceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(errs.RiceNotFound)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	if rice.Rice.State == models.Waiting && (token == nil || !token.IsAdmin) {
		c.Error(errs.RiceNotFound)
		return
	}

	c.JSON(http.StatusOK, rice.ToDTO())
}

func GetRiceComments(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	comments, err := repository.FetchCommentsByRiceId(path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.CommentsWithUserToDTO(comments))
}

func DownloadDotfiles(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	filePath, err := repository.IncrementDotfilesDownloads(path.RiceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(errs.RiceNotFound)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.Redirect(http.StatusFound, utils.Config.CDNUrl+filePath)
}

func CreateRice(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	// validate everything first
	form, err := c.MultipartForm()
	if err != nil {
		c.Error(errs.UserError("Invalid multipart form", http.StatusBadRequest))
		return
	}

	var metadata models.CreateRiceDTO
	if err := utils.ValidateForm(c, &metadata); err != nil {
		c.Error(err)
		return
	}

	previews := form.File["previews[]"]
	formDotfiles := form.File["dotfiles"]

	if len(previews) == 0 {
		c.Error(errs.UserError("At least one preview image is required", http.StatusBadRequest))
		return
	}

	maxPreviews := utils.Config.Limits.MaxPreviewsPerRice
	if len(previews) > maxPreviews {
		c.Error(errs.UserError(fmt.Sprintf("You cannot add more than %v previews", maxPreviews), http.StatusRequestEntityTooLarge))
		return
	}

	if len(formDotfiles) == 0 {
		c.Error(errs.UserError("Dotfiles are required", http.StatusBadRequest))
		return
	}
	dotfilesFile := formDotfiles[0]

	validPreviews := make(map[string]*multipart.FileHeader, len(previews))
	for _, preview := range previews {
		ext, err := utils.ValidateFileAsImage(preview)
		if err != nil {
			c.Error(err)
			return
		}

		previewPath := fmt.Sprintf("/previews/%v%v", uuid.New(), ext)
		validPreviews[previewPath] = preview
	}

	dotfilesExt, err := utils.ValidateFileAsArchive(dotfilesFile)
	if err != nil {
		c.Error(err)
		return
	}

	// check if title or description contains blacklisted words
	bl := utils.Config.Blacklist.Words
	if utils.ContainsBlacklistedWord(metadata.Title, bl) {
		c.Error(blacklistedTitle)
		return
	}
	if utils.ContainsBlacklistedWord(metadata.Description, bl) {
		c.Error(blacklistedDescription)
		return
	}

	// end validating

	ctx := context.Background()
	tx, err := repository.StartTx(ctx)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	defer tx.Rollback(context.Background())

	// insert the rice base (we need rice id for db relation)
	rice, err := repository.InsertRice(tx, token.Subject, metadata.Title, slug.Make(metadata.Title), metadata.Description, token.IsAdmin)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			c.Error(errs.UserError("Provided rice title is already in use!", http.StatusConflict))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	dto := rice.ToDTO()

	for path, file := range validPreviews {
		c.SaveUploadedFile(file, "./public"+path)

		if err := repository.InsertRicePreviewTx(tx, rice.ID, path); err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		dto.Previews = append(dto.Previews, utils.Config.CDNUrl+path)
	}

	// save dotfiles on the disk
	dotfilesPath := fmt.Sprintf("/dotfiles/%v%v", uuid.New(), dotfilesExt)
	c.SaveUploadedFile(dotfilesFile, "./public"+dotfilesPath)

	dotfilesSize := dotfilesFile.Size
	dotfiles, err := repository.InsertRiceDotfiles(tx, rice.ID, dotfilesPath, dotfilesSize)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	dto.Dotfiles = dotfiles.ToDTO()

	// finish the tx
	if err := tx.Commit(ctx); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, dto)
}

func UpdateRiceMetadata(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	var metadata *models.UpdateRiceDTO
	if err := utils.ValidateJSON(c, &metadata); err != nil {
		c.Error(err)
		return
	}

	if metadata.Title == nil && metadata.Description == nil {
		c.Error(errs.UserError("No field to update provided", http.StatusBadRequest))
		return
	}

	// check against blacklisted words
	bl := utils.Config.Blacklist.Words
	if metadata.Title != nil && utils.ContainsBlacklistedWord(*metadata.Title, bl) {
		c.Error(blacklistedTitle)
		return
	}
	if metadata.Description != nil && utils.ContainsBlacklistedWord(*metadata.Description, bl) {
		c.Error(blacklistedDescription)
		return
	}

	rice, err := repository.UpdateRice(path.RiceID, metadata.Title, metadata.Description)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, rice)
}

func UpdateDotfiles(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.Error(errs.MissingFile)
		return
	}

	ext, err := utils.ValidateFileAsArchive(file)
	if err != nil {
		c.Error(err)
		return
	}

	// delete old dotfiles (if exist)
	oldDotfiles, err := repository.FetchRiceDotfilesPath(path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if oldDotfiles != nil {
		path := "./public" + *oldDotfiles
		if err := os.Remove(path); err != nil {
			zap.L().Warn("Failed to remove old dotfiles from storage", zap.String("path", path))
		}
	}

	filePath := fmt.Sprintf("/dotfiles/%v%v", uuid.New(), ext)
	c.SaveUploadedFile(file, "./public"+filePath)

	fileSize := file.Size
	df, err := repository.UpdateRiceDotfiles(path.RiceID, filePath, fileSize)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, df.ToDTO())
}

func AddPreview(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

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

	count, err := repository.RicePreviewCount(path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if count >= int64(utils.Config.Limits.MaxPreviewsPerRice) {
		c.Error(errs.UserError("You have already reached the maximum amount of previews for this rice", http.StatusRequestEntityTooLarge))
		return
	}

	filePath := fmt.Sprintf("/previews/%v%v", uuid.New(), ext)
	c.SaveUploadedFile(file, "./public"+filePath)

	_, err = repository.InsertRicePreview(path.RiceID, filePath)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"preview": utils.Config.CDNUrl + filePath})
}

func UpdateRiceState(c *gin.Context) {
	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	var update *models.UpdateRiceStateDTO
	if err := utils.ValidateJSON(c, &update); err != nil {
		c.Error(err)
		return
	}

	rice, err := repository.FindRiceById(nil, path.RiceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(errs.RiceNotFound)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}
	if rice.Rice.State == models.Accepted {
		c.Error(errs.UserError("This rice has been already accepted", http.StatusConflict))
		return
	}

	switch update.NewState {
	case "accepted":
		err := repository.UpdateRiceState(path.RiceID, models.Accepted)
		if err != nil {
			c.Error(errs.InternalError(err))
			return
		}
		c.Status(http.StatusOK)
	case "rejected":
		_, err := repository.DeleteRice(path.RiceID)
		if err != nil {
			zap.L().Error("Database error when trying to delete rejected rice", zap.String("riceId", path.RiceID), zap.Error(err))
			c.Error(errs.InternalError(err))
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func DeletePreview(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path struct {
		RiceID    string `uri:"id" binding:"required,uuid"`
		PreviewID string `uri:"previewId" binding:"required,uuid"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "RiceID") {
			msg = invalidRiceID.Error()
		} else if strings.Contains(msg, "PreviewID") {
			msg = "Invalid preview ID path parameter. It must be a valid UUID."
		}

		c.Error(errs.UserError(msg, http.StatusBadRequest))
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	// check if there's at least one preview before deleting
	count, err := repository.FetchRicePreviewCount(path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if count <= 1 {
		c.Error(errs.UserError("You cannot delete this preview! At least one preview is required for a rice.", http.StatusUnprocessableEntity))
		return
	}

	deleted, err := repository.DeleteRicePreview(path.RiceID, path.PreviewID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !deleted {
		c.Error(errs.UserError("Rice preview with provided ID not found", http.StatusNotFound))
		return
	}

	c.Status(http.StatusNoContent)
}

func AddRiceStar(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	if err := repository.InsertRiceStar(path.RiceID, token.Subject); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.UniqueViolation:
				c.Status(http.StatusCreated)
				return
			case pgerrcode.ForeignKeyViolation:
				c.Error(errs.RiceNotFound)
				return
			}
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusCreated)
}

func DeleteRiceStar(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	if err := repository.DeleteRiceStar(path.RiceID, token.Subject); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func DeleteRice(c *gin.Context) {
	token := c.MustGet("token").(*security.AccessToken)
	if err := security.VerifyUserID(token.Subject); err != nil {
		c.Error(err)
		return
	}

	var path ricesPath
	if err := c.ShouldBindUri(&path); err != nil {
		c.Error(invalidRiceID)
		return
	}

	if err := checkCanUserModifyRice(token, path.RiceID); err != nil {
		c.Error(err)
		return
	}

	deleted, err := repository.DeleteRice(path.RiceID)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !deleted {
		c.Error(errs.RiceNotFound)
		return
	}

	c.Status(http.StatusNoContent)
}
