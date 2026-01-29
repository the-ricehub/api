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
`
const insertCommentSql = `
INSERT INTO rice_comments (rice_id, author_id, content)
VALUES ($1, $2, $3)
RETURNING *
`
const updateCommentSql = `
UPDATE rice_comments SET content = $1 WHERE id = $2
RETURNING *
`
const deleteCommentSql = `
DELETE FROM rice_comments
WHERE id = $1
`

func HasUserCommentWithId(commentId string, userId string) (bool, error) {
	var exists bool
	err := db.QueryRow(context.Background(), hasUserCommentSql, commentId, userId).Scan(&exists)
	return exists, err
}

func FetchCommentsByRiceId(riceId string) (c []models.CommentWithUser, err error) {
	c, err = rowsToStruct[models.CommentWithUser](riceCommentsSql, riceId)
	return
}

func InsertComment(riceId string, authorId string, content string) (c models.RiceComment, err error) {
	c, err = rowToStruct[models.RiceComment](insertCommentSql, riceId, authorId, content)
	return
}

func UpdateComment(commentId string, content string) (c models.RiceComment, err error) {
	c, err = rowToStruct[models.RiceComment](updateCommentSql, content, commentId)
	return
}

func DeleteComment(commentId string) error {
	_, err := db.Exec(context.Background(), deleteCommentSql, commentId)
	return err
}
