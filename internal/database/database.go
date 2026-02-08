package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "middelmatigheid/internal/models"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

type Database struct {
	*sql.DB
	logger *slog.Logger
}

// Connect establishes connection with PostgreSQL database
func Connect(logger *slog.Logger) (*Database, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	conn := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbName)
	database, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}
	if err = database.Ping(); err != nil {
		return nil, err
	}
	logger.Info("Connection with the database is established", slog.String("function", "Connect"))
	return &Database{database, logger}, nil
}

// Migrate migrates the database
func (db *Database) Migrate() error {
	dbName := os.Getenv("DB_NAME")

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	migrationDirs := []string{
		filepath.Join(wd, "migrations"),
		filepath.Join(wd, "..", "migrations"),
		filepath.Join(wd, "..", "..", "migrations"),
	}
	var migrationPath string
	for _, dir := range migrationDirs {
		if _, err := os.Stat(dir); err == nil {
			migrationPath = dir
			break
		}
	}
	if migrationPath == "" {
		return fmt.Errorf("migrations directory not found")
	}
	migrationPath = strings.ReplaceAll(migrationPath, "\\", "/")

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationPath),
		dbName,
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	version, dirty, err := m.Version()
	if err == nil && dirty {
		db.logger.Warn("Dirty database detected, forcing version", slog.Int("version", int(version)))
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("failed to force version: %w", err)
		}
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

func (db *Database) MigrateDown() error {
	dbName := os.Getenv("DB_NAME")

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	migrationDirs := []string{
		filepath.Join(wd, "migrations"),
		filepath.Join(wd, "..", "migrations"),
		filepath.Join(wd, "..", "..", "migrations"),
	}
	var migrationPath string
	for _, dir := range migrationDirs {
		if _, err := os.Stat(dir); err == nil {
			migrationPath = dir
			break
		}
	}
	if migrationPath == "" {
		return fmt.Errorf("migrations directory not found")
	}
	migrationPath = strings.ReplaceAll(migrationPath, "\\", "/")

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationPath),
		dbName,
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	version, dirty, err := m.Version()
	if err == nil && dirty {
		db.logger.Warn("Dirty database detected, forcing version", slog.Int("version", int(version)))
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("failed to force version: %w", err)
		}
	}

	err = m.Down()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

// Create insert new subscription into the database, returns its id if the insertion was successful
func (db *Database) Create(subscription Subscription) (IdResponse, *Error) {
	// Checking if the subscription is already stored in the database
	var id int
	req := `SELECT id FROM subscriptions WHERE user_id = $1 AND service_name = $2;`
	err := db.QueryRow(req, subscription.UserId, subscription.ServiceName).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return IdResponse{}, &Error{TypeOf: ErrInternalServer, Err: err}
	} else if err == nil {
		return IdResponse{Id: id}, &Error{TypeOf: ErrConflict, Err: nil}
	}

	// Parsing date
	startDate, err := time.Parse("01-2006", subscription.StartDate)
	if err != nil {
		return IdResponse{}, &Error{TypeOf: ErrTimeParse, Err: err}
	}
	var endDate *time.Time
	if len(subscription.EndDate) > 0 {
		end, err := time.Parse("01-2006", subscription.EndDate)
		if err != nil || end.Before(startDate) {
			return IdResponse{}, &Error{TypeOf: ErrTimeParse, Err: err}
		}
		endDate = &end
	} else {
		endDate = nil
	}

	// Inserting subscription into the database
	query := `INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;`
	db.QueryRow(query, subscription.ServiceName, subscription.Price, subscription.UserId, startDate, endDate, time.Now(), time.Now()).Scan(&subscription.Id)
	return IdResponse{Id: subscription.Id}, nil
}

