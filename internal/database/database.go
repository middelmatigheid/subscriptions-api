package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/middelmatigheid/subscriptions-api/internal/config"
	"github.com/middelmatigheid/subscriptions-api/internal/models"
)

type Database struct {
	*sql.DB
	logger *slog.Logger
}

// Connect establishes connection with PostgreSQL database
func Connect(config *config.Config, logger *slog.Logger) (*Database, error) {
	// Connecting to the database
	conn := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", config.DBUser, config.DBPassword, config.DBHost, config.DBName)
	database, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, models.NewErrInternalServer(err)
	}
	if err = database.Ping(); err != nil {
		return nil, models.NewErrInternalServer(err)
	}
	logger.Info("Connection with the database is established", slog.String("function", "Connect"))
	return &Database{database, logger}, nil
}

// Close terminates the connection with the database
func (db *Database) Close() error {
	err := db.DB.Close()
	if err != nil {
		return models.NewErrInternalServer(err)
	}
	return nil
}

// Create inserts new subscription into the database and returns its id, if the insertion was successful, or returs id of conflicting subscription
func (db *Database) Create(ctx context.Context, subscription models.Subscription) (models.IDResponse, error) {
	// Checking if the subscription is being already stored in the database
	sub, err := db.Read(ctx, models.SubscriptionIdentifier{UserUUID: subscription.UserUUID, ServiceName: subscription.ServiceName})
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		return models.IDResponse{}, models.NewErrInternalServer(err)
	} else if !errors.Is(err, models.ErrNotFound) {
		return models.IDResponse{ID: sub.ID}, models.NewErrConflict()
	}

	// Inserting subscription into the database
	query := `INSERT INTO subscriptions (service_name, price, user_uuid, start_date, end_date, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;`
	err = db.QueryRowContext(ctx, query, subscription.ServiceName, subscription.Price, subscription.UserUUID, subscription.StartDate, subscription.EndDate,
		time.Now(), time.Now()).Scan(&subscription.ID)
	if err != nil {
		return models.IDResponse{}, models.NewErrInternalServer(err)
	}
	return models.IDResponse{ID: subscription.ID}, nil
}

// Read returns the subscription's info stored in the database. The subscription is being specified by its id or combination of user uuid and service name
func (db *Database) Read(ctx context.Context, identifier models.SubscriptionIdentifier) (models.Subscription, error) {
	// Getting subscription info
	var subscription models.Subscription
	query := `SELECT id, service_name, price, user_uuid, start_date, end_date, created_at, updated_at FROM subscriptions WHERE ($1 <= 0 OR id = $1) AND 
		($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR user_uuid = $2) AND ($3::text = ''::text OR service_name = $3);`
	err := db.QueryRowContext(ctx, query, identifier.ID, identifier.UserUUID, identifier.ServiceName).Scan(&subscription.ID, &subscription.ServiceName, &subscription.Price,
		&subscription.UserUUID, &subscription.StartDate, &subscription.EndDate, &subscription.CreatedAt, &subscription.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Subscription{}, models.NewErrNotFound()
	} else if err != nil {
		return models.Subscription{}, models.NewErrInternalServer(err)
	}

	return subscription, nil
}

// Update updates subscription's info in the database. The subscription is being specified by its id
func (db *Database) Update(ctx context.Context, subscription models.Subscription) error {
	// Checking if the same subscription exists in the database
	exists, err := db.Read(ctx, models.SubscriptionIdentifier{UserUUID: subscription.UserUUID, ServiceName: subscription.ServiceName})
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		return err
	} else if err == nil && subscription.ID != exists.ID {
		return models.NewErrConflict()
	}

	// Getting database response
	query := `UPDATE subscriptions SET service_name = $2, price = $3, user_uuid = $4, start_date = $5, end_date = $6, updated_at = $7 WHERE id = $1;`
	_, err = db.ExecContext(ctx, query, subscription.ID, subscription.ServiceName, subscription.Price, subscription.UserUUID, subscription.StartDate, subscription.EndDate, time.Now())
	if err != nil {
		return models.NewErrInternalServer(err)
	}

	return nil
}

