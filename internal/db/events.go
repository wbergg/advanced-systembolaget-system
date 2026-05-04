package db

import (
	"database/sql"
	"fmt"
	"time"
)

type Event struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	EventDate     string          `json:"eventDate"`
	OwnerID       int             `json:"ownerId"`
	OwnerName     string          `json:"ownerName"`
	Locked        bool            `json:"locked"`
	Type          string          `json:"type"`
	Hidden        bool            `json:"hidden"`
	Public        bool            `json:"public"`
	ArchivedAt    *time.Time      `json:"archivedAt,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	Attendees     []EventAttendee `json:"attendees,omitempty"`
	Beers         []EventBeer     `json:"beers,omitempty"`
	Scores        []EventScore    `json:"scores,omitempty"`
	AttendeeCount int             `json:"attendeeCount"`
	BeerCount     int             `json:"beerCount"`
}

type EventAttendee struct {
	UserID   int    `json:"userId"`
	Username string `json:"username"`
}

type EventBeer struct {
	ID              int     `json:"id"`
	ProductID       string  `json:"productId"`
	ProductNameBold string  `json:"productNameBold"`
	ProductNameThin *string `json:"productNameThin"`
	ProducerName    string  `json:"producerName"`
	ImageURL        string  `json:"imageUrl"`
}

type EventScore struct {
	EventBeerID int `json:"eventBeerId"`
	UserID      int `json:"userId"`
	Score       int `json:"score"`
}

func (db *DB) GetEventType(eventID int, eventType *string) error {
	return db.conn.QueryRow("SELECT type FROM events WHERE id = ?", eventID).Scan(eventType)
}

func (db *DB) CanAccessEvent(eventID, userID int, isAdmin bool) (isOwner bool, err error) {
	if isAdmin {
		return true, nil
	}
	return db.canAccessEvent(eventID, userID)
}

func (db *DB) canAccessEvent(eventID, userID int) (isOwner bool, err error) {
	var ownerID int
	var archivedAt sql.NullTime
	err = db.conn.QueryRow("SELECT user_id, archived_at FROM events WHERE id = ?", eventID).Scan(&ownerID, &archivedAt)
	if err != nil {
		return false, fmt.Errorf("event not found")
	}
	if archivedAt.Valid {
		return false, fmt.Errorf("event not found")
	}
	if ownerID == userID {
		return true, nil
	}
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM event_attendees WHERE event_id = ? AND user_id = ?", eventID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	return false, fmt.Errorf("event not found")
}

func (db *DB) ListEvents(userID int, isAdmin bool) ([]Event, error) {
	adminVal := 0
	if isAdmin {
		adminVal = 1
	}
	rows, err := db.conn.Query(`
		SELECT e.id, e.name, e.description, e.event_date,
			e.user_id, COALESCE(u.username, ''), e.locked,
			e.type, e.hidden, e.public,
			e.archived_at,
			e.created_at,
			(SELECT COUNT(*) FROM event_attendees WHERE event_id = e.id) AS attendee_count,
			CASE WHEN e.type = 'roll'
				THEN (SELECT COUNT(*) FROM roll_pool WHERE event_id = e.id)
				ELSE (SELECT COUNT(*) FROM event_beers WHERE event_id = e.id)
			END AS beer_count
		FROM events e
		LEFT JOIN users u ON e.user_id = u.id
		WHERE (? = 1
		       OR e.user_id = ?
		       OR e.id IN (SELECT event_id FROM event_attendees WHERE user_id = ?))
		  AND (e.hidden = 0 OR ? = 1)
		  AND e.archived_at IS NULL
		ORDER BY e.created_at DESC
	`, adminVal, userID, userID, adminVal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var ev Event
		var archivedAt sql.NullTime
		if err := rows.Scan(&ev.ID, &ev.Name, &ev.Description, &ev.EventDate,
			&ev.OwnerID, &ev.OwnerName, &ev.Locked,
			&ev.Type, &ev.Hidden, &ev.Public,
			&archivedAt,
			&ev.CreatedAt,
			&ev.AttendeeCount, &ev.BeerCount); err != nil {
			return nil, err
		}
		if archivedAt.Valid {
			t := archivedAt.Time
			ev.ArchivedAt = &t
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

func (db *DB) CreateEvent(name, description, eventDate string, userID int, eventType string, hiddenByDefault bool) (*Event, error) {
	if eventType == "" {
		eventType = "tasting"
	}
	hidden := 0
	if eventType == "roll" && hiddenByDefault {
		hidden = 1
	}

	res, err := db.conn.Exec(
		"INSERT INTO events (name, description, event_date, user_id, type, hidden) VALUES (?, ?, ?, ?, ?, ?)",
		name, description, eventDate, userID, eventType, hidden)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	return &Event{
		ID: int(id), Name: name, Description: description, EventDate: eventDate,
		OwnerID: userID, Type: eventType, Hidden: hidden == 1,
		CreatedAt: time.Now(),
	}, nil
}

func (db *DB) GetEvent(id, userID int, isAdmin bool) (*Event, error) {
	if !isAdmin {
		_, err := db.canAccessEvent(id, userID)
		if err != nil {
			return nil, err
		}
	}

	var ev Event
	var archivedAt sql.NullTime
	err := db.conn.QueryRow(`
		SELECT e.id, e.name, e.description, e.event_date,
			e.user_id, COALESCE(u.username, ''), e.locked,
			e.type, e.hidden, e.public,
			e.archived_at, e.created_at
		FROM events e LEFT JOIN users u ON e.user_id = u.id
		WHERE e.id = ?
	`, id).Scan(&ev.ID, &ev.Name, &ev.Description, &ev.EventDate,
		&ev.OwnerID, &ev.OwnerName, &ev.Locked,
		&ev.Type, &ev.Hidden, &ev.Public,
		&archivedAt, &ev.CreatedAt)
	if err != nil {
		return nil, err
	}
	if archivedAt.Valid {
		t := archivedAt.Time
		ev.ArchivedAt = &t
	}

	// Hidden events only accessible by admins
	if ev.Hidden && !isAdmin {
		return nil, fmt.Errorf("event not found")
	}

	// Archived events only accessible by admins
	if ev.ArchivedAt != nil && !isAdmin {
		return nil, fmt.Errorf("event not found")
	}

	// Attendees
	attRows, err := db.conn.Query(`
		SELECT ea.user_id, u.username
		FROM event_attendees ea JOIN users u ON ea.user_id = u.id
		WHERE ea.event_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer attRows.Close()
	for attRows.Next() {
		var a EventAttendee
		if err := attRows.Scan(&a.UserID, &a.Username); err != nil {
			return nil, err
		}
		ev.Attendees = append(ev.Attendees, a)
	}
	ev.AttendeeCount = len(ev.Attendees)

	// Beers (only for tasting events)
	if ev.Type != "roll" {
		beerRows, err := db.conn.Query(`
			SELECT eb.id, eb.product_id, p.name_bold, p.name_thin, p.producer_name, p.image_url
			FROM event_beers eb
			JOIN products p ON eb.product_id = p.product_id
			WHERE eb.event_id = ?
			ORDER BY eb.added_at
		`, id)
		if err != nil {
			return nil, err
		}
		defer beerRows.Close()
		for beerRows.Next() {
			var b EventBeer
			if err := beerRows.Scan(&b.ID, &b.ProductID, &b.ProductNameBold, &b.ProductNameThin, &b.ProducerName, &b.ImageURL); err != nil {
				return nil, err
			}
			ev.Beers = append(ev.Beers, b)
		}
		ev.BeerCount = len(ev.Beers)

		// Scores
		scoreRows, err := db.conn.Query(`
			SELECT es.event_beer_id, es.user_id, es.score
			FROM event_scores es
			JOIN event_beers eb ON es.event_beer_id = eb.id
			WHERE eb.event_id = ?
		`, id)
		if err != nil {
			return nil, err
		}
		defer scoreRows.Close()
		for scoreRows.Next() {
			var s EventScore
			if err := scoreRows.Scan(&s.EventBeerID, &s.UserID, &s.Score); err != nil {
				return nil, err
			}
			ev.Scores = append(ev.Scores, s)
		}
	} else {
		// For roll events, beer count comes from roll_pool
		var cnt int
		if err := db.conn.QueryRow("SELECT COUNT(*) FROM roll_pool WHERE event_id = ?", id).Scan(&cnt); err != nil {
			return nil, err
		}
		ev.BeerCount = cnt
	}

	return &ev, nil
}

