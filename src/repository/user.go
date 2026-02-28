package repository

import (
	"context"
	"ricehub/src/models"

	"github.com/jackc/pgx/v5"
)

func IsUsernameTaken(username string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS ( SELECT 1 FROM users WHERE username = $1 )"

	err := db.QueryRow(context.Background(), query, username).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// I have ZERO idea how to name this function
// ik this isnt even correct english or at least doesnt feel like
func DoesUserExistsByUsername(username string) (exists bool, err error) {
	const sql = `
	SELECT EXISTS(
		SELECT 1 FROM users WHERE username = $1
	)
	`

	err = db.QueryRow(context.Background(), sql, username).Scan(&exists)
	return
}

func InsertUser(username string, displayName string, password string) error {
	query := `
	INSERT INTO users (username, display_name, password)
	VALUES ($1, $2, $3)
	`

	_, err := db.Exec(context.Background(), query, username, displayName, password)
	if err != nil {
		return err
	}

	return nil
}

func FetchRecentUsers(limit int) (users []models.User, err error) {
	const sql = `
	SELECT *
	FROM users_with_ban_status
	ORDER BY created_at DESC
	LIMIT $1
	`

	users, err = rowsToStruct[models.User](sql, limit)
	return
}

// Fetches all banned users with ban data and orders it from recent ban
func FetchBannedUsers() ([]models.UserWithBan, error) {
	// const query = `
	// SELECT
	// 	to_jsonb(u) AS "user",
	// 	to_jsonb(b) AS "ban"
	// FROM users u
	// JOIN user_bans b ON b.user_id = u.id
	// WHERE (b.expires_at > NOW() OR b.expires_at IS NULL) AND b.is_revoked = false
	// ORDER BY b.banned_at DESC
	// `
	const query = `
	SELECT DISTINCT ON (u.id)
		to_jsonb(u) AS "user",
		to_jsonb(b) AS "ban"
	FROM users_with_ban_status u
	JOIN user_bans b ON b.user_id = u.id
	WHERE
		u.is_banned = true
	ORDER BY u.id, b.banned_at DESC
	`

	return rowsToStruct[models.UserWithBan](query)
}

func FindUserByUsername(username string) (*models.User, error) {
	query := "SELECT * FROM users_with_ban_status WHERE username = $1 LIMIT 1"
	rows, _ := db.Query(context.Background(), query, username)
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	return &user, err
}

func FindUserById(userID string) (*models.User, error) {
	query := "SELECT * FROM users_with_ban_status WHERE id = $1 LIMIT 1"
	rows, _ := db.Query(context.Background(), query, userID)
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	return &user, err
}

func FetchUserAvatarPath(userID string) (*string, error) {
	var avatarPath *string
	query := "SELECT avatar_path FROM users WHERE id = $1"
	err := db.QueryRow(context.Background(), query, userID).Scan(&avatarPath)
	return avatarPath, err
}

// should I just use single `UpdateUser` function with struct of fields to update and utilize COALESCE?
func UpdateUserDisplayName(userID string, displayName string) error {
	query := "UPDATE users SET display_name = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, displayName, userID)
	return err
}

func UpdateUserPassword(userID string, password string) error {
	query := "UPDATE users SET password = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, password, userID)
	return err
}

func UpdateUserAvatarPath(userID string, avatarPath *string) error {
	query := "UPDATE users SET avatar_path = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, avatarPath, userID)
	return err
}

// I have no idea how to name this function
// It sets `is_admin` column for provided user ID to false
func RemoveAdminFromUser(userID string) error {
	query := "UPDATE users SET is_admin = false WHERE id = $1"
	_, err := db.Exec(context.Background(), query, userID)
	return err
}

func DeleteUser(userID string) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := db.Exec(context.Background(), query, userID)
	return err
}
