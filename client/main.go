// Package client - a pastbin client lib
//
// supports user accounts, add, delete and list pastes.
//
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"regexp"
)

const defaultUrl = "https://pastebin.com/api/"
const api_dev_key = "10b20d3ff00b856a455ba5004ea9d2a1" // api_dev_key of florianwalther

// Debug turns on debugging
// If Debug is set to true debugging output is turned on.
var Debug bool
var err error
var APIOption = "paste"
var APIPasteCode string

var file string // file to be uploaded
var SessionKey string

var expireValues = map[string]string{
	"N":   "Never",
	"10M": "10 Minutes",
	"1H":  "1 Hour",
	"1D":  "1 Day",
	"1W":  "1 Week",
	"2W":  "2 Weeks",
	"1M":  "1 Month",
	"6M":  "6 Month",
	"1Y":  "1 Year",
}

var defaultExpire = "10M"

// Parameters type used to hold the POST parameters to send to the API
type Parameters map[string]string

// Client a Pastebin API Client
type Client struct {
	Url        string      `json:"url"`        // url of the pastebin API to be used
	devKey     string      `json:"devkey"`     // api_dev_key to be used
	SessionKey string      `json:"SessionKey"` // api_user_key to be used to connect to the API
	client     http.Client `json:"client"`     // httpClient to be used to connect to the API
	Username   string      `json:"Username"`   // api_user_name to be used to login to the API
	password   string      `json:"password"`   // api_user_password to be used to login to the API
	Expire     string      `json:"expire"`     // default api_paste_expire_date for this client
}

func init() {
}

// OptionFunc is function used to configure the Client
type OptionFunc func(*Client) error

// SetClient configures the http Client used by the API Client
func SetClient(client http.Client) OptionFunc {
	return func(c *Client) error {
		c.client = client
		return nil
	}
}

// SetUrl configures the URL of the API used by the API Client
func SetUrl(urlA string) OptionFunc {
	u, err := url.Parse(urlA)
	if err != nil {
		return func(c *Client) error {
			return err
		}
	}
	return func(c *Client) error {
		c.Url = u.String()
		return nil
	}
}

// SetSession configures a api_user_key used by the API Client
func SetSession(session string) OptionFunc {
	if keyOK(session) == false {
		err := fmt.Errorf("supplied session does not seem to be a possible valid value")
		return func(c *Client) error {
			return err
		}
	}
	return func(c *Client) error {
		c.SessionKey = session
		return nil
	}
}

// SetUsername configures the api_user_name used by the API Client
func SetUsername(Username string) OptionFunc {
	return func(c *Client) error {
		c.Username = Username
		return nil
	}
}

// SetPassword configures the api_user_password used by the Client
func SetPassword(Password string) OptionFunc {
	return func(c *Client) error {
		c.password = Password
		return nil
	}
}

func SetExpire(expire string) OptionFunc {
	return func(c *Client) error {
		// TODO: check if expire value is valid
		_, ok := expireValues[expire]
		if ok == true {
			c.Expire = expire
			return nil
		}
		err = fmt.Errorf("expire value is not valid. Valid values are: %v\n", expireValues)
		return err
	}
}

// HasPassword checks if a password was set
func (c *Client) HasPassword() bool {
	if c.password != "" {
		return true
	}
	return false
}

// SetDevKey configures a non default api_dev_key onto the Client
// By default the hard-coded api_dev_key will be used.
// Usually this option is not needed, since going with the default is fine.
// See constant api_dev_key.
func SetDevKey(DevKey string) OptionFunc {
	if keyOK(DevKey) == false {
		err := fmt.Errorf("supplied DevKey does not seem to be a possible valid key")
		return func(c *Client) error {
			return err
		}
	}
	return func(c *Client) error {
		c.devKey = DevKey
		return nil
	}
}

