package repository

import (
	"context"
	"fmt"
	"ricehub/src/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// idk if thats how you're supposed to write golang code but whatever
// let a man be happy after going through Rust horror
const hasUserRiceSql = `
SELECT EXISTS (
	SELECT 1
	FROM rices
	WHERE id = $1 AND author_id = $2
)
`

func buildFetchRicesSql(sortBy string, subsequent bool, withUser bool) string {
	argCount := 1

	baseSelect := `
	SELECT
		r.id, r.title, r.slug, r.created_at,
		u.display_name, u.username,
		p.file_path AS thumbnail,
		count(DISTINCT s.user_id) AS star_count,
		df.download_count
	`

	userSelect := ",false AS is_starred"
	if withUser {
		userSelect = fmt.Sprintf(`
		,EXISTS (
			SELECT 1
			FROM rice_stars rs
			WHERE rs.rice_id = r.id AND rs.user_id = $%v
		) AS is_starred
		`, argCount)
		argCount += 1
	}

	base := `
	FROM rices r
	JOIN users u ON u.id = r.author_id
	LEFT JOIN rice_stars s ON s.rice_id = r.id
	JOIN rice_dotfiles df ON df.rice_id = r.id
	JOIN LATERAL (
		SELECT p.file_path
		FROM rice_previews p
		WHERE p.rice_id = r.id
		ORDER BY p.created_at
		LIMIT 1
	) p ON TRUE
	`
	where := ""
	order := ""

	switch sortBy {
	case "trending":
		if subsequent {
			where = fmt.Sprintf("WHERE r.id < $%v", argCount)
			argCount += 1
		}
		order = "ORDER BY (df.download_count + count(DISTINCT s.user_id)) / pow(extract(EPOCH FROM (current_timestamp - r.created_at)) / 3600 + 2, 1.5) DESC, r.id DESC"
	case "recent":
		where = fmt.Sprintf("WHERE (r.created_at, r.id) < ($%v, $%v)", argCount, argCount+1)
		argCount += 2
		order = "ORDER BY r.created_at DESC, r.id DESC"
	case "downloads":
		if subsequent {
			where = fmt.Sprintf("WHERE (df.download_count, r.id) < ($%v, $%v)", argCount, argCount+1)
			argCount += 2
		}
		order = "ORDER BY download_count DESC, r.id DESC"
	case "stars":
		if subsequent {
			where = fmt.Sprintf("WHERE r.id < $%v", argCount)
			argCount += 1
		}
		order = "ORDER BY star_count DESC, r.id DESC"
	}

	return baseSelect + userSelect + base + where + " GROUP BY r.id, r.slug, r.title, r.created_at, df.download_count, u.display_name, u.username, p.file_path " + order + " LIMIT 20"
}

type FindRiceBy uint8

const (
	RiceID FindRiceBy = iota
	SlugAndUsername
)

func buildFindRiceSql(findBy FindRiceBy) string {
	suffix := `
	SELECT
		to_jsonb(base) AS rice,
		to_jsonb(u) AS "user",
		to_jsonb(df) AS dotfiles,
		jsonb_agg(to_jsonb(p) ORDER BY p.id) AS previews,
		count(DISTINCT s.user_id) AS star_count,
		coalesce(bool_or(s.user_id = $1), false) AS is_starred
	FROM base
	JOIN users u ON u.id = base.author_id
	JOIN rice_dotfiles df ON df.rice_id = base.id
	JOIN rice_previews p ON p.rice_id = base.id
	LEFT JOIN rice_stars s ON s.rice_id = base.id
	GROUP BY base.*, df.*, u.*
	`

	switch findBy {
	case SlugAndUsername:
		return `
		WITH base AS (
			SELECT r.*
			FROM rices r
			JOIN users u ON u.id = r.author_id
			WHERE r.slug = $2 AND u.username = $3
		)
		` + suffix
	default: // fallback to find by rice id
		return `
		WITH base AS (
			SELECT r.*
			FROM rices r
			WHERE r.id = $2
		)
		` + suffix
	}
}

var findRiceSql = buildFindRiceSql(RiceID)
var findRiceBySlugSql = buildFindRiceSql(SlugAndUsername)

const fetchUserRicesSql = `
SELECT
	r.id, r.title, r.slug, r.created_at,
	u.display_name, u.username,
	p.file_path AS thumbnail,
	count(DISTINCT s.user_id) AS star_count,
	df.download_count, EXISTS (
		SELECT 1
		FROM rice_stars rs
		WHERE rs.rice_id = r.id AND rs.user_id = $1
	) AS is_starred
FROM rices r
JOIN users u ON u.id = r.author_id
LEFT JOIN rice_stars s ON s.rice_id = r.id
JOIN rice_dotfiles df ON df.rice_id = r.id
JOIN LATERAL (
	SELECT p.file_path
	FROM rice_previews p
	WHERE p.rice_id = r.id
	ORDER BY p.created_at
	LIMIT 1
) p ON TRUE
WHERE u.id = $1
GROUP BY
    r.id, r.slug, r.title, r.created_at, df.download_count,
    u.display_name, u.username, p.file_path
ORDER BY r.created_at DESC, r.id DESC
`

const insertRiceSql = `
INSERT INTO rices (author_id, title, slug, description)
VALUES ($1, $2, $3, $4)
RETURNING *
`
const insertPreviewSql = `
INSERT INTO rice_previews (rice_id, file_path)
VALUES ($1, $2)
RETURNING *
`
const insertDotfilesSql = `
INSERT INTO rice_dotfiles (rice_id, file_path, file_size)
VALUES ($1, $2, $3)
RETURNING *
`
const insertStarSql = `
INSERT INTO rice_stars (rice_id, user_id)
VALUES ($1, $2)
`
const updateDotfilesSql = `
UPDATE rice_dotfiles
SET file_path = $2, file_size = $3
WHERE rice_id = $1
RETURNING *
`
const incrementDownloadsSql = `
UPDATE rice_dotfiles df
SET download_count = download_count + 1
FROM rices r
WHERE r.id = $1 AND r.id = df.rice_id
RETURNING df.file_path
`
const deletePreviewSql = `
DELETE FROM rice_previews
WHERE id = $1 AND rice_id = $2
`
const previewCountSql = `
SELECT COUNT(*)
FROM rice_previews
WHERE rice_id = $1
`

type Pagination struct {
	LastID        *uuid.UUID
	LastCreatedAt time.Time
	LastDownloads int
}

func HasUserRiceWithId(riceID string, userID string) (bool, error) {
	var exists bool
	err := db.QueryRow(context.Background(), hasUserRiceSql, riceID, userID).Scan(&exists)
	return exists, err
}

func RicePreviewCount(riceID string) (int64, error) {
	var count int64
	err := db.QueryRow(context.Background(), previewCountSql, riceID).Scan(&count)
	return count, err
}

func FetchTrendingRices(pag *Pagination, userID *string) (r []models.PartialRice, err error) {
	query := buildFetchRicesSql("trending", pag.LastID != nil, userID != nil)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}

	if pag.LastID != nil {
		args = append(args, pag.LastID)
	}

	r, err = rowsToStruct[models.PartialRice](query, args...)
	return
}

