package entities

type Email struct {
	ID      string `json:"id"`
	To      string `json:"to_address"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type CreateEmail struct {
	To      string `json:"to_address" validate:"email,required"`
	Subject string `json:"subject" validate:"required"`
	Body    string `json:"body" validate:"required"`
}
