package models

import()

type Balance struct {
	UID *int `json:"-" db:"user_id"`
	Current *float64 `json:"current"`
	Withdrawn *float64 `json:"withdrawn"`
}
