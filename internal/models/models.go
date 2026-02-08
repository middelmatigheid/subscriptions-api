package models

import "errors"

type Subscription struct {
	Id          int    `json:"id" example:"1"`
	ServiceName string `json:"service_name" example:"Yandex Plus"`
	Price       int    `json:"price" example:"400"`
	UserId      string `json:"user_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   string `json:"start_date" example:"07-2025"`
	EndDate     string `json:"end_date" example:"08-2025"`
	CreatedAt   string `json:"created_at" example:"01-07-2025 14:00"`
	UpdatedAt   string `json:"updated_at" example:"01-07-2025 14:00"`
}

type IdResponse struct {
	Id int `json:"id" example:"1"`
}

type SummaryResponse struct {
	Amount int `json:"amount" example:"1"`
	Months int `json:"months" example:"2"`
	Total  int `json:"total" example:"400"`
}

// Custom error type to distinguish what type of problem occured
type Error struct {
	TypeOf error
	Err    error
}

func (err *Error) Type() string {
	if err.TypeOf != nil {
		return err.TypeOf.Error()
	}
	return "None"
}

func (err *Error) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	}
	return "None"
}

var (
	ErrConflict       error = errors.New("Conflict")
	ErrNotFound       error = errors.New("Not Found")
	ErrInternalServer error = errors.New("Internal Server Error")
	ErrTimeParse      error = errors.New("Time Parse Error")
)
