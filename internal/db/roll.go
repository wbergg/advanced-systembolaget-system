package db

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

type RollPoolItem struct {
	ID              int     `json:"id"`
	ProductID       string  `json:"productId"`
	ProductNameBold string  `json:"productNameBold"`
	ProductNameThin *string `json:"productNameThin"`
	ProducerName    string  `json:"producerName"`
	Country         string  `json:"country"`
	ImageURL        string  `json:"imageUrl"`
	Consumed        bool    `json:"consumed"`
	ConsumedByUID   *int    `json:"consumedByUserId,omitempty"`
	ConsumedByName  *string `json:"consumedByName,omitempty"`
	ConsumedAt      *string `json:"consumedAt,omitempty"`
	Vetoed          bool    `json:"vetoed"`
}

type RollTurn struct {
	ID              int     `json:"id"`
	EventID         int     `json:"eventId"`
	PoolID          int     `json:"poolId"`
	UserID          int     `json:"userId"`
	Username        string  `json:"username"`
	ProductNameBold string  `json:"productNameBold"`
	ProductNameThin *string `json:"productNameThin"`
	ProducerName    string  `json:"producerName"`
	Country         string  `json:"country"`
	ImageURL        string  `json:"imageUrl"`
	Status          string  `json:"status"`
	CanVeto         bool    `json:"canVeto"`
	CreatedAt       string  `json:"createdAt"`
	ResolvedAt      *string `json:"resolvedAt,omitempty"`
}

type VetoedItem struct {
	PoolID          int     `json:"poolId"`
	ProductNameBold string  `json:"productNameBold"`
	ProductNameThin *string `json:"productNameThin"`
	ProducerName    string  `json:"producerName"`
	Country         string  `json:"country"`
	ImageURL        string  `json:"imageUrl"`
	VetoedByName    string  `json:"vetoedByName"`
	VetoedAt        string  `json:"vetoedAt"`
}

type RollState struct {
	PoolCount   int            `json:"poolCount"`
	TotalCount  int            `json:"totalCount"`
	Consumed    []RollPoolItem `json:"consumed"`
	Vetoed      []VetoedItem   `json:"vetoed"`
	PendingTurn *RollTurn      `json:"pendingTurn"`
	UserVetoes  map[int]bool   `json:"userVetoes"`
	Finished    bool           `json:"finished"`
}

