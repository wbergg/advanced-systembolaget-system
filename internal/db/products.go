package db

import (
	"fmt"
	"strings"

	"advanced-systembolaget-system/internal/systembolaget"
)

type ProductWithNote struct {
	systembolaget.Product
	Note *string `json:"note"`
}

type ListFilter struct {
	Search    string
	Category  string
	MinPrice  *float64
	MaxPrice  *float64
	MinAbv    *float64
	MaxAbv    *float64
	SortBy    string
	SortDir   string
	Page      int
	PageSize  int
	Name      string
	Producer  string
	Countries []string
	Packagings []string
	Volumes   []string
}

func (db *DB) UpsertProducts(products []systembolaget.Product) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO products (
			product_id, product_number, name_bold, name_thin,
			producer_name, price, volume, volume_text, alcohol_pct,
			country, category_level1, category_level2, assortment_text,
			taste, usage, is_organic, is_news, packaging_level1,
			assortment, product_launch_date, vintage, image_url, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range products {
		_, err := stmt.Exec(
			p.ProductID, p.ProductNumber, p.ProductNameBold, p.ProductNameThin,
			p.ProducerName, p.Price, p.Volume, p.VolumeText, p.AlcoholPercent,
			p.Country, p.CategoryLevel1, p.CategoryLevel2, p.AssortmentText,
			p.Taste, p.Usage, p.IsOrganic, p.IsNews, p.PackagingLevel1,
			p.Assortment, p.ProductLaunchDate, p.Vintage, p.ImageURL,
		)
		if err != nil {
			return fmt.Errorf("upsert product %s: %w", p.ProductID, err)
		}
	}

	return tx.Commit()
}

