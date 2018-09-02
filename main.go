package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

/* Constants */
const (
	HOSTNAME = "https://hostname"
	USERNAME = "USERNAME"
	PASSWORD = "PASSWORD"

	DATACENTER          = "DATACENTER_NAME"
	PRODUCT             = "generic-host"
	MONITOR             = "/Common/tcp"
	PARTITION           = "Common"
	LOAD_BALANCING_MODE = "round-robin"

	DOMAIN      = "demo.com"
	SERVER_NAME = "server_1.1.2.1"
	POOL_NAME   = "pool_demo.com"
	IP          = "1.1.2.1"
	TTL         = 30
)

var VIRTUAL_SERVERS = []VirtualServer{
	VirtualServer{Name: "vs_1.1.2.1:80", Destination: "1.1.2.1:80"},
	VirtualServer{Name: "vs_1.1.2.1:81", Destination: "1.1.2.1:81"},
}

type Token struct {
	Token            string `json:"token"`
	Name             string `json:"name"`
	UserName         string `json:"userName"`
	AuthProviderName string `json:"authProviderName"`
}

type Auth struct {
	Username          string `json:"username"`
	LoginProviderName string `json:"loginProviderName"`
	Token             Token  `json:"token"`
}

type GtmCollection struct {
	Kind  string                   `json:"kind"`
	Items []map[string]interface{} `json:"items"`
}

type VirtualServer struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
}

// Get F5 authentication token.
func GetAuthToken() (string, error) {
	var params = map[string]string{
		"username": USERNAME,
		"password": PASSWORD,
	}

	data := new(bytes.Buffer)
	json.NewEncoder(data).Encode(&params)

	client := GetClient()
	res, err := client.Post(HOSTNAME+"/mgmt/shared/authn/login", "application/json; charset=utf-8", data)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	var auth Auth
	err = json.NewDecoder(res.Body).Decode(&auth)

	if err != nil {
		return "", err
	}

	if len(auth.Token.Token) == 0 {
		return "", errors.New("Parsing token failed, token is empty.")
	}

	return auth.Token.Token, nil
}

// Get server list.
func GetServers() []map[string]interface{} {
	token, err := GetAuthToken()

	if err != nil {
		log.Fatal(err)
	}

	req, err := getReq("GET", "/mgmt/tm/gtm/server", token, nil)
	if err != nil {
		log.Fatal(err)
	}

	client := GetClient()
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	var collection GtmCollection
	json.NewDecoder(res.Body).Decode(&collection)

	return collection.Items
}

// Create a server, with addresses and virtual servers.
func CreateServer() {
	type VirtualServer struct {
		Name        string `json:"name"`
		Destination string `json:"destination"`
	}
	addresses := []map[string]string{
		{"name": IP, "deviceName": SERVER_NAME},
	}

	params := make(map[string]interface{})
	params["name"] = SERVER_NAME
	params["datacenter"] = DATACENTER
	params["product"] = PRODUCT
	params["monitor"] = MONITOR
	params["addresses"] = addresses
	params["virtualServers"] = VIRTUAL_SERVERS

	token, _ := GetAuthToken()
	req, _ := getReq("POST", "/mgmt/tm/gtm/server", token, params)
	client := GetClient()
	res, _ := client.Do(req)

	defer res.Body.Close()
}

// Create a pool a, with pool members.
func CreatePool() string {
	members := []map[string]string{
		{"name": "server_1.1.2.1:vs_1.1.2.1:80"},
		{"name": "server_1.1.2.1:vs_1.1.2.1:81"},
	}

	params := make(map[string]interface{})
	params["name"] = "pool_demo.com"
	params["partition"] = PARTITION
	params["monitor"] = MONITOR
	params["ttl"] = TTL
	params["alternate_mode"] = LOAD_BALANCING_MODE
	params["load_balancing_mode"] = LOAD_BALANCING_MODE
	params["members"] = members

	token, _ := GetAuthToken()
	req, _ := getReq("POST", "/mgmt/tm/gtm/pool/a", token, params)
	client := GetClient()
	res, _ := client.Do(req)

	defer res.Body.Close()

	return convertBodyToString(res.Body)
}

// Create Wideip a
func CreateWideip() string {
	pools := []map[string]string{
		{"name": "pool_demo.com"},
	}
	params := make(map[string]interface{})
	params["name"] = DOMAIN
	params["partition"] = PARTITION
	params["pool_lb_mode"] = LOAD_BALANCING_MODE
	params["ttl"] = TTL
	params["description"] = "demo.com's description"
	params["pools"] = pools

	token, _ := GetAuthToken()
	req, _ := getReq("POST", "/mgmt/tm/gtm/wideip/a", token, params)
	client := GetClient()
	res, _ := client.Do(req)

	defer res.Body.Close()

	return convertBodyToString(res.Body)
}

// Get HTTP client, which is disabled security checks.
func GetClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return client
}

// Create HTTP request helper
func getReq(method, uri, token string, params map[string]interface{}) (*http.Request, error) {

	data := new(bytes.Buffer)
	if params != nil {
		json.NewEncoder(data).Encode(&params)
	}

	req, err := http.NewRequest(method, HOSTNAME+uri, data)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-F5-Auth-Token", token)

	return req, nil
}

// Convert response body to string data
func convertBodyToString(body io.ReadCloser) string {
	if body == nil {
		return ""
	}

	bytes, _ := ioutil.ReadAll(body)
	return string(bytes)
}

func main() {
	/*
		Step 1: Create servers and virtual servers.
		Step 2: Create pools.
		Step 3: Create a wide ip.
	*/
}
