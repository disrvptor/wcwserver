package router

// Client is a Router client
type Client struct {
	Name   string `json:"name"`
	MAC    string `json:"mac"`
	IP     string `json:"ip"`
	Vendor string `json:"vendor"`
	Online bool   `json:"online"`
}

var routers map[string]Router = make(map[string]Router)

// Router is a router API
type Router interface {
	Connect(url string, username string, password string) error
	Clients() ([]Client, error)
}

// GetRouter returns a router interface for the given name
func GetRouter(name string) (Router, bool) {
	rtr, prs := routers[name]
	return rtr, prs
}

// AddRouter adds a router type instance to the lookup table
func AddRouter(name string, rtr Router) {
	routers[name] = rtr
}