func (db *DB) ListProducts(f ListFilter) ([]ProductWithNote, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 50
	}

	var where []string
	var args []any

	if f.Search != "" {
		where = append(where, "(p.name_bold LIKE ? OR p.producer_name LIKE ? OR p.taste LIKE ?)")
		s := "%" + f.Search + "%"
		args = append(args, s, s, s)
	}
	if f.Category != "" {
		where = append(where, "p.category_level1 = ?")
		args = append(args, f.Category)
	}
	if f.MinPrice != nil {
		where = append(where, "p.price >= ?")
		args = append(args, *f.MinPrice)
	}
	if f.MaxPrice != nil {
		where = append(where, "p.price <= ?")
		args = append(args, *f.MaxPrice)
	}
	if f.Name != "" {
		where = append(where, "p.name_bold LIKE ?")
		args = append(args, "%"+f.Name+"%")
	}
	if f.Producer != "" {
		where = append(where, "p.producer_name LIKE ?")
		args = append(args, "%"+f.Producer+"%")
	}
	if f.MinAbv != nil {
		where = append(where, "p.alcohol_pct >= ?")
		args = append(args, *f.MinAbv)
	}
	if f.MaxAbv != nil {
		where = append(where, "p.alcohol_pct <= ?")
		args = append(args, *f.MaxAbv)
	}
	if len(f.Countries) > 0 {
		placeholders := make([]string, len(f.Countries))
		for i, v := range f.Countries {
			placeholders[i] = "?"
			args = append(args, v)
		}
		where = append(where, "p.country IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(f.Packagings) > 0 {
		placeholders := make([]string, len(f.Packagings))
		for i, v := range f.Packagings {
			placeholders[i] = "?"
			args = append(args, v)
		}
		where = append(where, "p.packaging_level1 IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(f.Volumes) > 0 {
		placeholders := make([]string, len(f.Volumes))
		for i, v := range f.Volumes {
			placeholders[i] = "?"
			args = append(args, v)
		}
		where = append(where, "p.volume_text IN ("+strings.Join(placeholders, ",")+")")
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) FROM products p " + whereClause
	if err := db.conn.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Sort
	sortCol := "p.name_bold"
	switch f.SortBy {
	case "price":
		sortCol = "p.price"
	case "name":
		sortCol = "p.name_bold"
	case "alcohol":
		sortCol = "p.alcohol_pct"
	case "volume":
		sortCol = "p.volume"
	case "country":
		sortCol = "p.country"
	case "category":
		sortCol = "p.category_level1"
	case "producer":
		sortCol = "p.producer_name"
	}

	sortDir := "ASC"
	if strings.EqualFold(f.SortDir, "desc") {
		sortDir = "DESC"
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf(`
		SELECT p.product_id, p.product_number, p.name_bold, p.name_thin,
			p.producer_name, p.price, p.volume, p.volume_text, p.alcohol_pct,
			p.country, p.category_level1, p.category_level2, p.assortment_text,
			p.taste, p.usage, p.is_organic, p.is_news, p.packaging_level1,
			p.assortment, p.vintage, p.image_url, n.note
		FROM products p
		LEFT JOIN notes n ON p.product_id = n.product_id
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, sortCol, sortDir)

	queryArgs := append(args, f.PageSize, offset)
	rows, err := db.conn.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []ProductWithNote
	for rows.Next() {
		var p ProductWithNote
		err := rows.Scan(
			&p.ProductID, &p.ProductNumber, &p.ProductNameBold, &p.ProductNameThin,
			&p.ProducerName, &p.Price, &p.Volume, &p.VolumeText, &p.AlcoholPercent,
			&p.Country, &p.CategoryLevel1, &p.CategoryLevel2, &p.AssortmentText,
			&p.Taste, &p.Usage, &p.IsOrganic, &p.IsNews, &p.PackagingLevel1,
			&p.Assortment, &p.Vintage, &p.ImageURL, &p.Note,
		)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}

	return products, total, rows.Err()
}

func (db *DB) GetProduct(id string) (*ProductWithNote, error) {
	var p ProductWithNote
	err := db.conn.QueryRow(`
		SELECT p.product_id, p.product_number, p.name_bold, p.name_thin,
			p.producer_name, p.price, p.volume, p.volume_text, p.alcohol_pct,
			p.country, p.category_level1, p.category_level2, p.assortment_text,
			p.taste, p.usage, p.is_organic, p.is_news, p.packaging_level1,
			p.assortment, p.vintage, p.image_url, n.note
		FROM products p
		LEFT JOIN notes n ON p.product_id = n.product_id
		WHERE p.product_id = ?
	`, id).Scan(
		&p.ProductID, &p.ProductNumber, &p.ProductNameBold, &p.ProductNameThin,
		&p.ProducerName, &p.Price, &p.Volume, &p.VolumeText, &p.AlcoholPercent,
		&p.Country, &p.CategoryLevel1, &p.CategoryLevel2, &p.AssortmentText,
		&p.Taste, &p.Usage, &p.IsOrganic, &p.IsNews, &p.PackagingLevel1,
		&p.Assortment, &p.Vintage, &p.ImageURL, &p.Note,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *DB) DistinctValues(column string) ([]string, error) {
	allowed := map[string]string{
		"country":   "country",
		"packaging": "packaging_level1",
		"volume":    "volume_text",
	}
	col, ok := allowed[column]
	if !ok {
		return nil, fmt.Errorf("invalid column: %s", column)
	}
	rows, err := db.conn.Query(
		fmt.Sprintf("SELECT DISTINCT %s FROM products WHERE %s != '' ORDER BY %s", col, col, col),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vals []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}

func (db *DB) DeleteProduct(id string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range []string{
		"notes", "comments", "basket_items", "shared_list_items",
		"event_beers", "roll_pool",
	} {
		if _, err := tx.Exec("DELETE FROM "+table+" WHERE product_id = ?", id); err != nil {
			return fmt.Errorf("clearing %s: %w", table, err)
		}
	}

	res, err := tx.Exec("DELETE FROM products WHERE product_id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("product not found")
	}
	return tx.Commit()
}

func (db *DB) DeleteAllProducts() (int64, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Delete from all tables that reference products
	for _, table := range []string{
		"notes", "comments", "basket_items", "shared_list_items",
		"event_beers", "roll_pool",
	} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return 0, fmt.Errorf("clearing %s: %w", table, err)
		}
	}

	res, err := tx.Exec("DELETE FROM products")
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}
