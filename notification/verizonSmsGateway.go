package notification

import (
	"fmt"
	"log"

	"github.com/disrvptor/wifi_client_watch/preferences"
)

// VerizonSmsGateway implemets Notification
type VerizonSmsGateway struct{}

func init() {
	log.Println("Registering 'verizon' notification driver")
	AddNotification("verizon", &VerizonSmsGateway{})
}

// func (VerizonSmsGateway *) ToAddress(number string) string {
// 	return fmt.Sprintf("%s@vtext.com", number)
// }

// Send a notification via the Verizon SMS Gateway
func (gw *VerizonSmsGateway) Send(to string, message string, prefs *preferences.Preferences) error {
	// TODO: ensure only 10 digits in to
	return sendSmsGatewayMessage(fmt.Sprintf("%s@vtext.com", to), message, prefs)
}
