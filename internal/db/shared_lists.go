package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type SharedListShareUser struct {
	UserID   int    `json:"userId"`
	Username string `json:"username"`
}

type SharedList struct {
	ID         int                   `json:"id"`
	UUID       string                `json:"uuid"`
	Name       string                `json:"name"`
	UserID     int                   `json:"userId"`
	OwnerName  string                `json:"ownerName"`
	Shared     bool                  `json:"shared"`
	Locked     bool                  `json:"locked"`
	SharedWith []SharedListShareUser `json:"sharedWith,omitempty"`
	ItemCount  int                   `json:"itemCount"`
	Total      float64               `json:"total"`
	CreatedAt  string                `json:"createdAt"`
	Items      []SharedListItem      `json:"items,omitempty"`
}

type SharedListItem struct {
	ProductID       string  `json:"productId"`
	ProductNumber   string  `json:"productNumber"`
	ProductNameBold string  `json:"productNameBold"`
	ProductNameThin *string `json:"productNameThin"`
	ProducerName    string  `json:"producerName"`
	Price           float64 `json:"price"`
	Volume          float64 `json:"volume"`
	VolumeText      string  `json:"volumeText"`
	AlcoholPct      float64 `json:"alcoholPercentage"`
	Country         string  `json:"country"`
	CategoryLevel1  string  `json:"categoryLevel1"`
	CategoryLevel2  string  `json:"categoryLevel2"`
	PackagingLevel1 string  `json:"packagingLevel1"`
	ImageURL        string  `json:"imageUrl"`
	Taste           string  `json:"taste"`
	Usage           string  `json:"usage"`
	IsOrganic       bool    `json:"isOrganic"`
	Quantity        int     `json:"quantity"`
	AddedBy         string  `json:"addedBy"`
	AddedAt         string  `json:"addedAt"`
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

func (db *DB) CreateSharedList(name string, userID int) (SharedList, error) {
	uuid := generateUUID()
	now := time.Now().Format(time.RFC3339)
	res, err := db.conn.Exec(
		`INSERT INTO shared_lists (uuid, name, user_id, created_at) VALUES (?, ?, ?, ?)`,
		uuid, name, userID, now,
	)
	if err != nil {
		return SharedList{}, err
	}
	id, _ := res.LastInsertId()
	return SharedList{
		ID:        int(id),
		UUID:      uuid,
		Name:      name,
		UserID:    userID,
		ItemCount: 0,
		CreatedAt: now,
	}, nil
}

func (db *DB) canAccessSharedList(listID, userID int) (isOwner bool, err error) {
	var ownerID int
	err = db.conn.QueryRow("SELECT user_id FROM shared_lists WHERE id = ?", listID).Scan(&ownerID)
	if err != nil {
		return false, fmt.Errorf("list not found")
	}
	if ownerID == userID {
		return true, nil
	}
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM shared_list_shares WHERE list_id = ? AND user_id = ?", listID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	return false, fmt.Errorf("list not found")
}

func (db *DB) getSharedListCollaborators(listID int) ([]SharedListShareUser, error) {
	rows, err := db.conn.Query(`
		SELECT sls.user_id, u.username
		FROM shared_list_shares sls
		JOIN users u ON u.id = sls.user_id
		WHERE sls.list_id = ?
		ORDER BY u.username
	`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []SharedListShareUser
	for rows.Next() {
		var u SharedListShareUser
		if err := rows.Scan(&u.UserID, &u.Username); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (db *DB) ListSharedLists(userID int) ([]SharedList, error) {
	rows, err := db.conn.Query(`
		SELECT sl.id, sl.uuid, sl.name, sl.user_id, u.username,
			sl.locked,
			COALESCE(SUM(sli.quantity), 0),
			COALESCE(SUM(sli.quantity * p.price), 0),
			sl.created_at
		FROM shared_lists sl
		JOIN users u ON u.id = sl.user_id
		LEFT JOIN shared_list_items sli ON sli.list_id = sl.id
		LEFT JOIN products p ON p.product_id = sli.product_id
		WHERE sl.user_id = ?
		   OR sl.id IN (SELECT list_id FROM shared_list_shares WHERE user_id = ?)
		GROUP BY sl.id
		ORDER BY sl.created_at DESC
	`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []SharedList
	for rows.Next() {
		var l SharedList
		if err := rows.Scan(&l.ID, &l.UUID, &l.Name, &l.UserID, &l.OwnerName, &l.Locked, &l.ItemCount, &l.Total, &l.CreatedAt); err != nil {
			return nil, err
		}
		l.Shared = l.UserID != userID
		sw, _ := db.getSharedListCollaborators(l.ID)
		l.SharedWith = sw
		lists = append(lists, l)
	}
	return lists, nil
}

func (db *DB) GetSharedList(id, userID int) (SharedList, error) {
	_, err := db.canAccessSharedList(id, userID)
	if err != nil {
		return SharedList{}, err
	}

	var l SharedList
	err = db.conn.QueryRow(`
		SELECT sl.id, sl.uuid, sl.name, sl.user_id, u.username, sl.locked, sl.created_at
		FROM shared_lists sl
		JOIN users u ON u.id = sl.user_id
		WHERE sl.id = ?
	`, id).Scan(&l.ID, &l.UUID, &l.Name, &l.UserID, &l.OwnerName, &l.Locked, &l.CreatedAt)
	if err != nil {
		return SharedList{}, err
	}

	l.Shared = l.UserID != userID
	sw, _ := db.getSharedListCollaborators(id)
	l.SharedWith = sw

	items, err := db.getSharedListItems(id)
	if err != nil {
		return SharedList{}, err
	}
	l.Items = items
	l.ItemCount = len(items)
	for _, item := range items {
		l.Total += float64(item.Quantity) * item.Price
	}
	return l, nil
}

func (db *DB) GetSharedListByUUID(uuid string) (SharedList, error) {
	var l SharedList
	err := db.conn.QueryRow(`
		SELECT sl.id, sl.uuid, sl.name, sl.user_id, u.username, sl.locked, sl.created_at
		FROM shared_lists sl
		JOIN users u ON u.id = sl.user_id
		WHERE sl.uuid = ?
	`, uuid).Scan(&l.ID, &l.UUID, &l.Name, &l.UserID, &l.OwnerName, &l.Locked, &l.CreatedAt)
	if err != nil {
		return SharedList{}, err
	}

	items, err := db.getSharedListItems(l.ID)
	if err != nil {
		return SharedList{}, err
	}
	l.Items = items
	l.ItemCount = len(items)
	for _, item := range items {
		l.Total += float64(item.Quantity) * item.Price
	}
	return l, nil
}

func (db *DB) getSharedListItems(listID int) ([]SharedListItem, error) {
	rows, err := db.conn.Query(`
		SELECT sli.product_id, p.product_number, p.name_bold, p.name_thin, p.producer_name,
			p.price, p.volume, p.volume_text, p.alcohol_pct,
			p.country, p.category_level1, p.category_level2,
			p.packaging_level1, p.image_url, p.taste, p.usage, p.is_organic,
			sli.quantity, COALESCE(u.username, ''), sli.added_at
		FROM shared_list_items sli
		JOIN products p ON p.product_id = sli.product_id
		LEFT JOIN users u ON u.id = sli.added_by
		WHERE sli.list_id = ?
		ORDER BY sli.added_at
	`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SharedListItem
	for rows.Next() {
		var item SharedListItem
		var nameThin sql.NullString
		var taste, usage sql.NullString
		var isOrganic int
		if err := rows.Scan(
			&item.ProductID, &item.ProductNumber, &item.ProductNameBold, &nameThin, &item.ProducerName,
			&item.Price, &item.Volume, &item.VolumeText, &item.AlcoholPct,
			&item.Country, &item.CategoryLevel1, &item.CategoryLevel2,
			&item.PackagingLevel1, &item.ImageURL, &taste, &usage, &isOrganic,
			&item.Quantity, &item.AddedBy, &item.AddedAt,
		); err != nil {
			return nil, err
		}
		if nameThin.Valid {
			item.ProductNameThin = &nameThin.String
		}
		if taste.Valid {
			item.Taste = taste.String
		}
		if usage.Valid {
			item.Usage = usage.String
		}
		item.IsOrganic = isOrganic == 1
		items = append(items, item)
	}
	return items, nil
}

func (db *DB) isSharedListLocked(listID int) bool {
	var locked bool
	db.conn.QueryRow("SELECT locked FROM shared_lists WHERE id = ?", listID).Scan(&locked)
	return locked
}

func (db *DB) AddToSharedList(listID int, productID string, quantity int, userID int) error {
	_, err := db.canAccessSharedList(listID, userID)
	if err != nil {
		return err
	}
	if db.isSharedListLocked(listID) {
		return fmt.Errorf("list is locked")
	}
	_, err = db.conn.Exec(`
		INSERT INTO shared_list_items (list_id, product_id, quantity, added_by, added_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(list_id, product_id) DO UPDATE SET quantity = quantity + ?
	`, listID, productID, quantity, userID, time.Now().Format(time.RFC3339), quantity)
	return err
}

func (db *DB) RemoveFromSharedList(listID int, productID string, userID int) error {
	_, err := db.canAccessSharedList(listID, userID)
	if err != nil {
		return err
	}
	if db.isSharedListLocked(listID) {
		return fmt.Errorf("list is locked")
	}
	res, err := db.conn.Exec(
		`DELETE FROM shared_list_items WHERE list_id = ? AND product_id = ?`,
		listID, productID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("item not found")
	}
	return nil
}

func (db *DB) UpdateSharedListItemQuantity(listID int, productID string, quantity int, userID int) error {
	_, err := db.canAccessSharedList(listID, userID)
	if err != nil {
		return err
	}
	if db.isSharedListLocked(listID) {
		return fmt.Errorf("list is locked")
	}
	if quantity <= 0 {
		return db.RemoveFromSharedList(listID, productID, userID)
	}
	_, err = db.conn.Exec(
		"UPDATE shared_list_items SET quantity = ? WHERE list_id = ? AND product_id = ?",
		quantity, listID, productID)
	return err
}

func (db *DB) RenameSharedList(id int, name string, userID int) error {
	res, err := db.conn.Exec("UPDATE shared_lists SET name = ? WHERE id = ? AND user_id = ?", name, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("list not found or not owned by you")
	}
	return nil
}

func (db *DB) SetSharedListLocked(listID int, locked bool, userID int, isAdmin bool) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM shared_lists WHERE id = ?", listID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("list not found")
	}
	if ownerID != userID && !isAdmin {
		return fmt.Errorf("only the owner or admin can lock/unlock a list")
	}
	val := 0
	if locked {
		val = 1
	}
	_, err = db.conn.Exec("UPDATE shared_lists SET locked = ? WHERE id = ?", val, listID)
	return err
}

func (db *DB) DeleteSharedList(id, userID int) error {
	res, err := db.conn.Exec(
		`DELETE FROM shared_lists WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("list not found or not owned by user")
	}
	return nil
}

func (db *DB) ShareSharedList(listID, ownerID, targetUserID int) error {
	if ownerID == targetUserID {
		return fmt.Errorf("cannot share with yourself")
	}
	var actualOwner int
	err := db.conn.QueryRow("SELECT user_id FROM shared_lists WHERE id = ?", listID).Scan(&actualOwner)
	if err != nil {
		return fmt.Errorf("list not found")
	}
	if actualOwner != ownerID {
		return fmt.Errorf("only the owner can share a list")
	}
	_, err = db.conn.Exec(`
		INSERT INTO shared_list_shares (list_id, user_id) VALUES (?, ?)
		ON CONFLICT DO NOTHING
	`, listID, targetUserID)
	return err
}

func (db *DB) UnshareSharedList(listID, callerID, targetUserID int) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM shared_lists WHERE id = ?", listID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("list not found")
	}
	if callerID != ownerID && callerID != targetUserID {
		return fmt.Errorf("not allowed")
	}
	_, err = db.conn.Exec("DELETE FROM shared_list_shares WHERE list_id = ? AND user_id = ?", listID, targetUserID)
	return err
}
