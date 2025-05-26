package entities

const (
	Failed     = "failed"
	Pending    = "pending"
	Processing = "processing"
	Sent       = "sent"
)

type Email struct {
	ID      int    `db:"id" json:"id"`
	To      string `db:"to_address" json:"to_address"`
	Subject string `db:"subject" json:"subject"`
	Body    string `db:"body" json:"body"`
	Status  string `db:"status" json:"status"`
}

type CreateEmail struct {
	To      string `json:"to_address" validate:"email,required"`
	Subject string `json:"subject" validate:"required"`
	Body    string `json:"body" validate:"required"`
}
