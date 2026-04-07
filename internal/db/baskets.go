package db

import (
	"fmt"
	"time"
)

type Basket struct {
	ID         int          `json:"id"`
	Name       string       `json:"name"`
	OwnerID    int          `json:"ownerId"`
	OwnerName  string       `json:"ownerName"`
	Shared     bool         `json:"shared"`
	Locked     bool         `json:"locked"`
	SharedWith []ShareUser  `json:"sharedWith,omitempty"`
	CreatedAt  time.Time    `json:"createdAt"`
	Items      []BasketItem `json:"items,omitempty"`
	ItemCount  int          `json:"itemCount"`
	Total      float64      `json:"total"`
}

type ShareUser struct {
	UserID   int    `json:"userId"`
	Username string `json:"username"`
}

type BasketItem struct {
	ProductID        string  `json:"productId"`
	ProductNameBold  string  `json:"productNameBold"`
	ProductNameThin  *string `json:"productNameThin"`
	ProducerName     string  `json:"producerName"`
	Price            float64 `json:"price"`
	VolumeText       string  `json:"volumeText"`
	AlcoholPercent   float64 `json:"alcoholPercentage"`
	ImageURL         string  `json:"imageUrl"`
	Quantity         int     `json:"quantity"`
	AddedBy          string  `json:"addedBy"`
}