func (db *DB) GetRollState(eventID int) (*RollState, error) {
	state := &RollState{
		Consumed:   []RollPoolItem{},
		Vetoed:     []VetoedItem{},
		UserVetoes: make(map[int]bool),
	}

	// Total + remaining counts
	if err := db.conn.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN consumed = 0 THEN 1 ELSE 0 END), 0)
		FROM roll_pool WHERE event_id = ?
	`, eventID).Scan(&state.TotalCount, &state.PoolCount); err != nil {
		return nil, err
	}

	// Consumed items
	rows, err := db.conn.Query(`
		SELECT rp.id, rp.product_id, p.name_bold, p.name_thin, p.producer_name, p.country, COALESCE(p.image_url, ''),
			rp.consumed_by, u.username, rp.consumed_at, rp.vetoed
		FROM roll_pool rp
		JOIN products p ON rp.product_id = p.product_id
		LEFT JOIN users u ON rp.consumed_by = u.id
		WHERE rp.event_id = ? AND rp.consumed = 1
		ORDER BY rp.consumed_at DESC
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item RollPoolItem
		item.Consumed = true
		if err := rows.Scan(&item.ID, &item.ProductID, &item.ProductNameBold, &item.ProductNameThin,
			&item.ProducerName, &item.Country, &item.ImageURL,
			&item.ConsumedByUID, &item.ConsumedByName, &item.ConsumedAt, &item.Vetoed); err != nil {
			return nil, err
		}
		state.Consumed = append(state.Consumed, item)
	}

	// Vetoed items (pool entries that were vetoed and are back in the pool)
	vetoRows, err := db.conn.Query(`
		SELECT rp.id, p.name_bold, p.name_thin, p.producer_name, p.country, COALESCE(p.image_url, ''),
			u.username, rt.resolved_at
		FROM roll_turns rt
		JOIN roll_pool rp ON rt.pool_id = rp.id
		JOIN products p ON rp.product_id = p.product_id
		JOIN users u ON rt.user_id = u.id
		WHERE rt.event_id = ? AND rt.status = 'vetoed'
		ORDER BY rt.resolved_at DESC
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer vetoRows.Close()
	for vetoRows.Next() {
		var v VetoedItem
		if err := vetoRows.Scan(&v.PoolID, &v.ProductNameBold, &v.ProductNameThin,
			&v.ProducerName, &v.Country, &v.ImageURL, &v.VetoedByName, &v.VetoedAt); err != nil {
			return nil, err
		}
		state.Vetoed = append(state.Vetoed, v)
	}

	// Pending turn (with veto info inlined)
	var turn RollTurn
	var resolvedAt *string
	var poolVetoed bool
	var userVetoCount int
	pendingErr := db.conn.QueryRow(`
		SELECT rt.id, rt.event_id, rt.pool_id, rt.user_id, u.username,
			p.name_bold, p.name_thin, p.producer_name, p.country, COALESCE(p.image_url, ''),
			rt.status, rt.created_at, rt.resolved_at,
			rp.vetoed,
			(SELECT COUNT(*) FROM roll_turns rt2
			 WHERE rt2.event_id = rt.event_id AND rt2.user_id = rt.user_id
			   AND rt2.status = 'vetoed') AS user_veto_count
		FROM roll_turns rt
		JOIN users u ON rt.user_id = u.id
		JOIN roll_pool rp ON rt.pool_id = rp.id
		JOIN products p ON rp.product_id = p.product_id
		WHERE rt.event_id = ? AND rt.status = 'pending'
		LIMIT 1
	`, eventID).Scan(&turn.ID, &turn.EventID, &turn.PoolID, &turn.UserID, &turn.Username,
		&turn.ProductNameBold, &turn.ProductNameThin, &turn.ProducerName, &turn.Country, &turn.ImageURL,
		&turn.Status, &turn.CreatedAt, &resolvedAt,
		&poolVetoed, &userVetoCount)
	if pendingErr == nil {
		turn.ResolvedAt = resolvedAt
		turn.CanVeto = userVetoCount == 0 && !poolVetoed
		state.PendingTurn = &turn
	}

	// User vetoes: which users have used their veto
	userVetoRows, err := db.conn.Query(`
		SELECT DISTINCT user_id FROM roll_turns WHERE event_id = ? AND status = 'vetoed'
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer userVetoRows.Close()
	for userVetoRows.Next() {
		var uid int
		if err := userVetoRows.Scan(&uid); err != nil {
			return nil, err
		}
		state.UserVetoes[uid] = true
	}

	state.Finished = state.PoolCount == 0 && state.PendingTurn == nil

	return state, nil
}

