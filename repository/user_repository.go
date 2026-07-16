package repository

import (
	"context"
	"refer_api/models"

	"github.com/uptrace/bun"
)

// UserRepository defines the interface for database operations related to users and roles.
type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id string) (*models.User, error)
	FindAll() ([]models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
	Delete(id string) error
	UpdatePassword(userID string, hashedPassword string) error
	FindRoleByName(name string) (*models.Role, error)
	FindRolesByNames(names []string) ([]models.Role, error)
	CreateRole(role *models.Role) error
	GetStarBalance(userID string) (int, error)
}

type userRepository struct {
	db *bun.DB
}

// NewUserRepository creates a new Bun-backed UserRepository instance.
func NewUserRepository(db *bun.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.NewSelect().Model(&user).
		Where("?TableAlias.email = ?", email).
		Relation("Roles").
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.NewSelect().Model(&user).
		Where("?TableAlias.id = ?", id).
		Relation("Roles").
		Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindAll() ([]models.User, error) {
	var users []models.User
	err := r.db.NewSelect().Model(&users).
		Relation("Roles").
		Scan(context.Background())
	return users, err
}

func (r *userRepository) Create(user *models.User) error {
	ctx := context.Background()
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		roles := user.Roles
		user.Roles = nil

		if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
			return err
		}

		for _, role := range roles {
			if _, err := tx.ExecContext(ctx,
				"INSERT IGNORE INTO users_roles (usersId, rolesId) VALUES (?, ?)",
				user.ID, role.ID,
			); err != nil {
				return err
			}
		}

		user.Roles = roles
		return nil
	})
}

func (r *userRepository) Update(user *models.User) error {
	ctx := context.Background()
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		roles := user.Roles
		user.Roles = nil

		if _, err := tx.NewUpdate().Model(user).WherePK().Exec(ctx); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, "DELETE FROM users_roles WHERE usersId = ?", user.ID); err != nil {
			return err
		}

		for _, role := range roles {
			if _, err := tx.ExecContext(ctx,
				"INSERT IGNORE INTO users_roles (usersId, rolesId) VALUES (?, ?)",
				user.ID, role.ID,
			); err != nil {
				return err
			}
		}

		user.Roles = roles
		return nil
	})
}

func (r *userRepository) Delete(id string) error {
	ctx := context.Background()
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM users_roles WHERE usersId = ?", id); err != nil {
			return err
		}
		_, err := tx.NewDelete().Model((*models.User)(nil)).Where("id = ?", id).Exec(ctx)
		return err
	})
}

func (r *userRepository) UpdatePassword(userID string, hashedPassword string) error {
	_, err := r.db.NewUpdate().Model((*models.User)(nil)).
		Set("password = ?", hashedPassword).
		Where("id = ?", userID).
		Exec(context.Background())
	return err
}

func (r *userRepository) FindRoleByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.NewSelect().Model(&role).Where("name = ?", name).Scan(context.Background())
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *userRepository) FindRolesByNames(names []string) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.NewSelect().Model(&roles).Where("name IN (?)", bun.In(names)).Scan(context.Background())
	return roles, err
}

func (r *userRepository) CreateRole(role *models.Role) error {
	_, err := r.db.NewInsert().Model(role).Exec(context.Background())
	return err
}

func (r *userRepository) GetStarBalance(userID string) (int, error) {
	var result struct {
		Total int `bun:"total"`
	}
	err := r.db.NewSelect().
		TableExpr("star_transactions").
		ColumnExpr("COALESCE(SUM(stars), 0) AS total").
		Where("userId = ?", userID).
		Scan(context.Background(), &result)
	return result.Total, err
}
