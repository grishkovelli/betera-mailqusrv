package entities

// Email status constants
const (
	Failed     = "failed"     // Email delivery failed
	Pending    = "pending"    // Email is waiting to be processed
	Processing = "processing" // Email is currently being processed
	Sent       = "sent"       // Email was successfully sent
)

// Email represents an email record in the system
type Email struct {
	ID      int    `db:"id" json:"id"`                 // Unique identifier
	To      string `db:"to_address" json:"to_address"` // Recipient email address
	Subject string `db:"subject" json:"subject"`       // Email subject
	Body    string `db:"body" json:"body"`             // Email body content
	Status  string `db:"status" json:"status"`         // Current status of the email
}

// CreateEmail represents the data needed to create a new email
type CreateEmail struct {
	To      string `json:"to_address" validate:"email,required"` // Recipient email address
	Subject string `json:"subject" validate:"required"`          // Email subject
	Body    string `json:"body" validate:"required"`             // Email body content
}
