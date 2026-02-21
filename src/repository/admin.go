package repository

import "ricehub/src/models"

func FetchServiceStatistics() (stats models.ServiceStatistics, err error) {
	const sql = `
	SELECT
		(
        SELECT COUNT(*) FROM users
        ) AS user_count,
        (
        SELECT COUNT(*) FROM rices
        ) AS rice_count,
        (
        SELECT COUNT(*) FROM rice_comments
        ) AS comment_count,
        (
        SELECT COUNT(*) FROM reports
        ) AS report_count
	`

	stats, err = rowToStruct[models.ServiceStatistics](sql)
	return
}
