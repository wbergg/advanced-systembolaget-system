package db

import "database/sql"

type Comment struct {
	ID        int    `json:"id"`
	ProductID string `json:"productId"`
	UserID    int    `json:"userId"`
	Username  string `json:"username"`
	Comment   string `json:"comment"`
	CreatedAt string `json:"createdAt"`
}

func (db *DB) AddComment(productID string, userID int, comment string) (Comment, error) {
	res, err := db.conn.Exec(`
		INSERT INTO comments (product_id, user_id, comment) VALUES (?, ?, ?)
	`, productID, userID, comment)
	if err != nil {
		return Comment{}, err
	}
	id, _ := res.LastInsertId()

	var c Comment
	err = db.conn.QueryRow(`
		SELECT c.id, c.product_id, c.user_id, u.username, c.comment, c.created_at
		FROM comments c JOIN users u ON c.user_id = u.id
		WHERE c.id = ?
	`, id).Scan(&c.ID, &c.ProductID, &c.UserID, &c.Username, &c.Comment, &c.CreatedAt)
	return c, err
}

func (db *DB) GetComments(productID string) ([]Comment, error) {
	rows, err := db.conn.Query(`
		SELECT c.id, c.product_id, c.user_id, u.username, c.comment, c.created_at
		FROM comments c JOIN users u ON c.user_id = u.id
		WHERE c.product_id = ?
		ORDER BY c.created_at DESC
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ProductID, &c.UserID, &c.Username, &c.Comment, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (db *DB) DeleteComment(id int) error {
	res, err := db.conn.Exec("DELETE FROM comments WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
