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

var blacklistedTitle = errs.UserError("Title contains blacklisted words!", http.StatusUnprocessableEntity)
var blacklistedDescription = errs.UserError("Description contains blacklisted words!", http.StatusUnprocessableEntity)

func checkCanUserModifyRice(token *utils.AccessToken, riceId string) error {
	if token.IsAdmin {
		return nil
	}

	isAuthor, err := repository.HasUserRiceWithId(riceId, token.Subject)
	if err != nil || !isAuthor {
		return errs.NoAccess
	}

	return nil
}

var availableSorts = []string{"trending", "recent", "mostDownloads", "mostStars"}

func FetchRices(c *gin.Context) {
	sort := c.DefaultQuery("sort", "trending")
	if !slices.Contains(availableSorts, sort) {
		c.Error(errs.UserError("Unsupported sorting method requested!", http.StatusBadRequest))
		return
	}

	lastId := c.Query("lastId")
	lastCreatedAt := c.Query("lastCreatedAt")
	lastDownloads := c.Query("lastDownloads")

	var pag repository.Pagination
	useDefault := lastId == "" && lastCreatedAt == "" && lastDownloads == ""

	if lastId != "" {
		id, err := uuid.Parse(lastId)
		if err != nil {
			c.Error(errs.UserError("Failed to parse last id", http.StatusBadRequest))
			return
		}

		pag.LastId = &id
	}

	if lastCreatedAt != "" {
		ts, err := time.Parse("2006-01-02T15:04:05.000000-07:00", lastCreatedAt)
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

	if useDefault {
		pag.LastCreatedAt = time.Now().AddDate(999, 1, 1)
		pag.LastId = nil
		pag.LastDownloads = -1
	}

	rices := []models.PartialRice{}
	var err error

	userId := GetUserIdFromRequest(c)

	switch sort {
	case "trending":
		rices, err = repository.FetchTrendingRices(&pag, userId)
	case "recent":
		rices, err = repository.FetchRecentRices(&pag, userId)
	case "mostDownloads":
		rices, err = repository.FetchMostDownloadedRices(&pag, userId)
	case "mostStars":
		rices, err = repository.FetchMostStarredRices(&pag, userId)
	}

	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.PartialRicesToDTOs(rices))
}

func GetRiceById(c *gin.Context) {
	riceId := c.Param("id")

	userId := GetUserIdFromRequest(c)
	rice, err := repository.FindRiceById(userId, riceId)
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

func GetRiceComments(c *gin.Context) {
	riceId := c.Param("id")

	comments, err := repository.FetchCommentsByRiceId(riceId)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.CommentsWithUserToDTOs(comments))
}

func DownloadDotfiles(c *gin.Context) {
	riceId := c.Param("id")

	filePath, err := repository.IncrementDotfilesDownloads(riceId)
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
	token := c.MustGet("token").(*utils.AccessToken)

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
	for _, word := range utils.Config.Blacklist.Words {
		if strings.Contains(strings.ToLower(metadata.Title), word) {
			c.Error(blacklistedTitle)
			return
		}

		if strings.Contains(strings.ToLower(metadata.Description), word) {
			c.Error(blacklistedDescription)
			return
		}
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
	rice, err := repository.InsertRice(tx, token.Subject, metadata.Title, slug.Make(metadata.Title), metadata.Description)
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

		if err := repository.InsertRicePreviewTx(tx, rice.Id, path); err != nil {
			c.Error(errs.InternalError(err))
			return
		}

		dto.Previews = append(dto.Previews, utils.Config.CDNUrl+path)
	}

	// save dotfiles on the disk
	dotfilesPath := fmt.Sprintf("/dotfiles/%v%v", uuid.New(), dotfilesExt)
	c.SaveUploadedFile(dotfilesFile, "./public"+dotfilesPath)

	dotfilesSize := dotfilesFile.Size
	dotfiles, err := repository.InsertRiceDotfiles(tx, rice.Id, dotfilesPath, dotfilesSize)
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
	riceId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	if err := checkCanUserModifyRice(token, riceId); err != nil {
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
	for _, word := range utils.Config.Blacklist.Words {
		if metadata.Title != nil && strings.Contains(strings.ToLower(*metadata.Title), word) {
			c.Error(blacklistedTitle)
			return
		}

		if metadata.Description != nil && strings.Contains(strings.ToLower(*metadata.Description), word) {
			c.Error(blacklistedDescription)
			return
		}
	}

	rice, err := repository.UpdateRice(riceId, metadata.Title, metadata.Description)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, rice)
}

func UpdateDotfiles(c *gin.Context) {
	riceId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	if err := checkCanUserModifyRice(token, riceId); err != nil {
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
	oldDotfiles, err := repository.FetchRiceDotfilesPath(riceId)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if oldDotfiles != nil {
		path := "./public" + *oldDotfiles
		if err := os.Remove(path); err != nil {
			zap.L().Warn("Failed to remove old dotfiles from CDN", zap.String("path", path))
		}
	}

	path := fmt.Sprintf("/dotfiles/%v%v", uuid.New(), ext)
	c.SaveUploadedFile(file, "./public"+path)

	fileSize := file.Size
	df, err := repository.UpdateRiceDotfiles(riceId, path, fileSize)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, df.ToDTO())
}

func AddPreview(c *gin.Context) {
	riceId := c.Param("id")

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

	count, err := repository.RicePreviewCount(riceId)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if count >= int64(utils.Config.Limits.MaxPreviewsPerRice) {
		c.Error(errs.UserError("You have already reached the maximum amount of previews for this rice", http.StatusRequestEntityTooLarge))
		return
	}

	path := fmt.Sprintf("/previews/%v%v", uuid.New(), ext)
	c.SaveUploadedFile(file, "./public"+path)

	_, err = repository.InsertRicePreview(riceId, path)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"preview": utils.Config.CDNUrl + path})
}

func DeletePreview(c *gin.Context) {
	riceId := c.Param("id")
	previewId := c.Param("previewId")
	token := c.MustGet("token").(*utils.AccessToken)

	if err := checkCanUserModifyRice(token, riceId); err != nil {
		c.Error(err)
		return
	}

	// check if there's at least one preview before deleting
	count, err := repository.FetchRicePreviewCount(riceId)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if count <= 1 {
		c.Error(errs.UserError("You cannot delete this preview! At least one preview is required for a rice.", http.StatusUnprocessableEntity))
		return
	}

	deleted, err := repository.DeleteRicePreview(riceId, previewId)
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
	riceId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	if err := repository.InsertRiceStar(riceId, token.Subject); err != nil {
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
	riceId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	if err := repository.DeleteRiceStar(riceId, token.Subject); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}

func DeleteRice(c *gin.Context) {
	riceId := c.Param("id")
	token := c.MustGet("token").(*utils.AccessToken)

	if err := checkCanUserModifyRice(token, riceId); err != nil {
		c.Error(err)
		return
	}

	deleted, err := repository.DeleteRice(riceId)
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
