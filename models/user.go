package models

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID          string    `bun:"id,pk,type:char(36)" json:"id"`
	Name        string    `bun:"name,notnull" json:"name"`
	Email       string    `bun:"email,unique,notnull" json:"email"`
	Password    string    `bun:"password,notnull" json:"-"`
	KeepUpdated bool      `bun:"keepUpdated,default:false" json:"keepUpdated"`
	PhotoURL    *string   `bun:"photoUrl" json:"photoUrl"`
	PhoneNumber *string   `bun:"phoneNumber" json:"phoneNumber"`
	BirthDate   *string   `bun:"birthDate" json:"birthDate"`
	CreatedAt   time.Time `bun:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `bun:"updatedAt" json:"updatedAt"`

	Roles []Role `bun:"m2m:users_roles,join:User=Role" json:"roles"`
}

// UserRole is the pivot table struct for the users_roles many-to-many join.
type UserRole struct {
	bun.BaseModel `bun:"table:users_roles"`

	UserID string `bun:"usersId,pk"`
	User   *User  `bun:"rel:belongs-to,join:usersId=id"`
	RoleID string `bun:"rolesId,pk"`
	Role   *Role  `bun:"rel:belongs-to,join:rolesId=id"`
}