// ListBaskets returns baskets owned by or shared with the given user.
func (db *DB) ListBaskets(userID int) ([]Basket, error) {
	rows, err := db.conn.Query(`
		SELECT b.id, b.name, b.user_id, COALESCE(u.username, ''),
			CASE WHEN b.user_id = ? THEN 0 ELSE 1 END AS shared,
			b.locked,
			COALESCE(SUM(bi.quantity), 0),
			COALESCE(SUM(bi.quantity * p.price), 0)
		FROM baskets b
		LEFT JOIN users u ON b.user_id = u.id
		LEFT JOIN basket_items bi ON b.id = bi.basket_id
		LEFT JOIN products p ON bi.product_id = p.product_id
		WHERE b.user_id = ?
		   OR b.id IN (SELECT basket_id FROM basket_shares WHERE user_id = ?)
		GROUP BY b.id
		ORDER BY shared ASC, b.created_at DESC
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var baskets []Basket
	for rows.Next() {
		var b Basket
		if err := rows.Scan(&b.ID, &b.Name, &b.OwnerID, &b.OwnerName, &b.Shared, &b.Locked, &b.ItemCount, &b.Total); err != nil {
			return nil, err
		}
		baskets = append(baskets, b)
	}
	return baskets, rows.Err()
}

func (db *DB) CreateBasket(name string, userID int) (*Basket, error) {
	res, err := db.conn.Exec("INSERT INTO baskets (name, user_id) VALUES (?, ?)", name, userID)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Basket{ID: int(id), Name: name, OwnerID: userID, CreatedAt: time.Now()}, nil
}

// canAccessBasket checks if the user owns or has been shared the basket.
func (db *DB) canAccessBasket(basketID, userID int) (isOwner bool, err error) {
	var ownerID int
	err = db.conn.QueryRow("SELECT user_id FROM baskets WHERE id = ?", basketID).Scan(&ownerID)
	if err != nil {
		return false, fmt.Errorf("basket not found")
	}
	if ownerID == userID {
		return true, nil
	}
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM basket_shares WHERE basket_id = ? AND user_id = ?", basketID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil // has access but is not owner
	}
	return false, fmt.Errorf("basket not found")
}

func (db *DB) RenameBasket(id int, name string, userID int) error {
	res, err := db.conn.Exec("UPDATE baskets SET name = ? WHERE id = ? AND user_id = ?", name, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("basket not found or not owned by you")
	}
	return nil
}

func (db *DB) DeleteBasket(id int, userID int) error {
	res, err := db.conn.Exec("DELETE FROM baskets WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("basket not found or not owned by you")
	}
	return nil
}

func (db *DB) GetBasket(id int, userID int) (*Basket, error) {
	_, err := db.canAccessBasket(id, userID)
	if err != nil {
		return nil, err
	}

	var b Basket
	err = db.conn.QueryRow(`
		SELECT b.id, b.name, b.user_id, COALESCE(u.username, ''), b.locked, b.created_at
		FROM baskets b LEFT JOIN users u ON b.user_id = u.id
		WHERE b.id = ?
	`, id).Scan(&b.ID, &b.Name, &b.OwnerID, &b.OwnerName, &b.Locked, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	b.Shared = b.OwnerID != userID

	// Load shared users
	shareRows, err := db.conn.Query(`
		SELECT bs.user_id, u.username
		FROM basket_shares bs JOIN users u ON bs.user_id = u.id
		WHERE bs.basket_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer shareRows.Close()
	for shareRows.Next() {
		var su ShareUser
		if err := shareRows.Scan(&su.UserID, &su.Username); err != nil {
			return nil, err
		}
		b.SharedWith = append(b.SharedWith, su)
	}

	rows, err := db.conn.Query(`
		SELECT bi.product_id, p.name_bold, p.name_thin, p.producer_name,
			p.price, p.volume_text, p.alcohol_pct, p.image_url, bi.quantity,
			COALESCE(u.username, '')
		FROM basket_items bi
		JOIN products p ON bi.product_id = p.product_id
		LEFT JOIN users u ON bi.added_by = u.id
		WHERE bi.basket_id = ?
		ORDER BY bi.added_at
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item BasketItem
		if err := rows.Scan(&item.ProductID, &item.ProductNameBold, &item.ProductNameThin,
			&item.ProducerName, &item.Price, &item.VolumeText, &item.AlcoholPercent, &item.ImageURL, &item.Quantity,
			&item.AddedBy); err != nil {
			return nil, err
		}
		b.Items = append(b.Items, item)
		b.ItemCount += item.Quantity
		b.Total += float64(item.Quantity) * item.Price
	}

	return &b, rows.Err()
}

func (db *DB) isBasketLocked(basketID int) bool {
	var locked bool
	db.conn.QueryRow("SELECT locked FROM baskets WHERE id = ?", basketID).Scan(&locked)
	return locked
}

// SetBasketLocked locks or unlocks a basket. Only owner or admin can do this.
func (db *DB) SetBasketLocked(basketID int, locked bool, userID int, isAdmin bool) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM baskets WHERE id = ?", basketID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("basket not found")
	}
	if ownerID != userID && !isAdmin {
		return fmt.Errorf("only the owner or admin can lock/unlock a basket")
	}
	val := 0
	if locked {
		val = 1
	}
	_, err = db.conn.Exec("UPDATE baskets SET locked = ? WHERE id = ?", val, basketID)
	return err
}

func (db *DB) AddToBasket(basketID int, productID string, quantity int, userID int) error {
	_, err := db.canAccessBasket(basketID, userID)
	if err != nil {
		return err
	}
	if db.isBasketLocked(basketID) {
		return fmt.Errorf("basket is locked")
	}
	if quantity < 1 {
		quantity = 1
	}
	_, err = db.conn.Exec(`
		INSERT INTO basket_items (basket_id, product_id, quantity, added_by)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(basket_id, product_id) DO UPDATE SET quantity = quantity + excluded.quantity
	`, basketID, productID, quantity, userID)
	return err
}

func (db *DB) UpdateBasketItemQuantity(basketID int, productID string, quantity int, userID int) error {
	_, err := db.canAccessBasket(basketID, userID)
	if err != nil {
		return err
	}
	if db.isBasketLocked(basketID) {
		return fmt.Errorf("basket is locked")
	}
	if quantity <= 0 {
		return db.removeFromBasketUnchecked(basketID, productID)
	}
	_, err = db.conn.Exec(
		"UPDATE basket_items SET quantity = ? WHERE basket_id = ? AND product_id = ?",
		quantity, basketID, productID)
	return err
}

func (db *DB) RemoveFromBasket(basketID int, productID string, userID int) error {
	_, err := db.canAccessBasket(basketID, userID)
	if err != nil {
		return err
	}
	if db.isBasketLocked(basketID) {
		return fmt.Errorf("basket is locked")
	}
	return db.removeFromBasketUnchecked(basketID, productID)
}

func (db *DB) removeFromBasketUnchecked(basketID int, productID string) error {
	_, err := db.conn.Exec(
		"DELETE FROM basket_items WHERE basket_id = ? AND product_id = ?",
		basketID, productID)
	return err
}

// ShareBasket shares a basket with another user. Only the owner can share.
func (db *DB) ShareBasket(basketID, ownerID, targetUserID int) error {
	if ownerID == targetUserID {
		return fmt.Errorf("cannot share with yourself")
	}
	var actualOwner int
	err := db.conn.QueryRow("SELECT user_id FROM baskets WHERE id = ?", basketID).Scan(&actualOwner)
	if err != nil {
		return fmt.Errorf("basket not found")
	}
	if actualOwner != ownerID {
		return fmt.Errorf("only the owner can share a basket")
	}
	_, err = db.conn.Exec(`
		INSERT INTO basket_shares (basket_id, user_id) VALUES (?, ?)
		ON CONFLICT DO NOTHING
	`, basketID, targetUserID)
	return err
}

// UnshareBasket removes sharing for a user. Owner can unshare anyone, user can unshare themselves.
func (db *DB) UnshareBasket(basketID, callerID, targetUserID int) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM baskets WHERE id = ?", basketID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("basket not found")
	}
	if callerID != ownerID && callerID != targetUserID {
		return fmt.Errorf("not allowed")
	}
	_, err = db.conn.Exec("DELETE FROM basket_shares WHERE basket_id = ? AND user_id = ?", basketID, targetUserID)
	return err
}
