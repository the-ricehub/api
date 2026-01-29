package repository

import (
	"context"
	"ricehub/src/models"
)

const insertReportSql = `
INSERT INTO reports (reporter_id, reason, rice_id, comment_id)
VALUES ($1, $2, $3, $4)
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

func InsertReport(reporterId string, reason string, riceId *string, commentId *string) error {
	_, err := db.Exec(context.Background(), insertReportSql, reporterId, reason, riceId, commentId)
	return err
}

func FetchReports() (r []models.ReportWithUser, err error) {
	r, err = rowsToStruct[models.ReportWithUser](fetchReportsSql)
	return
}

func FindReport(reportId string) (r models.ReportWithUser, err error) {
	r, err = rowToStruct[models.ReportWithUser](findReportSql, reportId)
	return
}

func SetReportIsClosed(reportId string, newState bool) (bool, error) {
	cmd, err := db.Exec(context.Background(), setIsClosedSql, newState, reportId)
	return cmd.RowsAffected() == 1, err
}
