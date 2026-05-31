package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	CodeUniqueViolation = "23505"
)

// Ключ контекста (неэкспортируемый, чтобы никто снаружи его не затер)
type txKey struct{}

// Tx описывает общие методы транзакции
type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// TxManager отвечает за управление транзакциями
type TxManager interface {
	Begin(ctx context.Context) (Tx, context.Context, error)
}

// PostgresTxManager реализует TxManager для pgx
type PostgresTxManager struct {
	pool *pgxpool.Pool
}

func NewPostgresTxManager(pool *pgxpool.Pool) *PostgresTxManager {
	return &PostgresTxManager{pool: pool}
}

func (m *PostgresTxManager) Begin(ctx context.Context) (Tx, context.Context, error) {
	pgxTx, err := m.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin pgx tx: %w", err)
	}

	customTx := &pgxTxWrapper{tx: pgxTx}
	txCtx := context.WithValue(ctx, txKey{}, pgxTx)

	return customTx, txCtx, nil
}

// pgxTxWrapper адаптирует pgx.Txпод наш общий интерфейс
type pgxTxWrapper struct {
	tx pgx.Tx
}

func (w *pgxTxWrapper) Commit(ctx context.Context) error   { return w.tx.Commit(ctx) }
func (w *pgxTxWrapper) Rollback(ctx context.Context) error { return w.tx.Rollback(ctx) }

// GetTxFromContext — публичный хелпер. Вытаскивает активную pgx.Tx из контекста.
// Если транзакции в контексте нет, возвращает nil.
func GetTxFromContext(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}
