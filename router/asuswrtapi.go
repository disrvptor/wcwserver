package router

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AsusConnection is a connection to an Asus router
type AsusConnection struct {
	url           string
	authorization string
	timestamp     time.Time
}

// AsusRouter is an Asus router
type AsusRouter struct {
	connection *AsusConnection
}

func init() {
	log.Println("Registering 'asuswrt' router driver")
	AddRouter("asuswrt", &AsusRouter{})
}

// ValidConnection is true if the connection is valid
func ValidConnection(conn *AsusConnection) bool {
	// See if the token is older than 1 hour
	if nil != conn {
		// log.Printf("%s ? %s", conn.timestamp, time.Now().Add(-1*time.Hour))
		return conn.timestamp.After(time.Now().Add(-1 * time.Hour))
	}
	return false
}

// Connect to a router
func (rtr *AsusRouter) Connect(url string, username string, password string) error {
	if nil != rtr.connection && ValidConnection(rtr.connection) {
		return nil
	}

	log.Println("AsusRouter: Invalid connection, attempting to connect")

	client := &http.Client{}

	// body := fmt.Sprintf("group_id=&action_mode=&action_script=&action_wait=5&current_page=Main_Login.asp&next_page=index.asp&login_authorization=%s", b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))))
	body := fmt.Sprintf("login_authorization=%s", b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))))
	// fmt.Println(body)
	loginURL := fmt.Sprintf("%s/login.cgi", url)
	// fmt.Println(loginURL)
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(body))
	if nil != err {
		return err
	}

	req.Header.Add("User-Agent", "asusrouter-Android-DUTUtil-1.0.0.3.58-163")
	// req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:73.0) Gecko/20100101 Firefox/73.0")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	if nil != err {
		return err
	}

	// for i, s := range resp.Header {
	// 	fmt.Printf("%s  ===  %s\n", i, s)
	// }
	// fmt.Println(resp.Body)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		bodyString := string(bodyBytes)
		log.Println(bodyString)

		var result map[string]interface{}
		json.Unmarshal([]byte(bodyString), &result)
		// Note: If result["error_status"] exists then the connection failed
		// TODO: Check for an error
		rtr.connection = &AsusConnection{url: url, authorization: result["asus_token"].(string), timestamp: time.Now()}
		log.Printf("Connected to %s with token %s", rtr.connection.url, rtr.connection.authorization)
		// fmt.Println(connection)
		return nil
	}

	return fmt.Errorf("Router responded with code %d", resp.StatusCode)
}

// Clients connected to the router
func (rtr *AsusRouter) Clients() ([]Client, error) {
	client := http.Client{}

	url := fmt.Sprintf("%s/appGet.cgi?hook=get_clientlist()", rtr.connection.url)
	// fmt.Println(url)
	req, err := http.NewRequest("GET", url, nil)
	if nil != err {
		return nil, err
	}

	req.Header.Add("User-Agent", "asusrouter-Android-DUTUtil-1.0.0.3.58-163")
	// req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:73.0) Gecko/20100101 Firefox/73.0")
	req.Header.Add("Accept", "application/json")
	req.AddCookie(&http.Cookie{Name: "asus_token", Value: rtr.connection.authorization})
	resp, err := client.Do(req)
	if nil != err {
		return nil, err
	}

	// for i, s := range resp.Header {
	// 	fmt.Printf("%s  ===  %s\n", i, s)
	// }
	// fmt.Println(resp.Body)

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		bodyString := string(bodyBytes)
		// fmt.Println(bodyString)

		var result map[string]interface{}
		json.Unmarshal([]byte(bodyString), &result)

		clientList := result["get_clientlist"].(map[string]interface{})
		macList := clientList["maclist"].([]interface{})
		clients := make([]Client, len(macList))
		for i, s := range macList {
			// fmt.Println(i, s)
			rawClient := clientList[s.(string)].(map[string]interface{})
			online, _ := strconv.ParseBool(rawClient["isOnline"].(string))
			clients[i] = Client{
				Name:   rawClient["name"].(string),
				MAC:    rawClient["mac"].(string),
				IP:     rawClient["ip"].(string),
				Vendor: rawClient["vendor"].(string),
				Online: online,
			}
		}
		// fmt.Println(clients)
		return clients, nil
	}

	return nil, fmt.Errorf("Router responded with code %d", resp.StatusCode)
}