// Delete deletes a subscription from the database. The subscriptions can be specified by its id or combination of user uuid and service name
func (db *Database) Delete(ctx context.Context, identifier models.SubscriptionIdentifier) error {
	// Checking if the subscription exists in the database
	subscription, err := db.Read(ctx, identifier)
	if err != nil {
		return err
	}

	// Deleting from the database
	req := `DELETE FROM subscriptions WHERE id = $1;`
	_, err = db.ExecContext(ctx, req, subscription.ID)
	if err != nil {
		return models.NewErrInternalServer(err)
	}

	return nil
}

// List returns an array of subscriptions filtered by user uuid and service name. The list of subscriptions can be filtered by the period, user uuid and service name
func (db *Database) List(ctx context.Context, params models.SubscriptionsWithinPeriod) ([]models.Subscription, error) {
	// Getting subscritions from the database
	var rows *sql.Rows
	query := `SELECT id, service_name, price, user_uuid, start_date, end_date, created_at, updated_at FROM subscriptions
		WHERE ($1::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR user_uuid = $1) AND ($2::text = ''::text OR service_name = $2) and 
		($4::timestamp IS NULL OR start_date <= $4) AND ($3::timestamp IS NULL OR end_date IS NULL OR end_date >= $3) ORDER BY id LIMIT $5 OFFSET $6;`
	rows, err := db.QueryContext(ctx, query, params.UserUUID, params.ServiceName, params.StartDate, params.EndDate, params.Limit, params.Offset)
	if err != nil {
		return []models.Subscription{}, err
	}
	defer rows.Close()

	// Parsing info to subscription type
	var subscriptions []models.Subscription
	for rows.Next() {
		var subscription models.Subscription
		err = rows.Scan(&subscription.ID, &subscription.ServiceName, &subscription.Price, &subscription.UserUUID, &subscription.StartDate, &subscription.EndDate,
			&subscription.CreatedAt, &subscription.UpdatedAt)
		if err != nil {
			return []models.Subscription{}, models.NewErrInternalServer(err)
		}
		subscriptions = append(subscriptions, subscription)
	}

	return subscriptions, nil
}

// Summary return amount of subscriptions within the provided period and total amount that was payed.
// The subscriptions can be filtered by the period, user uuid and service name
func (db *Database) Summary(ctx context.Context, params models.SubscriptionsWithinPeriod) (models.SummaryResponse, error) {
	// Getting subscritions from the database
	var amount, months, total int
	query := `SELECT 
		COUNT(*) AS amount,
		(EXTRACT(YEAR FROM $4::timestamp) - 
			EXTRACT(YEAR FROM $3::timestamp)) * 12 +
		EXTRACT(MONTH FROM $4::timestamp) - 
			EXTRACT(MONTH FROM $3::timestamp) + 1 AS months,
		SUM(
			((EXTRACT(YEAR FROM LEAST(COALESCE(end_date, $4::timestamp), $4::timestamp)) - 
                EXTRACT(YEAR FROM GREATEST(start_date, $3::timestamp))) * 12 +
			EXTRACT(MONTH FROM LEAST(COALESCE(end_date, $4::timestamp), $4::timestamp)) - 
                EXTRACT(MONTH FROM GREATEST(start_date, $3::timestamp)) + 1) 
			* price) AS total
		FROM subscriptions
		WHERE 
			($1::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR user_uuid = $1)
			AND ($2::text = ''::text OR service_name = $2)
			AND start_date <= $4 
			AND (end_date IS NULL OR end_date >= $3);`
	err := db.QueryRowContext(ctx, query, params.UserUUID, params.ServiceName, params.StartDate, params.EndDate).Scan(&amount, &months, &total)
	if err != nil {
		return models.SummaryResponse{}, models.NewErrInternalServer(err)
	}
	return models.SummaryResponse{Amount: amount, Months: months, Total: total}, nil
}
