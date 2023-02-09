package views

import (
	"log"

	"github.com/cowboyhat-io/gittyupsec.com/models"
)

const (

	//AlertLvlError default bootstrap danger color
	AlertLvlError = "danger"
	// AlertLvlWarning maps to default bootstrap warning color
	AlertLvlWarning = "warning"
	// AlertLvlInfo maps to default bootstrap info color
	AlertLvlInfo = "info"
	// AlertLvlSuccess maps to default bootstrap success color
	AlertLvlSuccess = "success"
	// AlertMsgGeneric is displayed when any random error
	// is encountered by our backend.
	AlertMsgGeneric = "Something went wrong. Please try " +
		"again, and pricing us if the problem persists."
)

// Data is for content to be served in a
// webpage
type Data struct {
	Alert *Alert
	User  *models.User
	Yield interface{}
}

// Alert is for displaying bootstrap alert messages
// to the end user
type Alert struct {
	Level   string
	Message string
}

// PublicError is to allow for the generation
// of dynamic error messages
type PublicError interface {
	error
	Public() string
}

// SetAlert is to build the alert msg if possible
// else default to a generic message
func (d *Data) SetAlert(err error) {
	var msg string
	if pErr, ok := err.(PublicError); ok {
		msg = pErr.Public()
	} else {
		log.Println(err)
		msg = AlertMsgGeneric
	}
	d.Alert = &Alert{
		Level:   AlertLvlError,
		Message: msg,
	}
}

// AlertError accepts the alert data and
// creates an alert message for the webpage
func (d *Data) AlertError(msg string) {
	d.Alert = &Alert{
		Level:   AlertLvlError,
		Message: msg,
	}
}