// New returns a new Pastebin API Client
//
// simple usage example, post as a user:
//
// apiClient, err := Client.New(
//	Client.SetUsername("johndoe"),
//	Client.SetPassword("superSecretPassword")
// )
// check(err)
//
// err := apiClient.Login()
// check(err)
//
// pasteUrl, err := apiClient.NewPasteFromFile("test.txt")
// check(url)
// fmt.Printf("%s\n", pasteUrl)
//
//
// posting anonymously as a guest is even more simple:
//
// apiClient, err := client.New()
// pasteUrl, err := apiClient.NewPasteFromFile("test.txt")
// check(url)
// fmt.Printf("%s\n", pasteUrl)
//
func New(options ...OptionFunc) (c *Client, err error) {
	c = new(Client)
	c.Url = defaultUrl
	c.Expire = defaultExpire
	c.devKey = api_dev_key
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *Client) Update(options ...OptionFunc) (err error) {
	for _, option := range options {
		if err := option(c); err != nil {
			return err
		}
	}
	return nil
}

// takes parameters and returns a reader and the content-type string
// it basically writes a multipart/mime body from the parameters supplied.
func parametersToBodyReader(parameters Parameters) (io.Reader, string) {
	// encode parameters as body for the POST request
	bodyReader, bodyWriter := io.Pipe()
	// create a multipat/mime writer
	writer := multipart.NewWriter(bodyWriter)
	fdct := writer.FormDataContentType()
	// get the Content-Type of our form data
	errChan := make(chan error, 1)
	go func() {
		defer bodyWriter.Close()
		for k, v := range parameters {
			if err := writer.WriteField(k, v); err != nil {
				errChan <- err
				return

			}

		}
		errChan <- writer.Close()

	}()
	return bodyReader, fdct
}

// Save will serialize the Client and save it to disk
func SaveClient(Client *Client, filename string) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	// serialize and write to file
	err = enc.Encode(Client)
	if err != nil {
		return err
	}
	// return
	return nil
}

// Restore will load a serialized Client from disk, deserialize and return the saved Client
func RestoreClient(filename string) (c *Client, err error) {
	// open file for reading
	f, err := os.Open(filename)
	if err != nil {
		return c, err
	}
	defer f.Close()
	// setup gob decoder reading from file opened
	dec := json.NewDecoder(f)
	// decode file to Client
	err = dec.Decode(&c)
	if err != nil {
		return c, err
	}
	if c.devKey == "" {
		c.devKey = api_dev_key
		if Debug {
			log.Printf("api_dev_key set to default")
		}
	}
	// return
	return c, nil
}

// Login - log into pastebin.com as a given user, with a given Password,
// retrieves a sessionkey to be used in subservient requests
// SessionKey will be configured to the Client
func (c *Client) Login() (SessionKey string, err error) {
	if c.Username == "" || c.password == "" {
		err = fmt.Errorf("login not possible, Username and Password not set in Client")
		return "", err
	}
	// generate request
	parameters := Parameters{
		"api_dev_key":       c.devKey,
		"api_user_name":     c.Username,
		"api_user_password": c.password,
	}
	// creating the fullurl for the request
	endpoint := "api_login.php"
	fullurl := createFullurl(c.Url, endpoint)
	// prepare request
	req, err := prepareRequest(parameters, fullurl)
	if err != nil {
		return SessionKey, err
	}
	if Debug == true {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			return SessionKey, err
		}
		log.Printf("REQUEST:\n%s\n====\n", dump)
	}
	// send the request, get response
	resp, err := c.client.Do(req)
	if err != nil {
		return SessionKey, err
	}
	// handle response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return SessionKey, err
	}
	if resp.StatusCode == 200 && keyOK(string(body)) {
		// set the SessionKey within the Client
		c.SessionKey = string(body)
		// delete the password for security reasons
		c.password = ""
		//log.Printf("%s\n", body)
		return string(body), nil
	} else {
		err = fmt.Errorf("login failed %s '%s'", resp.Status, body)
		return string(body), err
	}
}

// keyOK - checks if a suspected key is OK and could possibly a valid key.
func keyOK(key string) bool {
	var re = regexp.MustCompile(`(?mis)^[a-f0-9]{32}$`)
	return re.MatchString(key)
}

