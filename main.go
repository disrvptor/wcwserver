package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/disrvptor/wifi_client_watch/notification"
	"github.com/disrvptor/wifi_client_watch/preferences"
	"github.com/disrvptor/wifi_client_watch/router"
	_ "github.com/mattn/go-sqlite3"
)

type wifiClientWatchApp struct {
	clients       []router.Client
	ignoredMacs   []string
	myRouter      interface{ router.Router }
	notifications interface{ notification.Notification }
	preferences   preferences.Preferences
	dbFile        string
	stopPoller    chan bool
	ticker        *time.Ticker
}

var application *wifiClientWatchApp

type message struct {
	Message string `json:"message"`
}

func ignoreClient(mac string) {
	log.Printf("Ignoring client %s", mac)
	if contains(application.ignoredMacs, mac) {
		log.Printf("Already ignoring %s", mac)
	} else {
		application.ignoredMacs = append(application.ignoredMacs, mac)
		// TODO: Serialize this in the DB
	}
}

func unignoreClient(mac string) {
	log.Printf("Unignoring client %s", mac)
	if !contains(application.ignoredMacs, mac) {
		log.Printf("Already unignoring %s", mac)
	} else {
		// TODO: Remove MAC from the list
		// TODO: Serialize this in the DB
	}
}

// Not for production use!!
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func clientsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling client request")
	w.Header().Set("Content-Type", "application/json")
	enableCors(&w)
	switch r.URL.Query().Get("action") {
	case "ignore":
		mac := r.URL.Query().Get("mac")
		ignoreClient(mac)
		b, err := json.Marshal(message{Message: "ok"})
		if err != nil {
			http.Error(w, "Cannot create ok message", 500)
		} else {
			w.Write(b)
		}
	case "unignore":
		mac := r.URL.Query().Get("mac")
		unignoreClient(mac)
		b, err := json.Marshal(message{Message: "ok"})
		if err != nil {
			http.Error(w, "Cannot create ok message", 500)
		} else {
			w.Write(b)
		}
	case "ignored":
		log.Println("Returning ignored MACs")
		b, err := json.Marshal(application.ignoredMacs)
		if err != nil {
			http.Error(w, "Cannot read ignored MACs", 500)
		} else {
			w.Write(b)
		}
	default:
		log.Println("Returning clients")
		b, err := json.Marshal(application.clients)
		if err != nil {
			http.Error(w, "Cannot read clients", 500)
		} else {
			w.Write(b)
		}
	}
}

func prefsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling preferences request")
	w.Header().Set("Content-Type", "application/json")
	enableCors(&w)
	switch r.URL.Query().Get("action") {
	case "set":
		name := r.URL.Query().Get("name")
		value := r.URL.Query().Get("value")
		secure, _ := strconv.ParseBool(r.URL.Query().Get("secure"))
		log.Printf("Setting preference %s=%s, secure=%t", name, value, secure)
		application.preferences.Set(name, value, secure)
		b, err := json.Marshal(message{Message: "ok"})
		if err != nil {
			http.Error(w, "Cannot create ok message", 500)
		} else {
			w.Write(b)
		}
	default:
		log.Println("Returning preferences")
		b, err := json.Marshal(application.preferences)
		if err != nil {
			http.Error(w, "Cannot read preferences", 500)
		} else {
			w.Write(b)
		}
	}
}

