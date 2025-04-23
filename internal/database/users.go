package database

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreateUserParams
}

type CreateUserParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c Client) GetUsers() ([]User, error) {
	query := `
		SELECT
			id,
			email
		FROM users
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var user User
		var id string
		if err := rows.Scan(&id, &user.Email); err != nil {
			return nil, err
		}
		user.ID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (c Client) GetUserByEmail(email string) (User, error) {
	query := `
		SELECT id, created_at, updated_at, email, password
		FROM users
		WHERE email = ?
	`
	var user User
	var id string
	err := c.db.QueryRow(query, email).Scan(&id, &user.CreatedAt, &user.UpdatedAt, &user.Email, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, nil
		}
		return User{}, err
	}
	user.ID, err = uuid.Parse(id)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (c Client) GetUserByRefreshToken(token string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.created_at, u.updated_at, u.password
		FROM users u
		JOIN refresh_tokens rt ON u.id = rt.user_id
		WHERE rt.token = ?
	`
	// http 서버 과정에서는 where 구문에
	// 추가로 AND revoked_at IS NULL AND expires_at > NOW(); 을 써서 만료, 파기 여부도 확인함
	// @@@ sqlite는 NOW()가 없으므로 대신 expires_at > CURRENT_TIMESTAMP; 써야함

	var user User
	var id string
	err := c.db.QueryRow(query, token).Scan(&id, &user.Email, &user.CreatedAt, &user.UpdatedAt, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	user.ID, err = uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (c Client) CreateUser(params CreateUserParams) (*User, error) {
	id := uuid.New()

	query := `
		INSERT INTO users
		    (id, created_at, updated_at, email, password)
		VALUES
		    (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?)
	`
	// @@@ sqlite는 NOW()없고 대신 CURRENT_TIMESTAMP 써야함

	_, err := c.db.Exec(query, id.String(), params.Email, params.Password)
	// @@@ users 테이블의 id는 UUID가 아니고 TEXT이므로 db에 입력할 떄 uuid를 반드시 string화한 후 입력해야 함
	if err != nil {
		return nil, err
	}

	return c.GetUser(id)
}

func (c Client) GetUser(id uuid.UUID) (*User, error) {
	query := `
		SELECT id, created_at, updated_at, email, password
		FROM users
		WHERE id = ?
	`
	var user User
	var idStr string
	err := c.db.QueryRow(query, id.String()).Scan(&idStr, &user.CreatedAt, &user.UpdatedAt, &user.Email, &user.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	user.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c Client) DeleteUser(id uuid.UUID) error {
	query := `
		DELETE FROM users
		WHERE id = ?
	`
	_, err := c.db.Exec(query, id.String())
	return err
}
