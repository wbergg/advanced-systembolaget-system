package db

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (db *DB) GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password, role, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) GetUserByID(id int) (*User, error) {
	u := &User{}
	err := db.conn.QueryRow(
		"SELECT id, username, password, role, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) ListUsers() ([]User, error) {
	rows, err := db.conn.Query("SELECT id, username, role, created_at, updated_at FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *DB) CreateUser(u *User) error {
	res, err := db.conn.Exec(
		"INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
		u.Username, u.Password, u.Role,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	u.ID = int(id)
	return nil
}

func (db *DB) UpdateUser(u *User) error {
	if u.Password != "" {
		_, err := db.conn.Exec(
			"UPDATE users SET username=?, password=?, role=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			u.Username, u.Password, u.Role, u.ID,
		)
		return err
	}
	_, err := db.conn.Exec(
		"UPDATE users SET username=?, role=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
		u.Username, u.Role, u.ID,
	)
	return err
}

func (db *DB) UpdateUserPassword(id int, hashedPassword string) error {
	_, err := db.conn.Exec("UPDATE users SET password=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", hashedPassword, id)
	return err
}

func (db *DB) DeleteUser(id int) error {
	_, err := db.conn.Exec("DELETE FROM users WHERE id=?", id)
	return err
}

func (db *DB) UserCount() (int, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (db *DB) AuditLog(userID int, action, detail string) {
	_, _ = db.conn.Exec("INSERT INTO audit_log (user_id, action, detail) VALUES (?, ?, ?)", userID, action, detail)
}

func (db *DB) SeedAdmin(username, password string) error {
	count, err := db.UserCount()
	if err != nil {
		return fmt.Errorf("seed admin count: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("seed admin hash: %w", err)
	}

	if count == 0 {
		_, err = db.conn.Exec(
			"INSERT INTO users (username, password, role) VALUES (?, ?, 'admin')",
			username, string(hash),
		)
		if err != nil {
			return fmt.Errorf("seed admin insert: %w", err)
		}
		log.Printf("Seeded admin user: %s", username)
	}

	return nil
}
