package db

import (
	"context"
	"errors"

	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
)

type Database struct {
	conn *pgxpool.Pool
}

type contextKey int

const txKey contextKey = iota

func New(pg *pgxpool.Pool) *Database {
	return &Database{
		pg,
	}
}

func (db *Database) ExecContext(ctx context.Context) pgxtype.Querier {
	if tx := extractTx(ctx); tx != nil {
		return tx
	}
	return db.conn
}

func (db *Database) WithTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error {
	// check whether there is already a transaction open
	if maybeTx := extractTx(ctx); maybeTx != nil {
		log.Debug().Msg("Transaction is already open")
		return txFunc(ctx)
	}

	tx, err := db.conn.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to begin transaction")
		return err
	}
	defer func() {
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			if !errors.Is(errRollback, pgx.ErrTxClosed) {
				log.Warn().Err(errRollback).Msg("Failed to rollback on defer")
			}
			return
		}
		log.Info().Msg("Transaction rollback")
	}()

	// run callback inside the transaction
	err = txFunc(injectTx(ctx, tx))
	if err != nil {
		return err
	}

	// if no error, commit
	if errCommit := tx.Commit(ctx); errCommit != nil {
		log.Error().Err(errCommit).Msg("Failed to commit transaction")
		return errCommit
	}
	return nil
}

func injectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

func extractTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey).(pgx.Tx); ok {
		return tx
	}
	return nil
}
