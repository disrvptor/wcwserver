package notification

import (
	"log"
	"strconv"

	"github.com/disrvptor/wifi_client_watch/preferences"
	"gopkg.in/gomail.v2"
)

func sendSmsGatewayMessage(to string, message string, prefs *preferences.Preferences) error {
	if true {
		log.Printf("Sending SMS notification to '%s' with message '%s'", to, message)
		return nil
	}

	// TODO: ensure message length <= 140 chars

	// https://godoc.org/gopkg.in/gomail.v2#example-package
	m := gomail.NewMessage()
	m.SetHeader("From", "no-reply@wifi_client_watch")
	m.SetHeader("To", to)
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	// m.SetHeader("Subject", "Hello!")
	// m.SetBody("text/html", "Hello <b>Bob</b> and <i>Cora</i>!")
	// m.Attach("/home/Alex/lolcat.jpg")
	m.SetBody("text/plain", message)

	// d := gomail.NewDialer("smtp.example.com", 587, "user", "123456")
	// https://pepipost.com/tutorials/send-an-email-via-gmail-smtp-server-using-php/
	smtpServer, _ := prefs.Get("smtp_server")
	_smtpPort, _ := prefs.Get("smtp_port")
	smtpPort, _ := strconv.Atoi(*_smtpPort)
	smtpUser, _ := prefs.Get("smtp_user")
	smtpPass, _ := prefs.Get("smtp_password")
	d := gomail.NewDialer(*smtpServer, smtpPort, *smtpUser, *smtpPass)

	// Send the email to Bob, Cora and Dan.
	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