func (db *DB) PerformRoll(eventID, targetUserID int) (*RollTurn, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Verify event is a roll event and not locked
	var eventType string
	var locked bool
	err = tx.QueryRow("SELECT type, locked FROM events WHERE id = ?", eventID).Scan(&eventType, &locked)
	if err != nil {
		return nil, fmt.Errorf("event not found")
	}
	if eventType != "roll" {
		return nil, fmt.Errorf("not a roll event")
	}
	if locked {
		return nil, fmt.Errorf("event is locked")
	}

	// Check no pending turn
	var pendingCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM roll_turns WHERE event_id = ? AND status = 'pending'", eventID).Scan(&pendingCount); err != nil {
		return nil, err
	}
	if pendingCount > 0 {
		return nil, fmt.Errorf("a roll is already pending")
	}

	// Get available pool entries
	type poolEntry struct {
		id        int
		productID string
		vetoed    bool
	}
	poolRows, err := tx.Query("SELECT id, product_id, vetoed FROM roll_pool WHERE event_id = ? AND consumed = 0", eventID)
	if err != nil {
		return nil, err
	}
	defer poolRows.Close()
	var pool []poolEntry
	for poolRows.Next() {
		var e poolEntry
		if err := poolRows.Scan(&e.id, &e.productID, &e.vetoed); err != nil {
			return nil, err
		}
		pool = append(pool, e)
	}
	if len(pool) == 0 {
		return nil, fmt.Errorf("game finished — no beers remaining")
	}

	// Pick random entry
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))
	if err != nil {
		return nil, fmt.Errorf("random selection failed: %w", err)
	}
	picked := pool[idx.Int64()]

	// Insert turn
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.Exec(
		"INSERT INTO roll_turns (event_id, pool_id, user_id, status, created_at) VALUES (?, ?, ?, 'pending', ?)",
		eventID, picked.id, targetUserID, now)
	if err != nil {
		return nil, err
	}
	turnID, _ := res.LastInsertId()

	// Compute canVeto
	var vetoCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM roll_turns WHERE event_id = ? AND user_id = ? AND status = 'vetoed'",
		eventID, targetUserID).Scan(&vetoCount); err != nil {
		return nil, err
	}
	canVeto := vetoCount == 0 && !picked.vetoed

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Fetch product details for the response
	var turn RollTurn
	turn.ID = int(turnID)
	turn.EventID = eventID
	turn.PoolID = picked.id
	turn.UserID = targetUserID
	turn.Status = "pending"
	turn.CanVeto = canVeto
	turn.CreatedAt = now

	if err := db.conn.QueryRow("SELECT username FROM users WHERE id = ?", targetUserID).Scan(&turn.Username); err != nil {
		return nil, fmt.Errorf("fetch username: %w", err)
	}
	if err := db.conn.QueryRow("SELECT name_bold, name_thin, producer_name, country, COALESCE(image_url, '') FROM products WHERE product_id = ?",
		picked.productID).Scan(&turn.ProductNameBold, &turn.ProductNameThin, &turn.ProducerName, &turn.Country, &turn.ImageURL); err != nil {
		return nil, fmt.Errorf("fetch product: %w", err)
	}

	return &turn, nil
}

func (db *DB) GetRollTurn(eventID, turnID int) (*RollTurn, error) {
	var turn RollTurn
	var resolvedAt *string
	err := db.conn.QueryRow(`
		SELECT rt.id, rt.event_id, rt.pool_id, rt.user_id, u.username,
			p.name_bold, p.name_thin, p.producer_name, p.country, COALESCE(p.image_url, ''),
			rt.status, rt.created_at, rt.resolved_at
		FROM roll_turns rt
		JOIN users u ON rt.user_id = u.id
		JOIN roll_pool rp ON rt.pool_id = rp.id
		JOIN products p ON rp.product_id = p.product_id
		WHERE rt.event_id = ? AND rt.id = ?
	`, eventID, turnID).Scan(&turn.ID, &turn.EventID, &turn.PoolID, &turn.UserID, &turn.Username,
		&turn.ProductNameBold, &turn.ProductNameThin, &turn.ProducerName, &turn.Country, &turn.ImageURL,
		&turn.Status, &turn.CreatedAt, &resolvedAt)
	if err != nil {
		return nil, err
	}
	turn.ResolvedAt = resolvedAt
	return &turn, nil
}

