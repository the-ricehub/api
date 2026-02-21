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

func InsertUser(username string, displayName string, password string) error {
	query := "INSERT INTO users (username, display_name, password) VALUES ($1, $2, $3)"

	_, err := db.Exec(context.Background(), query, username, displayName, password)
	if err != nil {
		return err
	}

	return nil
}

func FetchRecentUsers(limit int64) (users []models.User, err error) {
	const sql = `
	SELECT *
	FROM users
	ORDER BY created_at DESC
	LIMIT $1
	`

	users, err = rowsToStruct[models.User](sql, limit)
	return
}

func FindUserByUsername(username string) (*models.User, error) {
	query := "SELECT * FROM users WHERE username = $1 LIMIT 1"
	rows, _ := db.Query(context.Background(), query, username)
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	return &user, err
}

func FindUserById(userId string) (*models.User, error) {
	query := "SELECT * FROM users WHERE id = $1 LIMIT 1"
	rows, _ := db.Query(context.Background(), query, userId)
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	return &user, err
}

func FetchUserAvatarPath(userId string) (*string, error) {
	var avatarPath *string
	query := "SELECT avatar_path FROM users WHERE id = $1"
	err := db.QueryRow(context.Background(), query, userId).Scan(&avatarPath)
	return avatarPath, err
}

// should I just use single `UpdateUser` function with struct of fields to update and utilize COALESCE?
func UpdateUserDisplayName(userId string, displayName string) error {
	query := "UPDATE users SET display_name = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, displayName, userId)
	return err
}

func UpdateUserPassword(userId string, password string) error {
	query := "UPDATE users SET password = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, password, userId)
	return err
}

func UpdateUserAvatarPath(userId string, avatarPath *string) error {
	query := "UPDATE users SET avatar_path = $1 WHERE id = $2"
	_, err := db.Exec(context.Background(), query, avatarPath, userId)
	return err
}

func DeleteUser(userId string) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := db.Exec(context.Background(), query, userId)
	return err
}