// Read returns the subscription's info stored in the database. The searching can be done base on subscription's id or combination of user_id and service_name. Provide an empty string if an argument is being not used
func (db *Database) Read(id, userId, serviceName string) (Subscription, *Error) {
	// Getting subscription info
	var subscription Subscription
	var end sql.NullString
	var err error
	if len(id) > 0 && len(userId) > 0 && len(serviceName) > 0 {
		req := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at FROM subscriptions WHERE id = $1 AND user_id = $2 AND service_name = $3;`
		err = db.QueryRow(req, id, userId, serviceName).Scan(&subscription.Id, &subscription.ServiceName, &subscription.Price, &subscription.UserId, &subscription.StartDate, &end, &subscription.CreatedAt, &subscription.UpdatedAt)
	} else if len(id) > 0 && len(userId) > 0 {
		req := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at FROM subscriptions WHERE id = $1 AND user_id = $2;`
		err = db.QueryRow(req, id, userId).Scan(&subscription.Id, &subscription.ServiceName, &subscription.Price, &subscription.UserId, &subscription.StartDate, &end, &subscription.CreatedAt, &subscription.UpdatedAt)
	} else if len(id) > 0 && len(serviceName) > 0 {
		req := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at FROM subscriptions WHERE id = $1 AND service_name = $2;`
		err = db.QueryRow(req, id, serviceName).Scan(&subscription.Id, &subscription.ServiceName, &subscription.Price, &subscription.UserId, &subscription.StartDate, &end, &subscription.CreatedAt, &subscription.UpdatedAt)
	} else if len(id) > 0 {
		req := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at FROM subscriptions WHERE id = $1;`
		err = db.QueryRow(req, id).Scan(&subscription.Id, &subscription.ServiceName, &subscription.Price, &subscription.UserId, &subscription.StartDate, &end, &subscription.CreatedAt, &subscription.UpdatedAt)
	} else {
		req := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at FROM subscriptions WHERE user_id = $1 AND service_name = $2;`
		err = db.QueryRow(req, userId, serviceName).Scan(&subscription.Id, &subscription.ServiceName, &subscription.Price, &subscription.UserId, &subscription.StartDate, &end, &subscription.CreatedAt, &subscription.UpdatedAt)
	}
	if err == sql.ErrNoRows {
		return Subscription{}, &Error{TypeOf: ErrNotFound, Err: err}
	} else if err != nil {
		return Subscription{}, &Error{TypeOf: ErrInternalServer, Err: err}
	}

	// Parsing date
	startDate, err := time.Parse("2006-01-02T15:04:05Z", subscription.StartDate)
	if err != nil {
		return Subscription{}, &Error{TypeOf: ErrTimeParse, Err: err}
	}
	subscription.StartDate = startDate.Format("01-2006")
	if end.Valid {
		endDate, err := time.Parse("2006-01-02T15:04:05Z", end.String)
		if err != nil {
			return Subscription{}, &Error{TypeOf: ErrTimeParse, Err: err}
		}
		subscription.EndDate = endDate.Format("01-2006")
	} else {
		subscription.EndDate = ""
	}
	createdAt, err := time.Parse("2006-01-02T15:04:05Z", subscription.CreatedAt)
	if err != nil {
		return Subscription{}, &Error{TypeOf: ErrTimeParse, Err: err}
	}
	subscription.CreatedAt = createdAt.Format("02-01-2006 15:04")
	updatedAt, err := time.Parse("2006-01-02T15:04:05Z", subscription.UpdatedAt)
	if err != nil {
		return Subscription{}, &Error{TypeOf: ErrTimeParse, Err: err}
	}
	subscription.UpdatedAt = updatedAt.Format("02-01-2006 15:04")
	return subscription, nil
}

// Update updates subscription's info in the database. The subscription is being specified with Subscription.Id field
func (db *Database) Update(subscription Subscription) *Error {
	// Checking if the subscription exists in the database
	_, customErr := db.Read(fmt.Sprintf("%d", subscription.Id), subscription.UserId, subscription.ServiceName)
	if customErr != nil && customErr.TypeOf == ErrNotFound {
		return &Error{TypeOf: ErrNotFound, Err: nil}
	} else if customErr != nil && customErr.TypeOf != ErrNotFound {
		return customErr
	}

	// Parsing date
	startDate, err := time.Parse("01-2006", subscription.StartDate)
	if err != nil {
		return &Error{TypeOf: ErrTimeParse, Err: err}
	}
	var endDate *time.Time
	if len(subscription.EndDate) > 0 {
		end, err := time.Parse("01-2006", subscription.EndDate)
		if err != nil || end.Before(startDate) {
			return &Error{TypeOf: ErrTimeParse, Err: err}
		}
		endDate = &end
	} else {
		endDate = nil
	}

	// Getting database response
	query := `UPDATE subscriptions SET service_name = $2, price = $3, user_id = $4, start_date = $5, end_date = $6, updated_at = $7 WHERE id = $1;`
	_, err = db.Exec(query, subscription.Id, subscription.ServiceName, subscription.Price, subscription.UserId, startDate, endDate, time.Now())
	if err != nil {
		return &Error{TypeOf: ErrInternalServer, Err: err}
	}

	return nil
}

// Delete deletes a subscription from the database. The subscriptions can be specified by its id or combination of user_id and service_name. Provide an empty string if an argument is being not used
func (db *Database) Delete(id, userId, serviceName string) *Error {
	// Checking if the subscription exists in the database
	_, customErr := db.Read(id, userId, serviceName)
	if customErr != nil && customErr.TypeOf == ErrNotFound {
		return &Error{TypeOf: ErrNotFound, Err: nil}
	} else if customErr != nil {
		return customErr
	}

	fmt.Println(customErr)
	// Deleting from the database
	var err error
	if len(id) > 0 {
		req := `DELETE FROM subscriptions WHERE id = $1;`
		_, err = db.Exec(req, id)
	} else {
		req := `DELETE FROM subscriptions WHERE user_id = $1 AND service_name = $2;`
		_, err = db.Exec(req, userId, serviceName)
	}
	if err != nil {
		return &Error{TypeOf: ErrInternalServer, Err: err}
	}

	return nil
}

// List returns an array of subscriptions filtered by user_id and service_name. Provide an empty string if an argument is being not used
func (db *Database) List(userId, serviceName, start, end string) ([]Subscription, *Error) {
	// Parsing date
	var err error
	var startDate, endDate time.Time
	if len(start) > 0 {
		startDate, err = time.Parse("01-2006", start)
		if err != nil {
			return []Subscription{}, &Error{TypeOf: ErrTimeParse, Err: err}
		}
	}
	if len(end) > 0 {
		endDate, err = time.Parse("01-2006", end)
		if err != nil {
			return []Subscription{}, &Error{TypeOf: ErrTimeParse, Err: err}
		}
	}

	// Getting subscritions from the database
	var rows *sql.Rows
	var conditions []string
	var args []interface{}
	if len(userId) > 0 {
		conditions = append(conditions, fmt.Sprintf(`user_id = $%d`, len(conditions)+1))
		args = append(args, userId)
	}
	if len(serviceName) > 0 {
		conditions = append(conditions, fmt.Sprintf(`service_name = $%d`, len(conditions)+1))
		args = append(args, serviceName)
	}
	if len(start) > 0 {
		conditions = append(conditions, fmt.Sprintf(`(end_date IS NULL OR end_date >= $%d)`, len(conditions)+1))
		args = append(args, startDate)
	}
	if len(end) > 0 {
		conditions = append(conditions, fmt.Sprintf(`start_date <= $%d`, len(conditions)+1))
		args = append(args, endDate)
	}
	req := `SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at FROM subscriptions`
	if len(conditions) > 0 {
		req += fmt.Sprintf(` WHERE %s`, strings.Join(conditions, " AND "))
	}
	req += `;`
	rows, err = db.Query(req, args...)
	if err != nil && err != sql.ErrNoRows {
		return []Subscription{}, &Error{TypeOf: ErrInternalServer, Err: err}
	}
	defer rows.Close()

	// Parsing info to Subscriotion type
	var subscriptions []Subscription
	for rows.Next() {
		var subscription Subscription
		var end sql.NullString
		err = rows.Scan(&subscription.Id, &subscription.ServiceName, &subscription.Price, &subscription.UserId, &subscription.StartDate, &end, &subscription.CreatedAt, &subscription.UpdatedAt)
		if err != nil {
			return []Subscription{}, &Error{TypeOf: ErrInternalServer, Err: err}
		}
		start_date, err := time.Parse("2006-01-02T15:04:05Z", subscription.StartDate)
		if err != nil {
			return []Subscription{}, &Error{TypeOf: ErrInternalServer, Err: err}
		}
		subscription.StartDate = start_date.Format("01-2006")
		if end.Valid {
			end_date, err := time.Parse("2006-01-02T15:04:05Z", end.String)
			if err != nil {
				return []Subscription{}, &Error{TypeOf: ErrInternalServer, Err: err}
			}
			subscription.EndDate = end_date.Format("01-2006")
		} else {
			subscription.EndDate = ""
		}
		createdAt, err := time.Parse("2006-01-02T15:04:05Z", subscription.CreatedAt)
		if err != nil {
			return []Subscription{}, &Error{TypeOf: ErrInternalServer, Err: err}
		}
		subscription.CreatedAt = createdAt.Format("02-01-2006 15:04")
		updatedAt, err := time.Parse("2006-01-02T15:04:05Z", subscription.UpdatedAt)
		if err != nil {
			return []Subscription{}, &Error{TypeOf: ErrInternalServer, Err: err}
		}
		subscription.UpdatedAt = updatedAt.Format("02-01-2006 15:04")
		subscriptions = append(subscriptions, subscription)
	}

	if len(subscriptions) == 0 {
		return []Subscription{}, &Error{TypeOf: ErrNotFound, Err: nil}
	}
	return subscriptions, nil
}

// Summary return amount of subscriptions within the provided period and total amount that was payed. Subscriptions can be filtered by the period, user_id and service_name. Provide an empty string if an argument is being not used
func (db *Database) Summary(startStr, endStr, userId, serviceName string) (SummaryResponse, *Error) {
	// Parsing date
	var err error
	startDate, err := time.Parse("01-2006", startStr)
	if err != nil {
		return SummaryResponse{}, &Error{TypeOf: ErrTimeParse, Err: err}
	}
	endDate, err := time.Parse("01-2006", endStr)
	if err != nil {
		return SummaryResponse{}, &Error{TypeOf: ErrTimeParse, Err: err}
	}
	if endDate.Before(startDate) {
		return SummaryResponse{}, &Error{TypeOf: ErrTimeParse, Err: nil}
	}

	// Getting subscritions from the database
	var rows *sql.Rows
	var conditions []string
	var args []any
	if len(userId) > 0 {
		conditions = append(conditions, fmt.Sprintf(`user_id = $%d`, len(conditions)+1))
		args = append(args, userId)
	}
	if len(serviceName) > 0 {
		conditions = append(conditions, fmt.Sprintf(`service_name = $%d`, len(conditions)+1))
		args = append(args, serviceName)
	}
	conditions = append(conditions, fmt.Sprintf(`start_date <= $%d AND (end_date IS NULL OR end_date >= $%d)`, len(conditions)+1, len(conditions)+2))
	args = append(args, endDate, startDate)
	req := `SELECT id, service_name, price, user_id, start_date, end_date FROM subscriptions`
	if len(conditions) > 0 {
		req += fmt.Sprintf(` WHERE %s`, strings.Join(conditions, " AND "))
	}
	req += `;`
	fmt.Println(req)
	rows, err = db.Query(req, args...)
	if err != nil && err != sql.ErrNoRows {
		return SummaryResponse{}, &Error{TypeOf: ErrInternalServer, Err: err}
	}
	defer rows.Close()

	// Calculating result
	var amount, total int
	for rows.Next() {
		var subscription Subscription
		var endStr sql.NullString
		err = rows.Scan(&subscription.Id, &subscription.ServiceName, &subscription.Price, &subscription.UserId, &subscription.StartDate, &endStr)
		if err != nil {
			return SummaryResponse{}, &Error{TypeOf: ErrInternalServer, Err: err}
		}
		// Parsing date
		start, err := time.Parse("2006-01-02T15:04:05Z", subscription.StartDate)
		if err != nil {
			return SummaryResponse{}, &Error{TypeOf: ErrInternalServer, Err: err}
		}
		if start.Before(startDate) {
			start = startDate
		}
		var end time.Time
		if endStr.Valid {
			end, err = time.Parse("2006-01-02T15:04:05Z", endStr.String)
			if err != nil {
				return SummaryResponse{}, &Error{TypeOf: ErrInternalServer, Err: err}
			}
			if end.After(endDate) {
				end = endDate
			}
		} else {
			end = endDate
		}

		// Calculating total price
		startYear, startMonth, _ := start.Date()
		endYear, endMonth, _ := end.Date()

		years := endYear - startYear
		months := int(endMonth) - int(startMonth) + 1

		totalMonths := years*12 + months
		amount++
		total += subscription.Price * totalMonths
	}
	_, startMonth, _ := startDate.Date()
	_, endMonth, _ := endDate.Date()
	months := int(endMonth) - int(startMonth) + 1
	return SummaryResponse{Amount: amount, Months: months, Total: total}, nil
}