func (db *DB) UpdateEvent(id int, name, description, eventDate string, userID int) error {
	res, err := db.conn.Exec(
		"UPDATE events SET name = ?, description = ?, event_date = ? WHERE id = ? AND user_id = ?",
		name, description, eventDate, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("event not found or not owned by you")
	}
	return nil
}

func (db *DB) DeleteEvent(id, userID int, isAdmin bool) error {
	var res sql.Result
	var err error
	if isAdmin {
		res, err = db.conn.Exec("DELETE FROM events WHERE id = ?", id)
	} else {
		res, err = db.conn.Exec("DELETE FROM events WHERE id = ? AND user_id = ?", id, userID)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("event not found or not owned by you")
	}
	return nil
}

func (db *DB) SetEventLocked(eventID int, locked bool, userID int, isAdmin bool) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM events WHERE id = ?", eventID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("event not found")
	}
	if ownerID != userID && !isAdmin {
		return fmt.Errorf("only the owner or admin can lock/unlock an event")
	}
	val := 0
	if locked {
		val = 1
	}
	_, err = db.conn.Exec("UPDATE events SET locked = ? WHERE id = ?", val, eventID)
	return err
}

func (db *DB) InviteToEvent(eventID, ownerID, targetUserID int) error {
	if ownerID == targetUserID {
		return fmt.Errorf("cannot invite yourself")
	}
	var actualOwner int
	err := db.conn.QueryRow("SELECT user_id FROM events WHERE id = ?", eventID).Scan(&actualOwner)
	if err != nil {
		return fmt.Errorf("event not found")
	}
	if actualOwner != ownerID {
		return fmt.Errorf("only the owner can invite users")
	}
	_, err = db.conn.Exec(`
		INSERT INTO event_attendees (event_id, user_id) VALUES (?, ?)
		ON CONFLICT DO NOTHING
	`, eventID, targetUserID)
	return err
}

