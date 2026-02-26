package repository

import (
	"context"
	"ricehub/src/models"

	"github.com/google/uuid"
)

const insertReportSql = `
INSERT INTO reports (reporter_id, reason, rice_id, comment_id)
VALUES ($1, $2, $3, $4)
RETURNING id
`
const fetchReportsSql = `
SELECT r.*, u.display_name, u.username
FROM reports r
JOIN users u ON u.id = r.reporter_id
ORDER BY r.created_at DESC
`
const findReportSql = `
SELECT r.*, u.display_name, u.username
FROM reports r
JOIN users u ON u.id = r.reporter_id
WHERE r.id = $1
`
const setIsClosedSql = `UPDATE reports SET is_closed = $1 WHERE id = $2`

func InsertReport(reporterID string, reason string, riceID *string, commentID *string) (id uuid.UUID, err error) {
	err = db.QueryRow(context.Background(), insertReportSql, reporterID, reason, riceID, commentID).Scan(&id)
	return
}

func FetchReports() (r []models.ReportWithUser, err error) {
	r, err = rowsToStruct[models.ReportWithUser](fetchReportsSql)
	return
}

func FindReport(reportID string) (r models.ReportWithUser, err error) {
	r, err = rowToStruct[models.ReportWithUser](findReportSql, reportID)
	return
}

func SetReportIsClosed(reportID string, newState bool) (bool, error) {
	cmd, err := db.Exec(context.Background(), setIsClosedSql, newState, reportID)
	return cmd.RowsAffected() == 1, err
}
