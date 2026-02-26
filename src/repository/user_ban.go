package repository

import (
	"context"
	"ricehub/src/models"
	"time"
)

func IsUserBanned(userID string) (state models.UserState, err error) {
	const query = `
	SELECT
		EXISTS(
			SELECT 1 FROM users WHERE id = $1
		) AS user_exists,
		EXISTS(
			SELECT 1
			FROM user_bans
			WHERE
				user_id = $1 AND
				(expires_at > NOW() OR expires_at IS NULL) AND
				is_revoked = false
		) AS user_banned
	`

	return rowToStruct[models.UserState](query, userID)
}

func InsertBan(userID string, adminID string, reason string, expiresAt *time.Time) (ban models.UserBan, err error) {
	const query = `
	INSERT INTO user_bans (user_id, admin_id, reason, expires_at)
	VALUES ($1, $2, $3, $4)
	RETURNING *
	`

	return rowToStruct[models.UserBan](query, userID, adminID, reason, expiresAt)
}

// revoke is an irreversible action therefore no need for generalized 'set is_revoked' function as it can only be updated to one state
func RevokeBan(userID string) error {
	const query = `
	UPDATE user_bans
	SET is_revoked = true
	WHERE user_id = $1
	`

	_, err := db.Exec(context.Background(), query, userID)
	return err
}
