package domain

import (
	"errors"
	"time"
)

type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Desc      string    `json:"desc"`
	Status    string    `json:"status"`
	Priority  int       `json:"priority"`
	DueDate   time.Time `json:"due_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrNotFound = errors.New("task not found")
	ErrInvalid  = errors.New("invalid task data")
)

const (
	StatusOpen    = "open"
	StatusDoing   = "doing"
	StatusDone    = "done"
	StatusArchved = "archived"
)

func ValidStatus(s string) bool {
	switch s {
	case StatusOpen, StatusDoing, StatusDone, StatusArchved:
		return true
	}
	return false
}