func checkClients() {
	log.Println("Beginning checking clients")

	url, prs := application.preferences.Get("url")
	if !prs {
		log.Println("No router URL defined")
		log.Println("Ended checking clients")
		return
	}
	username, prs := application.preferences.Get("username")
	if !prs {
		log.Println("No router Username defined")
		log.Println("Ended checking clients")
		return
	}
	password, prs := application.preferences.Get("password")
	if !prs {
		log.Println("No router Password defined")
		log.Println("Ended checking clients")
		return
	}

	err := (application.myRouter).Connect(*url, *username, *password)
	if nil != err {
		log.Println("An error occurred establishing a connection:", err)
		log.Println("Ended checking clients")
		return
	}

	// Get the clients
	newClients, err := (application.myRouter).Clients()
	if err != nil {
		log.Println("An error occurred retrieving the client list:", err)
	}

	// TODO: Compare clients and newClients for differences
	if nil != application.clients {
		// droppedClients := make([]asuswrtapi.Client, 0)
		for _, c := range application.clients {
			c2 := findClient(c.MAC, newClients)
			if nil == c2 {
				// droppedClients = append(droppedClients, c)
				log.Printf("Dropped client %s (MAC=%s, IP=%s)", c.Name, c.MAC, c.IP)
			} else if c.Online && !c2.Online {
				log.Printf("Offlined client %s (MAC=%s, IP=%s)", c.Name, c.MAC, c.IP)
			}
		}
	}

	if nil != newClients {
		// addedClients := make([]asuswrtapi.Client, 0)
		for _, c := range newClients {
			c2 := findClient(c.MAC, application.clients)
			if nil == c2 {
				// addedClients = append(addedClients, c)
				log.Printf("New client %s (MAC=%s, IP=%s)", c.Name, c.MAC, c.IP)
				if !contains(application.ignoredMacs, c.MAC) {
					sendNotification(fmt.Sprintf("New client %s (MAC=%s, IP=%s)", c.Name, c.MAC, c.IP))
				}
			} else if !c2.Online && c.Online {
				log.Printf("Onlined client %s (MAC=%s, IP=%s)", c.Name, c.MAC, c.IP)
				if !contains(application.ignoredMacs, c.MAC) {
					sendNotification(fmt.Sprintf("Connected client %s (MAC=%s, IP=%s)", c.Name, c.MAC, c.IP))
				}
			}
		}

		// Save the list of new clients in the DB
		db, err := sql.Open("sqlite3", "./wifi_client_watch.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		_, err = db.Exec("delete from clients;")
		if err != nil {
			log.Fatal(err)
		}
		stmt, err := db.Prepare(`
		insert into clients (name, ip, mac, vendor, online) values (?,?,?,?,?)
			on conflict(mac) do update set name = ?, ip = ?, vendor = ?, online = ?;
		`)
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		for _, c := range newClients {
			_, err = stmt.Exec(c.Name, c.IP, c.MAC, c.Vendor, c.Online, c.Name, c.IP, c.Vendor, c.Online)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Save our list of the new clients
		application.clients = newClients
	}
	log.Println("Ended checking clients")
}

func sendNotification(message string) {
	to, _ := application.preferences.Get("notification_to")
	application.notifications.Send(*to, message, &application.preferences)
}

func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func findClient(mac string, clients []router.Client) *router.Client {
	for _, c := range clients {
		if c.MAC == mac {
			return &c
		}
	}
	return nil
}

func readClients(app *wifiClientWatchApp) {
	var dbClients = make([]router.Client, 0)

	log.Printf("Reading clients from '%s'", app.dbFile)
	db, err := sql.Open("sqlite3", app.dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	sqlStmt := `
	create table IF NOT EXISTS clients (name text, ip text, mac text primary key, vendor text, online bool);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	rows, err := db.Query("select name, ip, mac, vendor, online from clients")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var ip string
		var mac string
		var vendor string
		var online bool
		err = rows.Scan(&name, &ip, &mac, &vendor, &online)
		if err != nil {
			log.Fatal(err)
		}
		dbClients = append(dbClients, router.Client{
			Name:   name,
			IP:     ip,
			MAC:    mac,
			Vendor: vendor,
			Online: online,
		})
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	application.clients = dbClients
}

func readIgnoredMacs(app *wifiClientWatchApp) {
	var ignoredMacs = make([]string, 0)

	log.Printf("Reading ignored MACs from '%s'", app.dbFile)
	db, err := sql.Open("sqlite3", app.dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	sqlStmt := `
	create table IF NOT EXISTS ignored_macs (mac text primary key, ignore bool);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	rows, err := db.Query("select mac, ignore from ignored_macs")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var mac string
		var ignored bool
		err = rows.Scan(&mac, &ignored)
		if err != nil {
			log.Fatal(err)
		}
		if ignored {
			ignoredMacs = append(ignoredMacs, mac)
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	application.ignoredMacs = ignoredMacs
}

func main() {
	application = &wifiClientWatchApp{
		dbFile: "./wifi_client_watch.db",
	}
	// application.myRouter = &asuswrtapi.AsusRouter{}

	// readPreferences(application)
	application.preferences.SetBackingStore(application.dbFile)

	// ensure default preferences
	application.preferences.SetDefaultPreference("poll_time", "60")
	application.preferences.SetDefaultPreference("url", "http://192.168.1.1")
	application.preferences.SetDefaultPreference("username", "admin", true)
	application.preferences.SetDefaultPreference("password", "admin", true)
	application.preferences.SetDefaultPreference("router", "asuswrt")
	application.preferences.SetDefaultPreference("notification", "verizon")
	application.preferences.SetDefaultPreference("notification_to", "555-123-6789")
	application.preferences.SetDefaultPreference("smtp_server", "smtp.gmail.com")
	application.preferences.SetDefaultPreference("smtp_port", "587")
	application.preferences.SetDefaultPreference("smtp_user", "user@gmail.com", true)
	application.preferences.SetDefaultPreference("smtp_pass", "password", true)

	rtr, prs := application.preferences.Get("router")
	application.myRouter, prs = router.GetRouter(*rtr)
	if !prs {
		log.Fatalf("A router implementation for %s was not found", *rtr)
	}

	readClients(application)
	readIgnoredMacs(application)

	notif, prs := application.preferences.Get("notification")
	application.notifications, prs = notification.GetNotification(*notif)
	if !prs {
		log.Fatalf("A notification implementation for %s was not found", *notif)
	}

	startBackgroundTask(checkClients)
	application.preferences.AddWatcher("poll_time", restartBackgroundTask)

	fs := http.FileServer(http.Dir("../wcwweb/dist/wcwweb"))
	// // http.Handle("/ui/", http.StripPrefix("/ui", fs))
	// jsFile := regexp.MustCompile("\\.js$")
	// http.HandleFunc("/ui/", func(w http.ResponseWriter, r *http.Request) {
	// 	ruri := r.RequestURI
	// 	if jsFile.MatchString(ruri) {
	// 		log.Printf("Setting content-type=text/javascript for %s", ruri)
	// 		w.Header().Set("Content-Type", "text/javascript")
	// 	} else {
	// 		log.Printf("Using default content-type for %s", ruri)
	// 	}
	// 	http.StripPrefix("/ui", fs).ServeHTTP(w, r)
	// })

	http.HandleFunc("/clients", clientsHandler)
	http.HandleFunc("/preferences", prefsHandler)
	// http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
	// 	// The "/" pattern matches everything, so we need to check
	// 	// that we're at the root here.
	// 	if req.URL.Path == "/" {
	// 		http.Redirect(w, req, "/ui/index.html", http.StatusPermanentRedirect)
	// 		return
	// 	}
	// 	fmt.Fprintf(w, "Welcome to the home page!")
	// })
	http.Handle("/", fs)

	log.Printf("Starting server")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
