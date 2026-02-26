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

var reportNotFound = errs.UserError("Report with provided ID not found!", http.StatusNotFound)

func FetchReports(c *gin.Context) {
	reports, err := repository.FetchReports()
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, models.ReportsToDTO(reports))
}

func GetReportById(c *gin.Context) {
	reportID := c.Param("reportId")
	report, err := repository.FindReport(reportID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Error(reportNotFound)
			return
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusOK, report.ToDTO())
}

func CreateReport(c *gin.Context) {
	token := c.MustGet("token").(*utils.AccessToken)

	var report models.CreateReportDTO
	if err := utils.ValidateJSON(c, &report); err != nil {
		c.Error(err)
		return
	}

	if report.RiceID != nil && report.CommentID != nil {
		c.Error(errs.UserError("Too many resources provided! You can only report one thing at a time.", http.StatusBadRequest))
		return
	}

	reportId, err := repository.InsertReport(token.Subject, report.Reason, report.RiceID, report.CommentID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerrcode.ForeignKeyViolation:
				c.Error(errs.UserError("Resource with provided ID not found!", http.StatusNotFound))
				return
			case pgerrcode.UniqueViolation:
				c.Error(errs.UserError("You have already submitted a similar report!", http.StatusConflict))
				return
			}
		}

		c.Error(errs.InternalError(err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"reportId": reportId})
}

func CloseReport(c *gin.Context) {
	reportID := c.Param("reportId")

	updated, err := repository.SetReportIsClosed(reportID, true)
	if err != nil {
		c.Error(errs.InternalError(err))
		return
	}
	if !updated {
		c.Error(reportNotFound)
		return
	}

	c.Status(http.StatusNoContent)
}
