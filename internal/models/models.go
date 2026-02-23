package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Storage interface {
	Close() error

	Create(context.Context, Subscription) (IDResponse, error)
	Read(context.Context, SubscriptionIdentifier) (Subscription, error)
	Update(context.Context, Subscription) error
	Delete(context.Context, SubscriptionIdentifier) error
	List(context.Context, SubscriptionsWithinPeriod) ([]Subscription, error)
	Summary(context.Context, SubscriptionsWithinPeriod) (SummaryResponse, error)
}

type SubscriptionService interface {
	Create(context.Context, Subscription) (IDResponse, error)
	Read(context.Context, SubscriptionIdentifier) (Subscription, error)
	Update(context.Context, Subscription) error
	Patch(context.Context, SubscriptionPatch) error
	Delete(context.Context, SubscriptionIdentifier) error
	List(context.Context, SubscriptionsWithinPeriod) ([]Subscription, error)
	Summary(context.Context, SubscriptionsWithinPeriod) (SummaryResponse, error)
}

// Custom date to deal with right format and null fields
type CustomDate struct {
	sql.NullTime `swaggerignore:"true"`
}

func (cd *CustomDate) ToString() string {
	if cd.Valid {
		return cd.Time.Format("01-2006")
	}
	return "null"
}

func (cd *CustomDate) Scan(value any) error {
	return cd.NullTime.Scan(value)
}

func (cd CustomDate) Value() (driver.Value, error) {
	return cd.NullTime.Value()
}

func (cd *CustomDate) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		cd.Valid = false
		return nil
	}

	if len(b) < 2 {
		return NewErrInternalServer(errors.New("Invalid JSON string"))
	}
	s := string(b[1 : len(b)-1])

	t, err := time.Parse("01-2006", s)
	if err != nil {
		return NewErrInternalServer(err)
	}

	cd.Time = t
	cd.Valid = true
	return nil
}

func (cd *CustomDate) MarshalJSON() ([]byte, error) {
	if cd.Valid {
		return []byte(fmt.Sprintf(`"%s"`, cd.Time.Format("01-2006"))), nil
	}
	return []byte("null"), nil
}

// Custom time to deal with right format and null fields
type CustomTime struct {
	sql.NullTime `swaggerignore:"true"`
}

func (ct *CustomTime) ToString() string {
	if ct.Valid {
		return ct.Time.Format("02-01-2006 15:04")
	}
	return "null"
}

func (ct *CustomTime) Scan(value any) error {
	return ct.NullTime.Scan(value)
}

func (ct CustomTime) Value() (driver.Value, error) {
	return ct.NullTime.Value()
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		ct.Valid = false
		return nil
	}

	if len(b) < 2 {
		return NewErrInternalServer(errors.New("Invalid JSON string"))
	}
	s := string(b[1 : len(b)-1])

	t, err := time.Parse("02-01-2006 15:04", s)
	if err != nil {
		return NewErrInternalServer(err)
	}

	ct.Time = t
	ct.Valid = true
	return nil
}

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	if ct.Valid {
		return []byte(fmt.Sprintf(`"%s"`, ct.Time.Format("02-01-2006 15:04"))), nil
	}
	return []byte("null"), nil
}

type Subscription struct {
	ID          int        `json:"id" example:"1"`
	ServiceName string     `json:"service_name" example:"Yandex Plus"`
	Price       int        `json:"price" example:"400"`
	UserUUID    uuid.UUID  `json:"user_uuid" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   CustomDate `json:"start_date" example:"07-2025" swaggertype:"string"`
	EndDate     CustomDate `json:"end_date" example:"08-2025" swaggertype:"string"`
	CreatedAt   CustomTime `json:"created_at" example:"01-07-2025 14:00" swaggerignore:"true"`
	UpdatedAt   CustomTime `json:"updated_at" example:"01-07-2025 14:00" swaggerignore:"true"`
}

// The pointers is being used to identify them from invalid empty request because in the patch endpoint some fields can be not provided
type SubscriptionPatch struct {
	ID          int         `json:"id" example:"1"`
	ServiceName *string     `json:"service_name" example:"Yandex Plus"`
	Price       *int        `json:"price" example:"400"`
	UserUUID    *uuid.UUID  `json:"user_uuid" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   *CustomDate `json:"start_date" example:"07-2025" swaggertype:"string"`
	EndDate     *CustomDate `json:"end_date" example:"08-2025" swaggertype:"string"`
}

type IDResponse struct {
	ID int `json:"id" example:"1"`
}

type SubscriptionIdentifier struct {
	ID          int
	ServiceName string
	UserUUID    uuid.UUID
}

type SubscriptionsWithinPeriod struct {
	ServiceName string     `json:"service_name"`
	UserUUID    uuid.UUID  `json:"user_uuid"`
	StartDate   CustomDate `json:"start_date"`
	EndDate     CustomDate `json:"end_date"`
	Limit       int        `json:"limit"`
	Offset      int        `json:"offset"`
}

type SummaryResponse struct {
	Amount int `json:"amount" example:"1"`
	Months int `json:"months" example:"2"`
	Total  int `json:"total" example:"400"`
}

// Custom errors
var (
	ErrConflict       error = errors.New("Conflict")
	ErrNotFound       error = errors.New("Not Found")
	ErrInternalServer error = errors.New("Internal Server Error")
	ErrBadRequest     error = errors.New("Bad request")
)

func NewErrConflict() error {
	return ErrConflict
}

func NewErrNotFound() error {
	return ErrNotFound
}

func NewErrInternalServer(err error) error {
	return fmt.Errorf("%w: %w", ErrInternalServer, err)
}

func NewErrBadRequest(err error) error {
	return fmt.Errorf("%w: %w", ErrBadRequest, err)
}
