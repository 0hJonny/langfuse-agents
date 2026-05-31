package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/0hJonny/langfuse-agents/internal/auth/domain"
	"github.com/0hJonny/langfuse-agents/pkg/postgres"
)

var _ domain.UserRepository = (*PostgresRepository)(nil)

type PostgresRepository struct {
	queries *Queries
}

func NewPostgresRepository(queries *Queries) *PostgresRepository {
	return &PostgresRepository{queries: queries}
}

func (r *PostgresRepository) getQueries(ctx context.Context) *Queries {
	if tx := postgres.GetTxFromContext(ctx); tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *domain.User) (domain.User, error) {
	q := r.getQueries(ctx)

	// Если роль не задана в домене, ставим дефолтную анонимную
	dbRole := domain.UserRoleAnonymous
	if user.Role != "" {
		dbRole = user.Role // Прямое присвоение без кастов!
	}

	dbUser, err := q.CreateUser(ctx, CreateUserParams{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         dbRole, // Передаем чистый тип domain.UserRole
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == postgres.CodeUniqueViolation {
			return domain.User{}, domain.ErrUserAlreadyExists
		}
		return domain.User{}, err
	}

	return domain.User{
		ID:           dbUser.ID.String(),
		Email:        dbUser.Email,
		PasswordHash: dbUser.PasswordHash,
		Role:         dbUser.Role, // Идеально маппится один к одному
		CreatedAt:    dbUser.CreatedAt.Time,
	}, nil
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	q := r.getQueries(ctx)

	dbUser, err := q.GetUserByEmail(ctx, &email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}

	return domain.User{
		ID:           dbUser.ID.String(),
		Email:        dbUser.Email,
		PasswordHash: dbUser.PasswordHash,
		Role:         dbUser.Role,
		CreatedAt:    dbUser.CreatedAt.Time,
	}, nil
}

func (r *PostgresRepository) UpdateUser(ctx context.Context, user *domain.User) (domain.User, error) {
	q := r.getQueries(ctx)

	// Парсим строковый ID из домена в тип pgtype.UUID, который ожидает sqlc
	var dbID pgtype.UUID
	if err := dbID.Scan(user.ID); err != nil {
		return domain.User{}, fmt.Errorf("failed to parse user uuid: %w", err)
	}

	// Вызываем сгенерированный sqlc метод для апгрейда юзера
	dbUser, err := q.UpdateUserToRegistered(ctx, UpdateUserToRegisteredParams{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		ID:           dbID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		// Если email, который ввел аноним, уже занят другим аккаунтом
		if errors.As(err, &pgErr) && pgErr.Code == postgres.CodeUniqueViolation {
			return domain.User{}, domain.ErrUserAlreadyExists
		}
		return domain.User{}, err
	}

	// Возвращаем обновленного пользователя обратно в сервис
	return domain.User{
		ID:           dbUser.ID.String(),
		Email:        dbUser.Email,
		PasswordHash: dbUser.PasswordHash,
		Role:         dbUser.Role, // Здесь уже будет UserRoleUser
		CreatedAt:    dbUser.CreatedAt.Time,
	}, nil
}
