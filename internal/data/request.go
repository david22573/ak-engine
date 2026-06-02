package data

import "time"

type CandleRequest struct {
	Source   string
	Market   string
	Symbol   string
	Interval string
	From     time.Time
	To       time.Time
	Path     string
}