// NewPasteFromFile basically uploads a file to pastebin
func (c *Client) NewPasteFromFile(filename string) (urlA string, err error) {
	// open file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return urlA, err
	}
	parameters := Parameters{
		"api_dev_key":           c.devKey,
		"api_option":            "paste",
		"api_paste_name":        filename,
		"api_paste_code":        string(data),
		"api_paste_expire_date": c.Expire,
		"api_paste_private":     "1",
		//p["api_paste_format"]
	}
	//log.Printf("NewPasteFromFile: SessionKey: %+v", SessionKey)
	if c.SessionKey != "" {
		parameters["api_user_key"] = c.SessionKey
		//log.Printf("useing SessionKey '%s' as api_user_key", SessionKey)
	}
	//log.Printf("%+v", parameters)
	// creating the fullurl for the request
	endpoint := "api_post.php"
	fullurl := createFullurl(c.Url, endpoint)
	req, err := prepareRequest(parameters, fullurl)
	if err != nil {
		return urlA, err
	}
	if Debug == true {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			return urlA, err
		}
		log.Printf("REQUEST:\n%s\n====\n", dump)
	}
	// create body and options
	resp, err := c.client.Do(req)
	if err != nil {
		return urlA, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return urlA, err
	}
	urlA = string(body)
	// TODO: check if url starts with 'http'
	return urlA, err
}

func (c *Client) DeletePaste(code string) (urlA string, err error) {
	// to delete make sure we have a api_user_key (SessionKey) set in the Client
	if keyOK(c.SessionKey) {
		parameters := Parameters{
			"api_user_key":  c.SessionKey,
			"api_paste_key": code,
			"api_dev_key":   c.devKey,
			"api_option":    "delete",
		}
		// creating the fullurl for the request
		endpoint := "api_post.php"
		fullurl := createFullurl(c.Url, endpoint)
		req, err := prepareRequest(parameters, fullurl)
		if err != nil {
			return urlA, err
		}
		if Debug == true {
			dump, err := httputil.DumpRequestOut(req, true)
			if err != nil {
				return urlA, err
			}
			log.Printf("REQUEST:\n%s\n====\n", dump)
		}
		// create body and options
		resp, err := c.client.Do(req)
		if err != nil {
			return urlA, err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return urlA, err
		}
		urlA = string(body)
		// TODO: check if url starts with 'http'
		if resp.StatusCode == 200 {
			//fmt.Printf("%s\n", urlA)
		}
		return urlA, err
	} else {
		err = fmt.Errorf("you are not logged in, login first")
		return urlA, err
	}
}

func createFullurl(apiurl, endpoint string) (fullurl string) {
	u, err := url.Parse(apiurl)
	if err != nil {
		return fullurl
	}
	u.Path = path.Join(u.Path, endpoint)
	fullurl = u.String()
	return fullurl
}

func (c *Client) ListPastes() (results string, err error) {
	if keyOK(c.SessionKey) {
		parameters := Parameters{
			"api_dev_key":       c.devKey,
			"api_user_key":      c.SessionKey,
			"api_results_limit": "100",
			"api_option":        "list",
		}
		// creating the fullurl for the request
		endpoint := "api_post.php"
		fullurl := createFullurl(c.Url, endpoint)

		req, err := prepareRequest(parameters, fullurl)
		if err != nil {
			return results, err
		}
		if Debug == true {
			dump, err := httputil.DumpRequestOut(req, true)
			if err != nil {
				return results, err
			}
			log.Printf("REQUEST:\n%s\n====\n", dump)
		}
		// create body and options
		resp, err := c.client.Do(req)
		if err != nil {
			return results, err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return results, err
		}
		results = string(body)
		if resp.StatusCode == 200 {
			//fmt.Printf("%s\n", results)
		}
		return results, err
	} else {
		err = fmt.Errorf("you are not logged in, login first")
		return results, err
	}
}

// TODO ParsePastes function that can parse the results for ListPastes
func ParsePastes(pastes string) {

}

// prepareRequest helper function to create a HTTP requests with given parameters for a given API endpoint.
func prepareRequest(parameters Parameters, fullurl string) (req *http.Request, err error) {
	bodyReader, fdct := parametersToBodyReader(parameters)
	req, err = http.NewRequest("POST", fullurl, bodyReader)
	if err != nil {
		return req, err
	}
	req.Header.Add("Content-Type", fdct)
	return req, err
}