func (db *DB) UninviteFromEvent(eventID, callerID, targetUserID int) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM events WHERE id = ?", eventID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("event not found")
	}
	if callerID != ownerID && callerID != targetUserID {
		return fmt.Errorf("not allowed")
	}
	_, err = db.conn.Exec("DELETE FROM event_attendees WHERE event_id = ? AND user_id = ?", eventID, targetUserID)
	return err
}

func (db *DB) ImportSharedListToEvent(eventID, listID, userID int, isAdmin bool) error {
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
		INSERT OR IGNORE INTO event_beers (event_id, product_id)
		SELECT ?, product_id FROM shared_list_items WHERE list_id = ?
	`, eventID, listID)
	return err
}

func (db *DB) AddBeerToEvent(eventID int, productID string, userID int, isAdmin bool) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM events WHERE id = ?", eventID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("event not found")
	}
	if ownerID != userID && !isAdmin {
		return fmt.Errorf("only the owner or admin can add beers")
	}
	_, err = db.conn.Exec(`
		INSERT INTO event_beers (event_id, product_id) VALUES (?, ?)
		ON CONFLICT DO NOTHING
	`, eventID, productID)
	return err
}

func (db *DB) RemoveBeerFromEvent(eventID, eventBeerID, userID int, isAdmin bool) error {
	var ownerID int
	err := db.conn.QueryRow("SELECT user_id FROM events WHERE id = ?", eventID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("event not found")
	}
	if ownerID != userID && !isAdmin {
		return fmt.Errorf("only the owner or admin can remove beers")
	}
	res, err := db.conn.Exec("DELETE FROM event_beers WHERE id = ? AND event_id = ?", eventBeerID, eventID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("beer not found in event")
	}
	return nil
}

func (db *DB) SetScore(eventBeerID, userID, score int) error {
	// Verify the event_beer exists and get event_id
	var eventID int
	err := db.conn.QueryRow("SELECT event_id FROM event_beers WHERE id = ?", eventBeerID).Scan(&eventID)
	if err != nil {
		return fmt.Errorf("beer not found")
	}

	// Check event is not locked
	var locked bool
	if err := db.conn.QueryRow("SELECT locked FROM events WHERE id = ?", eventID).Scan(&locked); err != nil {
		return fmt.Errorf("event not found")
	}
	if locked {
		return fmt.Errorf("event is locked")
	}

	// Check user has access (owner or attendee)
	_, err = db.canAccessEvent(eventID, userID)
	if err != nil {
		return err
	}

	if score < 0 || score > 10 {
		return fmt.Errorf("score must be 0-10")
	}

	_, err = db.conn.Exec(`
		INSERT INTO event_scores (event_beer_id, user_id, score, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(event_beer_id, user_id) DO UPDATE SET score = excluded.score, updated_at = CURRENT_TIMESTAMP
	`, eventBeerID, userID, score)
	return err
}

func (db *DB) DeleteScore(eventBeerID, userID int) error {
	var eventID int
	err := db.conn.QueryRow("SELECT event_id FROM event_beers WHERE id = ?", eventBeerID).Scan(&eventID)
	if err != nil {
		return fmt.Errorf("beer not found")
	}
	var locked bool
	if err := db.conn.QueryRow("SELECT locked FROM events WHERE id = ?", eventID).Scan(&locked); err != nil {
		return fmt.Errorf("event not found")
	}
	if locked {
		return fmt.Errorf("event is locked")
	}
	_, err = db.canAccessEvent(eventID, userID)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec("DELETE FROM event_scores WHERE event_beer_id = ? AND user_id = ?", eventBeerID, userID)
	return err
}

func (db *DB) ToggleEventPublic(eventID int, isAdmin bool) (bool, error) {
	if !isAdmin {
		return false, fmt.Errorf("only an admin can toggle public access")
	}
	var exists int
	err := db.conn.QueryRow("SELECT 1 FROM events WHERE id = ?", eventID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("event not found")
	}

	var current bool
	db.conn.QueryRow("SELECT public FROM events WHERE id = ?", eventID).Scan(&current)
	if current {
		// Disable public access
		_, err = db.conn.Exec("UPDATE events SET public = 0 WHERE id = ?", eventID)
		return false, err
	}

	// Check if another event is already public
	var otherID int
	var otherName string
	err = db.conn.QueryRow("SELECT id, name FROM events WHERE public = 1 AND id != ?", eventID).Scan(&otherID, &otherName)
	if err == nil {
		return false, fmt.Errorf("event '%s' is already public, disable it first", otherName)
	}

	// Enable public access
	_, err = db.conn.Exec("UPDATE events SET public = 1 WHERE id = ?", eventID)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) IsEventPublic(eventID int) (bool, error) {
	var public bool
	err := db.conn.QueryRow("SELECT public FROM events WHERE id = ?", eventID).Scan(&public)
	if err != nil {
		return false, err
	}
	return public, nil
}

func (db *DB) GetPublicEvent() (*Event, error) {
	var ev Event
	err := db.conn.QueryRow(`
		SELECT e.id, e.name, e.description, e.event_date,
			e.user_id, COALESCE(u.username, ''), e.locked,
			e.type, e.hidden, e.public, e.created_at
		FROM events e LEFT JOIN users u ON e.user_id = u.id
		WHERE e.public = 1 AND e.type = 'roll' AND e.archived_at IS NULL
	`).Scan(&ev.ID, &ev.Name, &ev.Description, &ev.EventDate,
		&ev.OwnerID, &ev.OwnerName, &ev.Locked,
		&ev.Type, &ev.Hidden, &ev.Public, &ev.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("no active public event")
	}

	// Load attendees
	attRows, err := db.conn.Query(`
		SELECT ea.user_id, u.username
		FROM event_attendees ea JOIN users u ON ea.user_id = u.id
		WHERE ea.event_id = ?
	`, ev.ID)
	if err != nil {
		return nil, err
	}
	defer attRows.Close()
	for attRows.Next() {
		var a EventAttendee
		if err := attRows.Scan(&a.UserID, &a.Username); err != nil {
			return nil, err
		}
		ev.Attendees = append(ev.Attendees, a)
	}
	ev.AttendeeCount = len(ev.Attendees)

	return &ev, nil
}

func (db *DB) ArchiveEvent(eventID, userID int, isAdmin bool) error {
	var res sql.Result
	var err error
	if isAdmin {
		res, err = db.conn.Exec(
			"UPDATE events SET archived_at = CURRENT_TIMESTAMP, public = 0 WHERE id = ? AND archived_at IS NULL",
			eventID)
	} else {
		res, err = db.conn.Exec(
			"UPDATE events SET archived_at = CURRENT_TIMESTAMP, public = 0 WHERE id = ? AND user_id = ? AND archived_at IS NULL",
			eventID, userID)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("event not found, already archived, or not owned by you")
	}
	return nil
}

func (db *DB) UnarchiveEvent(eventID int) error {
	res, err := db.conn.Exec(
		"UPDATE events SET archived_at = NULL WHERE id = ? AND archived_at IS NOT NULL",
		eventID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("event not found or not archived")
	}
	return nil
}

func (db *DB) ListArchivedEvents() ([]Event, error) {
	rows, err := db.conn.Query(`
		SELECT e.id, e.name, e.description, e.event_date,
			e.user_id, COALESCE(u.username, ''), e.locked,
			e.type, e.hidden, e.public,
			e.archived_at,
			e.created_at,
			(SELECT COUNT(*) FROM event_attendees WHERE event_id = e.id) AS attendee_count,
			CASE WHEN e.type = 'roll'
				THEN (SELECT COUNT(*) FROM roll_pool WHERE event_id = e.id)
				ELSE (SELECT COUNT(*) FROM event_beers WHERE event_id = e.id)
			END AS beer_count
		FROM events e
		LEFT JOIN users u ON e.user_id = u.id
		WHERE e.archived_at IS NOT NULL
		ORDER BY e.archived_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var ev Event
		var archivedAt sql.NullTime
		if err := rows.Scan(&ev.ID, &ev.Name, &ev.Description, &ev.EventDate,
			&ev.OwnerID, &ev.OwnerName, &ev.Locked,
			&ev.Type, &ev.Hidden, &ev.Public,
			&archivedAt,
			&ev.CreatedAt,
			&ev.AttendeeCount, &ev.BeerCount); err != nil {
			return nil, err
		}
		if archivedAt.Valid {
			t := archivedAt.Time
			ev.ArchivedAt = &t
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}
