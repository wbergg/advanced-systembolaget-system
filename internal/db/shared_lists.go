package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type SharedList struct {
	ID        int              `json:"id"`
	UUID      string           `json:"uuid"`
	Name      string           `json:"name"`
	UserID    int              `json:"userId"`
	OwnerName string           `json:"ownerName"`
	ItemCount int              `json:"itemCount"`
	CreatedAt string           `json:"createdAt"`
	Items     []SharedListItem `json:"items,omitempty"`
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

func (db *DB) ListSharedLists(userID int) ([]SharedList, error) {
	rows, err := db.conn.Query(`
		SELECT sl.id, sl.uuid, sl.name, sl.user_id, u.username,
			(SELECT COUNT(*) FROM shared_list_items WHERE list_id = sl.id),
			sl.created_at
		FROM shared_lists sl
		JOIN users u ON u.id = sl.user_id
		WHERE sl.user_id = ?
		ORDER BY sl.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []SharedList
	for rows.Next() {
		var l SharedList
		if err := rows.Scan(&l.ID, &l.UUID, &l.Name, &l.UserID, &l.OwnerName, &l.ItemCount, &l.CreatedAt); err != nil {
			return nil, err
		}
		lists = append(lists, l)
	}
	return lists, nil
}

func (db *DB) GetSharedList(id, userID int) (SharedList, error) {
	var l SharedList
	err := db.conn.QueryRow(`
		SELECT sl.id, sl.uuid, sl.name, sl.user_id, u.username, sl.created_at
		FROM shared_lists sl
		JOIN users u ON u.id = sl.user_id
		WHERE sl.id = ? AND sl.user_id = ?
	`, id, userID).Scan(&l.ID, &l.UUID, &l.Name, &l.UserID, &l.OwnerName, &l.CreatedAt)
	if err != nil {
		return SharedList{}, err
	}

	items, err := db.getSharedListItems(id)
	if err != nil {
		return SharedList{}, err
	}
	l.Items = items
	l.ItemCount = len(items)
	return l, nil
}

func (db *DB) GetSharedListByUUID(uuid string) (SharedList, error) {
	var l SharedList
	err := db.conn.QueryRow(`
		SELECT sl.id, sl.uuid, sl.name, sl.user_id, u.username, sl.created_at
		FROM shared_lists sl
		JOIN users u ON u.id = sl.user_id
		WHERE sl.uuid = ?
	`, uuid).Scan(&l.ID, &l.UUID, &l.Name, &l.UserID, &l.OwnerName, &l.CreatedAt)
	if err != nil {
		return SharedList{}, err
	}

	items, err := db.getSharedListItems(l.ID)
	if err != nil {
		return SharedList{}, err
	}
	l.Items = items
	l.ItemCount = len(items)
	return l, nil
}

func (db *DB) getSharedListItems(listID int) ([]SharedListItem, error) {
	rows, err := db.conn.Query(`
		SELECT sli.product_id, p.product_number, p.name_bold, p.name_thin, p.producer_name,
			p.price, p.volume, p.volume_text, p.alcohol_pct,
			p.country, p.category_level1, p.category_level2,
			p.packaging_level1, p.image_url, p.taste, p.usage, p.is_organic,
			sli.quantity, sli.added_at
		FROM shared_list_items sli
		JOIN products p ON p.product_id = sli.product_id
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
			&item.Quantity, &item.AddedAt,
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

func (db *DB) AddToSharedList(listID int, productID string, quantity int) error {
	_, err := db.conn.Exec(`
		INSERT INTO shared_list_items (list_id, product_id, quantity, added_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(list_id, product_id) DO UPDATE SET quantity = quantity + ?
	`, listID, productID, quantity, time.Now().Format(time.RFC3339), quantity)
	return err
}

func (db *DB) RemoveFromSharedList(listID int, productID string) error {
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

// ImportBasketToSharedList imports basket items into a shared list.
// If an item already exists with the same quantity, it is skipped.
// If it exists with a different quantity, the quantity is updated.
// Returns the number of items imported or updated.
func (db *DB) ImportBasketToSharedList(listID, basketID, userID int) (int, error) {
	// Verify the user owns the shared list
	var listOwner int
	err := db.conn.QueryRow("SELECT user_id FROM shared_lists WHERE id = ?", listID).Scan(&listOwner)
	if err != nil {
		return 0, fmt.Errorf("shared list not found")
	}
	if listOwner != userID {
		return 0, fmt.Errorf("not your shared list")
	}

	// Verify the user can access the basket
	_, err = db.canAccessBasket(basketID, userID)
	if err != nil {
		return 0, err
	}

	// Get basket items
	rows, err := db.conn.Query(
		"SELECT product_id, quantity FROM basket_items WHERE basket_id = ?", basketID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type basketEntry struct {
		productID string
		quantity  int
	}
	var entries []basketEntry
	for rows.Next() {
		var e basketEntry
		if err := rows.Scan(&e.productID, &e.quantity); err != nil {
			return 0, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	now := time.Now().Format(time.RFC3339)
	changed := 0
	for _, e := range entries {
		// Check if item already exists with the same quantity
		var existingQty int
		err := db.conn.QueryRow(
			"SELECT quantity FROM shared_list_items WHERE list_id = ? AND product_id = ?",
			listID, e.productID).Scan(&existingQty)
		if err == sql.ErrNoRows {
			// Insert new item
			_, err = db.conn.Exec(
				"INSERT INTO shared_list_items (list_id, product_id, quantity, added_at) VALUES (?, ?, ?, ?)",
				listID, e.productID, e.quantity, now)
			if err != nil {
				return 0, err
			}
			changed++
		} else if err != nil {
			return 0, err
		} else if existingQty != e.quantity {
			// Update quantity only if different
			_, err = db.conn.Exec(
				"UPDATE shared_list_items SET quantity = ? WHERE list_id = ? AND product_id = ?",
				e.quantity, listID, e.productID)
			if err != nil {
				return 0, err
			}
			changed++
		}
		// else: same quantity, skip
	}

	return changed, nil
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
