package repository

import (
	"context"
	"ricehub/src/models"

	"github.com/jackc/pgx/v5"
)

func FetchTags() (tags []models.Tag, err error) {
	query := "SELECT * FROM tags ORDER BY id"
	rows, _ := db.Query(context.Background(), query)
	tags, err = pgx.CollectRows(rows, pgx.RowToStructByName[models.Tag])
	return
}

func InsertTag(name string) (tag models.Tag, err error) {
	query := "INSERT INTO tags (name) VALUES ($1) RETURNING *"
	rows, _ := db.Query(context.Background(), query, name)
	tag, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[models.Tag])
	return
}

func UpdateTag(id int, name string) (tag models.Tag, err error) {
	query := "UPDATE tags SET name = $1 WHERE id = $2 RETURNING *"
	rows, _ := db.Query(context.Background(), query, name, id)
	tag, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Tag])
	return
}

func DeleteTag(id int) (bool, error) {
	query := "DELETE FROM tags WHERE id = $1"
	cmd, err := db.Exec(context.Background(), query, id)
	return cmd.RowsAffected() == 1, err
}
