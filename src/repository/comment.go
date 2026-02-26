package repository

import (
	"context"
	"ricehub/src/models"
)

const hasUserCommentSql = `
SELECT EXISTS (
	SELECT 1
	FROM rice_comments
	WHERE id = $1 AND author_id = $2
)
`
const riceCommentsSql = `
SELECT c.id AS comment_id, c.content, c.created_at, c.updated_at, u.display_name, u.username, u.avatar_path
FROM rice_comments c
JOIN users u ON u.id = c.author_id
WHERE rice_id = $1
ORDER BY created_at DESC
`
const insertCommentSql = `
INSERT INTO rice_comments (rice_id, author_id, content)
VALUES ($1, $2, $3)
RETURNING *
`
const fetchRecentCommentsSql = `
SELECT c.id AS comment_id, c.content, c.created_at, c.updated_at, u.display_name, u.username, u.avatar_path
FROM rice_comments c
JOIN users u ON u.id = c.author_id
ORDER BY c.created_at DESC
LIMIT $1
`

// deluxe version of find comment because it fetches username and slug too
const findCommentByIdSql = `
SELECT rc.*, r.slug AS rice_slug, u.username AS rice_author_username
FROM rice_comments rc
JOIN rices r ON r.id = rc.rice_id
JOIN users u ON u.id = r.author_id
WHERE rc.id = $1
`
const updateCommentSql = `
UPDATE rice_comments SET content = $1 WHERE id = $2
RETURNING *
`
const deleteCommentSql = `
DELETE FROM rice_comments
WHERE id = $1
`

func InsertComment(riceID string, authorID string, content string) (c models.RiceComment, err error) {
	c, err = rowToStruct[models.RiceComment](insertCommentSql, riceID, authorID, content)
	return
}

func HasUserCommentWithId(commentID string, userID string) (bool, error) {
	var exists bool
	err := db.QueryRow(context.Background(), hasUserCommentSql, commentID, userID).Scan(&exists)
	return exists, err
}

func FetchRecentComments(limit int64) (c []models.CommentWithUser, err error) {
	c, err = rowsToStruct[models.CommentWithUser](fetchRecentCommentsSql, limit)
	return
}

func FetchCommentsByRiceId(riceID string) (c []models.CommentWithUser, err error) {
	c, err = rowsToStruct[models.CommentWithUser](riceCommentsSql, riceID)
	return
}

func FindCommentById(commentID string) (c models.RiceCommentWithSlug, err error) {
	c, err = rowToStruct[models.RiceCommentWithSlug](findCommentByIdSql, commentID)
	return
}

func UpdateComment(commentID string, content string) (c models.RiceComment, err error) {
	c, err = rowToStruct[models.RiceComment](updateCommentSql, content, commentID)
	return
}

func DeleteComment(commentID string) error {
	_, err := db.Exec(context.Background(), deleteCommentSql, commentID)
	return err
}