func FetchRecentRices(pag *Pagination, userID *string) (r []models.PartialRice, err error) {
	query := buildFetchRicesSql("recent", false, userID != nil)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	args = append(args, pag.LastCreatedAt, pag.LastID)

	r, err = rowsToStruct[models.PartialRice](query, args...)
	return
}

func FetchMostDownloadedRices(pag *Pagination, userID *string) (r []models.PartialRice, err error) {
	query := buildFetchRicesSql("downloads", pag.LastDownloads != -1, userID != nil)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if pag.LastDownloads != -1 {
		args = append(args, pag.LastDownloads, pag.LastID)
	}

	r, err = rowsToStruct[models.PartialRice](query, args...)
	return
}

func FetchMostStarredRices(pag *Pagination, userID *string) (r []models.PartialRice, err error) {
	query := buildFetchRicesSql("stars", pag.LastID != nil, userID != nil)

	args := []any{}
	if userID != nil {
		args = append(args, userID)
	}
	if pag.LastID != nil {
		args = append(args, pag.LastID)
	}

	r, err = rowsToStruct[models.PartialRice](query, args...)
	return
}

func FetchRicePreviewCount(riceID string) (int, error) {
	var count int
	err := db.QueryRow(
		context.Background(),
		"SELECT count(*) FROM rice_previews WHERE rice_id = $1",
		riceID,
	).Scan(&count)
	return count, err
}

