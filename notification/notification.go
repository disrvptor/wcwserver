package notification

import "github.com/disrvptor/wifi_client_watch/preferences"

// Notification is a notification method
type Notification interface {
	Send(to string, message string, preferences *preferences.Preferences) error
}

var notifications map[string]Notification = make(map[string]Notification)

// GetNotification returns a Notification interface for the given name
func GetNotification(name string) (Notification, bool) {
	rtr, prs := notifications[name]
	return rtr, prs
}

// AddNotification adds a Notification type instance to the lookup table
func AddNotification(name string, notif Notification) {
	notifications[name] = notif
}
