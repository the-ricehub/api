package handlers

import (
	"errors"
	"net/http"
	"ricehub/src/errs"
	"ricehub/src/models"
	"ricehub/src/repository"
	"ricehub/src/utils"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func checkCanUserModifyComment(token *utils.AccessToken, commentID string) error {
	if token.IsAdmin {
		return nil
	}

	isAuthor, err := repository.HasUserCommentWithId(commentID, token.Subject)
	if err != nil || !isAuthor {
		return errs.NoAccess
	}

	return nil
}

func AddComment(c *gin.Context) {
	token := c.MustGet("token").(*utils.AccessToken)

	var body models.AddCommentDTO
	if err := utils.ValidateJSON(c, &body); err != nil {
		c.Error(err)
		return
	}

	comment, err := repository.InsertComment(body.RiceID, token.Subject, body.Content)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			c.Error(errs.RiceNotFound)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, comment.ToDTO())
}

func GetRecentComments(c *gin.Context) {
	limit, err := utils.ParseLimitQuery(c)
	if err != nil {
		c.Error(err)
		return
	}

	comments, err := repository.FetchRecentComments(limit)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.CommentsWithUserToDTOs(comments))
}

func GetCommentById(c *gin.Context) {
	commentID := c.Param("id")

	comment, err := repository.FindCommentById(commentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(errs.UserError("Comment with provided ID not found", http.StatusNotFound))
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, comment.ToDTO())
}

func UpdateComment(c *gin.Context) {
	token := c.MustGet("token").(*utils.AccessToken)
	commentID := c.Param("id")

	var update models.UpdateCommentDTO
	if err := utils.ValidateJSON(c, &update); err != nil {
		c.Error(err)
		return
	}

	if err := checkCanUserModifyComment(token, commentID); err != nil {
		c.Error(err)
		return
	}

	comment, err := repository.UpdateComment(commentID, update.Content)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, comment.ToDTO())
}

func DeleteComment(c *gin.Context) {
	token := c.MustGet("token").(*utils.AccessToken)
	commentID := c.Param("id")

	if err := checkCanUserModifyComment(token, commentID); err != nil {
		c.Error(err)
		return
	}

	if err := repository.DeleteComment(commentID); err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.Status(http.StatusNoContent)
}
