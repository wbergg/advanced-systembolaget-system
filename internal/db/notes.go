package db

func (db *DB) SetNote(productID string, note string) error {
	if note == "" {
		_, err := db.conn.Exec("DELETE FROM notes WHERE product_id = ?", productID)
		return err
	}
	_, err := db.conn.Exec(`
		INSERT INTO notes (product_id, note, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(product_id) DO UPDATE SET note = excluded.note, updated_at = CURRENT_TIMESTAMP
	`, productID, note)
	return err
}
