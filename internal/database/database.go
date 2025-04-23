package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// sql.DB의 포인터를 담은 구조체
// *sql.DB를 그대로 쓰지 않고 Client로 감싸주어서
// Client에 원하는 method들을 붙여줄 수 있다.
type Client struct {
	db *sql.DB
}

// .db 파일 경로를 받아 *sql.DB를 담은 Client 구조체 반환
func NewClient(pathToDB string) (Client, error) {
	db, err := sql.Open("sqlite3", pathToDB)
	if err != nil {
		return Client{}, err
	}
	c := Client{db}

	// migration(사용되는 테이블들 생성)
	err = c.autoMigrate()
	if err != nil {
		return Client{}, err
	}
	return c, nil

}

// 이전 프로젝트에서 goose를 쓰는것과는 다르게 직접 쿼리를 입력해 migration 실행
// 서버를 최초 실행할 때를 제외하고, 새로 실행할 때마다 새로 테이블 생성하려하면 안되므로 IF NOT EXISTS 키워드 사용
func (c *Client) autoMigrate() error {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		password TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL
	);
	`
	// @@@ id가 UUID가 아니고 TEXT이므로 db에 입력할 떄 uuid를 반드시 string화한 후 입력해야 함

	// sql.DB의 method Exec은 return 받는 row가 없는 쿼리들 실행에 사용
	_, err := c.db.Exec(userTable)
	if err != nil {
		return err
	}
	refreshTokenTable := `
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		token TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		revoked_at TIMESTAMP,
		user_id TEXT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	`
	// @@@ user_id가 UUID가 아니고 TEXT이므로 db에 입력할 떄 uuid를 반드시 string화한 후 입력해야 함

	_, err = c.db.Exec(refreshTokenTable)
	if err != nil {
		return err
	}

	videoTable := `
	CREATE TABLE IF NOT EXISTS videos (
		id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		title TEXT NOT NULL,
		description TEXT,
		thumbnail_url TEXT,
		video_url TEXT TEXT,
		user_id INTEGER,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	`

	_, err = c.db.Exec(videoTable)
	if err != nil {
		return err
	}
	return nil
}

// db의 테이블들 record 전부 삭제하는 함수
func (c Client) Reset() error {
	if _, err := c.db.Exec("DELETE FROM refresh_tokens"); err != nil {
		return fmt.Errorf("failed to reset table refresh_tokens: %w", err)
	}
	if _, err := c.db.Exec("DELETE FROM users"); err != nil {
		return fmt.Errorf("failed to reset table users: %w", err)
	}
	if _, err := c.db.Exec("DELETE FROM videos"); err != nil {
		return fmt.Errorf("failed to reset table videos: %w", err)
	}
	return nil
}
