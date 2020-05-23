package notification

import (
	"fmt"
	"log"

	"github.com/disrvptor/wifi_client_watch/preferences"
)

// TMobileSmsGateway implemets Notification
type TMobileSmsGateway struct{}

func init() {
	log.Println("Registering 'tmobile' notification driver")
	AddNotification("tmobile", &TMobileSmsGateway{})
}

// Send a notification via the T-Mobile SMS Gateway
func (gw *TMobileSmsGateway) Send(to string, message string, prefs *preferences.Preferences) error {
	// TODO: ensure only 10 digits in to
	return sendSmsGatewayMessage(fmt.Sprintf("%s@tmomail.net", to), message, prefs)
}
