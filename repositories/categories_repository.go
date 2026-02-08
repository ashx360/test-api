package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"test-api/models"
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (repo *CategoryRepository) GetAll() ([]models.Category, error) {
	query := "SELECT id, nama, description FROM categories"
	rows, err := repo.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]models.Category, 0)
	for rows.Next() {
		var c models.Category
		err := rows.Scan(&c.ID, &c.Nama, &c.Description)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

func (repo *CategoryRepository) Create(category *models.Category) error {
	query := "INSERT INTO categories (nama, description) VALUES ($1, $2) RETURNING id"
	err := repo.db.QueryRow(query, category.Nama, category.Description).Scan(&category.ID)
	return err
}

// GetByID - ambil kategori by ID
func (repo *CategoryRepository) GetByID(id int) (*models.Category, error) {
	query := "SELECT id, nama, description FROM categories WHERE id = $1"

	var c models.Category
	err := repo.db.QueryRow(query, id).Scan(&c.ID, &c.Nama, &c.Description)
	if err == sql.ErrNoRows {
		return nil, errors.New("kategori tidak ditemukan")
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (repo *CategoryRepository) Update(category *models.Category) error {
	query := "UPDATE categories SET nama = $1, description = $2 WHERE id = $3"
	result, err := repo.db.Exec(query, category.Nama, category.Description, category.ID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("kategori tidak ditemukan")
	}

	return nil
}

func (repo *CategoryRepository) Delete(id int) error {
	query := "DELETE FROM categories WHERE id = $1"
	result, err := repo.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("kategori tidak ditemukan")
	}

	return nil
}

//////////////////////// Transaction Repository //////////////////////

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (repo *TransactionRepository) CreateTransaction(items []models.CheckoutItem) (*models.Transaction, error) {
	tx, err := repo.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	totalAmount := 0
	details := make([]models.TransactionDetail, 0)

	for _, item := range items {
		var productPrice, stock int
		var productName string

		err := tx.QueryRow("SELECT name, price, stock FROM products WHERE id = $1", item.ProductID).Scan(&productName, &productPrice, &stock)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product id %d not found", item.ProductID)
		}
		if err != nil {
			return nil, err
		}

		subtotal := productPrice * item.Quantity
		totalAmount += subtotal

		_, err = tx.Exec("UPDATE products SET stock = stock - $1 WHERE id = $2", item.Quantity, item.ProductID)
		if err != nil {
			return nil, err
		}

		details = append(details, models.TransactionDetail{
			ProductID:   item.ProductID,
			ProductName: productName,
			Quantity:    item.Quantity,
			Subtotal:    subtotal,
		})
	}

	var transactionID int
	err = tx.QueryRow("INSERT INTO transactions (total_amount) VALUES ($1) RETURNING id", totalAmount).Scan(&transactionID)
	if err != nil {
		return nil, err
	}

	for i := range details {
		details[i].TransactionID = transactionID

		// Capture the auto-generated ID if your table has one
		var detailID int
		if err := tx.QueryRow(
			"INSERT INTO transaction_details (transaction_id, product_id, quantity, subtotal) VALUES ($1, $2, $3, $4) RETURNING id",
			transactionID, details[i].ProductID, details[i].Quantity, details[i].Subtotal,
		).Scan(&detailID); err != nil {
			return nil, err
		}

		details[i].ID = detailID
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.Transaction{
		ID:          transactionID,
		TotalAmount: totalAmount,
		Details:     details,
	}, nil
}

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) FetchReportToday() (map[string]interface{}, error) {
	query := `
		SELECT 
			COALESCE(SUM(total_amount),0) AS total_revenue,
			COUNT(*) AS total_transaksi
		FROM transactions
		WHERE DATE(created_at) = CURRENT_DATE
	`

	var totalRevenue, totalTransaksi int
	if err := r.db.QueryRow(query).Scan(&totalRevenue, &totalTransaksi); err != nil {
		return nil, err
	}

	// Get best-selling product today
	var productName string
	var qtyTerjual int
	err := r.db.QueryRow(`
		SELECT p.name, SUM(td.quantity) AS qty_terjual
		FROM transaction_details td
		JOIN products p ON td.product_id = p.id
		JOIN transactions t ON td.transaction_id = t.id
		WHERE DATE(t.created_at) = CURRENT_DATE
		GROUP BY p.name
		ORDER BY qty_terjual DESC
		LIMIT 1
	`).Scan(&productName, &qtyTerjual)

	if err == sql.ErrNoRows {
		productName = ""
		qtyTerjual = 0
	} else if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_revenue":   totalRevenue,
		"total_transaksi": totalTransaksi,
		"produk_terlaris": map[string]interface{}{
			"nama":        productName,
			"qty_terjual": qtyTerjual,
		},
	}, nil
}

func (r *ReportRepository) FetchReportByRange(startDate, endDate string) (map[string]interface{}, error) {
	query := `
		SELECT 
			COALESCE(SUM(total_amount),0) AS total_revenue,
			COUNT(*) AS total_transaksi
		FROM transactions
		WHERE DATE(created_at) BETWEEN $1 AND $2
	`

	var totalRevenue, totalTransaksi int
	if err := r.db.QueryRow(query, startDate, endDate).Scan(&totalRevenue, &totalTransaksi); err != nil {
		return nil, err
	}

	var productName string
	var qtyTerjual int
	err := r.db.QueryRow(`
		SELECT p.name, SUM(td.quantity) AS qty_terjual
		FROM transaction_details td
		JOIN products p ON td.product_id = p.id
		JOIN transactions t ON td.transaction_id = t.id
		WHERE DATE(t.created_at) BETWEEN $1 AND $2
		GROUP BY p.name
		ORDER BY qty_terjual DESC
		LIMIT 1
	`, startDate, endDate).Scan(&productName, &qtyTerjual)

	if err == sql.ErrNoRows {
		productName = ""
		qtyTerjual = 0
	} else if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_revenue":   totalRevenue,
		"total_transaksi": totalTransaksi,
		"produk_terlaris": map[string]interface{}{
			"nama":        productName,
			"qty_terjual": qtyTerjual,
		},
	}, nil
}