func (db *DB) AcceptRoll(eventID, turnID int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var poolID, userID int
	var status string
	err = tx.QueryRow("SELECT pool_id, user_id, status FROM roll_turns WHERE id = ? AND event_id = ?", turnID, eventID).Scan(&poolID, &userID, &status)
	if err != nil {
		return fmt.Errorf("turn not found")
	}
	if status != "pending" {
		return fmt.Errorf("turn is not pending")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.Exec("UPDATE roll_turns SET status = 'accepted', resolved_at = ? WHERE id = ?", now, turnID)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE roll_pool SET consumed = 1, consumed_by = ?, consumed_at = ? WHERE id = ?", userID, now, poolID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) VetoRoll(eventID, turnID int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var poolID, userID int
	var status string
	err = tx.QueryRow("SELECT pool_id, user_id, status FROM roll_turns WHERE id = ? AND event_id = ?", turnID, eventID).Scan(&poolID, &userID, &status)
	if err != nil {
		return fmt.Errorf("turn not found")
	}
	if status != "pending" {
		return fmt.Errorf("turn is not pending")
	}

	// Check user hasn't already used veto
	var vetoCount int
	if err := tx.QueryRow("SELECT COUNT(*) FROM roll_turns WHERE event_id = ? AND user_id = ? AND status = 'vetoed'",
		eventID, userID).Scan(&vetoCount); err != nil {
		return fmt.Errorf("check veto count: %w", err)
	}
	if vetoCount > 0 {
		return fmt.Errorf("veto already used")
	}

	// Check pool entry isn't already veto-immune
	var poolVetoed bool
	if err := tx.QueryRow("SELECT vetoed FROM roll_pool WHERE id = ?", poolID).Scan(&poolVetoed); err != nil {
		return fmt.Errorf("check pool veto: %w", err)
	}
	if poolVetoed {
		return fmt.Errorf("this beer has already been vetoed and cannot be vetoed again")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.Exec("UPDATE roll_turns SET status = 'vetoed', resolved_at = ? WHERE id = ?", now, turnID)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE roll_pool SET vetoed = 1 WHERE id = ?", poolID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) UndoVeto(eventID, poolID int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var vetoed bool
	err = tx.QueryRow("SELECT vetoed FROM roll_pool WHERE id = ? AND event_id = ?", poolID, eventID).Scan(&vetoed)
	if err != nil {
		return fmt.Errorf("pool entry not found")
	}
	if !vetoed {
		return fmt.Errorf("beer is not vetoed")
	}

	_, err = tx.Exec("UPDATE roll_pool SET vetoed = 0 WHERE id = ?", poolID)
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM roll_turns WHERE event_id = ? AND pool_id = ? AND status = 'vetoed'", eventID, poolID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) UndoConsumed(eventID, poolID int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Verify pool entry belongs to this event and is consumed
	var consumed bool
	err = tx.QueryRow("SELECT consumed FROM roll_pool WHERE id = ? AND event_id = ?", poolID, eventID).Scan(&consumed)
	if err != nil {
		return fmt.Errorf("pool entry not found")
	}
	if !consumed {
		return fmt.Errorf("beer is not consumed")
	}

	// Revert pool entry
	_, err = tx.Exec("UPDATE roll_pool SET consumed = 0, consumed_by = NULL, consumed_at = NULL WHERE id = ?", poolID)
	if err != nil {
		return err
	}

	// Remove the accepted turn that consumed this pool entry
	_, err = tx.Exec("DELETE FROM roll_turns WHERE event_id = ? AND pool_id = ? AND status = 'accepted'", eventID, poolID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) ImportSharedListToRollPool(eventID, listID, userID int, isAdmin bool) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM events WHERE id = ?", eventID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("event not found")
	}
	if ownerID != userID && !isAdmin {
		return fmt.Errorf("only the owner or admin can import lists")
	}
	_, err = db.canAccessSharedList(listID, userID)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec(`
		INSERT INTO roll_pool (event_id, product_id)
		SELECT ?, product_id FROM shared_list_items WHERE list_id = ?
		AND product_id NOT IN (SELECT product_id FROM roll_pool WHERE event_id = ?)
	`, eventID, listID, eventID)
	return err
}

func (db *DB) ReplaceRollPoolWithSharedList(eventID, listID, userID int, isAdmin bool) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM events WHERE id = ?", eventID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("event not found")
	}
	if ownerID != userID && !isAdmin {
		return fmt.Errorf("only the owner or admin can import lists")
	}
	if _, err := db.canAccessSharedList(listID, userID); err != nil {
		return err
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM roll_turns WHERE event_id = ?", eventID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM roll_pool WHERE event_id = ?", eventID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO roll_pool (event_id, product_id)
		SELECT ?, product_id FROM shared_list_items WHERE list_id = ?
	`, eventID, listID); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) ResetRoll(eventID int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM roll_turns WHERE event_id = ?", eventID)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE roll_pool SET consumed = 0, consumed_by = NULL, consumed_at = NULL, vetoed = 0 WHERE event_id = ?", eventID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) SetEventHidden(eventID int, hidden bool, callerUserID int, isAdmin bool) error {
	if !isAdmin {
		return fmt.Errorf("only admins can change event visibility")
	}
	val := 0
	if hidden {
		val = 1
	}
	_, err := db.conn.Exec("UPDATE events SET hidden = ? WHERE id = ?", val, eventID)
	return err
}
