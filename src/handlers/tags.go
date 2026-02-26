package handlers

import (
	"errors"
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"
	"ricehub/src/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var invalidTagID = errs.UserError("Failed to parse tag ID, it must be an integer!", http.StatusBadRequest)
var tagNotFound = errs.UserError("Tag with provided ID not found", http.StatusNotFound)

func GetAllTags(c *gin.Context) {
	tags, err := repository.FetchTags()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	dtos := make([]models.TagDTO, len(tags))
	for i, tag := range tags {
		dtos[i] = tag.ToDTO()
	}

	c.JSON(http.StatusOK, dtos)
}

func CreateTag(c *gin.Context) {
	var newTag *models.TagNameDTO
	if err := utils.ValidateJSON(c, &newTag); err != nil {
		c.Error(err)
		return
	}

	tag, err := repository.InsertTag(newTag.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			c.Error(errs.UserError("Tag with that name already exists!", http.StatusConflict))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, tag.ToDTO())
}

func UpdateTag(c *gin.Context) {
	tagId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Error(invalidTagID)
		return
	}

	var update *models.TagNameDTO
	if err := utils.ValidateJSON(c, &update); err != nil {
		c.Error(err)
		return
	}

	tag, err := repository.UpdateTag(tagId, update.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(tagNotFound)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, tag.ToDTO())
}

func DeleteTag(c *gin.Context) {
	tagId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Error(invalidTagID)
		return
	}

	deleted, err := repository.DeleteTag(tagId)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !deleted {
		c.Error(tagNotFound)
		return
	}

	c.Status(http.StatusNoContent)
}
