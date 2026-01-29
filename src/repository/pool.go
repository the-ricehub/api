package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var db *pgxpool.Pool

func Init(connUrl string) {
	logger := zap.L()

	ctx := context.Background()

	var err error
	db, err = pgxpool.New(ctx, connUrl)

	if err != nil {
		logger.Fatal("Failed to establish database connection", zap.Error(err))
	}

	// run a test query to make sure db is working
	var one uint
	err = db.QueryRow(ctx, "SELECT 1").Scan(&one)
	if err != nil {
		logger.Fatal("Failed to perform a connection test", zap.Error(err))
	}
	if one != 1 {
		logger.Fatal("Invalid connection test result",
			zap.Uint("expected", 1),
			zap.Uint("got", one),
		)
	}

	logger.Info("Connection with database successfully established")
}

func Close() {
	db.Close()
}

func StartTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	return tx, err
}

func rowToStruct[T any](sql string, args ...any) (res T, err error) {
	rows, _ := db.Query(context.Background(), sql, args...)
	res, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	return
}

func rowsToStruct[T any](sql string, args ...any) (res []T, err error) {
	rows, _ := db.Query(context.Background(), sql, args...)
	res, err = pgx.CollectRows(rows, pgx.RowToStructByName[T])
	return
}

func txRowToStruct[T any](tx pgx.Tx, sql string, args ...any) (res T, err error) {
	rows, _ := tx.Query(context.Background(), sql, args...)
	res, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	return
}