func FetchRiceDotfilesPath(riceID string) (*string, error) {
	var filePath *string
	query := "SELECT file_path FROM rice_dotfiles WHERE rice_id = $1"
	err := db.QueryRow(context.Background(), query, riceID).Scan(&filePath)
	return filePath, err
}

func FindRiceById(userID *string, riceID string) (r models.RiceWithRelations, err error) {
	r, err = rowToStruct[models.RiceWithRelations](findRiceSql, userID, riceID)
	return
}

func FindRiceBySlug(userID *string, slug string, username string) (r models.RiceWithRelations, err error) {
	r, err = rowToStruct[models.RiceWithRelations](findRiceBySlugSql, userID, slug, username)
	return
}

func FetchUserRices(userID string) (r []models.PartialRice, err error) {
	r, err = rowsToStruct[models.PartialRice](fetchUserRicesSql, userID)
	return
}

func InsertRice(tx pgx.Tx, authorID string, title string, slug string, description string) (rice models.Rice, err error) {
	rice, err = txRowToStruct[models.Rice](tx, insertRiceSql, authorID, title, slug, description)
	return
}

func InsertRicePreview(riceID string, previewPath string) (p models.RicePreview, err error) {
	p, err = rowToStruct[models.RicePreview](insertPreviewSql, riceID, previewPath)
	return
}

func InsertRicePreviewTx(tx pgx.Tx, riceID uuid.UUID, previewPath string) error {
	_, err := tx.Exec(context.Background(), insertPreviewSql, riceID, previewPath)
	return err
}

func InsertRiceDotfiles(tx pgx.Tx, riceID uuid.UUID, dotfilesPath string, dotfilesSize int64) (df models.RiceDotfiles, err error) {
	df, err = txRowToStruct[models.RiceDotfiles](tx, insertDotfilesSql, riceID, dotfilesPath, dotfilesSize)
	return
}

func InsertRiceStar(riceID string, userID string) error {
	_, err := db.Exec(context.Background(), insertStarSql, riceID, userID)
	return err
}

func UpdateRice(riceID string, title *string, description *string) (rice models.Rice, err error) {
	query := "UPDATE rices SET"
	args := []any{riceID}

	if title != nil {
		query += " title = $2"
		args = append(args, *title)
	}

	if description != nil {
		if len(args) > 1 {
			query += ","
		}
		query += " description = $3"
		args = append(args, *description)
	}

	query += " WHERE id = $1 RETURNING *"

	rice, err = rowToStruct[models.Rice](query, args...)
	return
}

func UpdateRiceDotfiles(riceID string, filePath string, fileSize int64) (df models.RiceDotfiles, err error) {
	df, err = rowToStruct[models.RiceDotfiles](updateDotfilesSql, riceID, filePath, fileSize)
	return
}

func IncrementDotfilesDownloads(riceID string) (string, error) {
	var filePath string
	err := db.QueryRow(context.Background(), incrementDownloadsSql, riceID).Scan(&filePath)
	return filePath, err
}

func DeleteRicePreview(riceID string, previewID string) (bool, error) {
	cmd, err := db.Exec(context.Background(), deletePreviewSql, previewID, riceID)
	return cmd.RowsAffected() == 1, err
}

// star deletion is the only query where i dont see the need to check if any row was affected
func DeleteRiceStar(riceID string, userID string) error {
	_, err := db.Exec(
		context.Background(),
		"DELETE FROM rice_stars WHERE rice_id = $1 AND user_id = $2",
		riceID, userID,
	)
	return err
}

func DeleteRice(riceID string) (bool, error) {
	cmd, err := db.Exec(
		context.Background(),
		"DELETE FROM rices WHERE id = $1",
		riceID,
	)
	return cmd.RowsAffected() == 1, err
}
