// pastebin cmd
//
// command line tool to post files on pastebin.com, or delete them.
//
// INSTALL:
//
// pastebin setup
//
// go get gtihub.com/scusi/pastebin
//
// The above command should fetch and install everything needed.
// NOTE: You need to have a working go workspace setup for this to work.
//
// SETUP:
//
// If you want to user your pastebin.com useraccount you need to setup the pastebin command first.
// This step is required only once.
// Your configuration will be saved at $HOME/.pastebin, or any other file you specify.
// Your pastebin password will just used during detup to retrieve a valid session token.
// The session token will be used for future requests.
// Your pastebin password will not be saved to disk.
//
// You can change the used user credentials by re-run 'pastebin setup'
//
// If you do not setup your client you will paste as a anonymous guest user.
// Same holds true if you set the '-a' flag.
//
// USAGE:
//
// After Install and Setup you can use 'pastebin' like this:
//
// Add a file to pastebin:
//
// me@mybox:~/$ pastebin add main.go
// main.go on pastebin: https://pastebin.com/B0SdAwyV
// https://pastebin.com/B0SdAwyV
//
// Delete a post from pastebin:
//
// me@mybox:~/$ pastebin del B0SdAwyV
// Paste Removed
//
// Paste a file as guest user, even when an account was setup
//
// me@mybox:~/$ pastebin -a add main.go
//
//
package main

import (
	"flag"
	"fmt"
	"github.com/scusi/pastebin/client"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"os/user"
	"syscall"
)

var err error

var action string     // action to perform. Can be: add, del
var file string       // the file to be uploaded, just relevant in case of action=add
var sessionKey string // the 'api_user_key' to be used for a request to pastebin.com
var username string   // 'api_user_name' to be used for a q request to pastebin.com
var password string   // 'api_user_password', just used to login to pastebin.com during setup phase to get a 'api_user_key'
var devkey string     // your pastebin 'api_dev_key'
var expire string     // expire setting to be used for your posting
var clientFile string // file where the pastebin settings are stored
var anonymous bool    // if set to true a configured user account will NOT be used.
var debug bool        // debug output enabled if true

func init() {
	usr, err := user.Current()
	check(err)
	homedir := usr.HomeDir
	clientFile = homedir + "/" + ".pastebin"
	flag.StringVar(&sessionKey, "s", "", "sessionkey to use")
	flag.StringVar(&expire, "e", "10M", "expireation for paste, default: 10M [10M,1H,1D,1W,2W,1M,6M,1Y,N]")
	flag.StringVar(&clientFile, "c", homedir+"/.pastebin", "file to save client to")
	flag.BoolVar(&anonymous, "a", false, "anonymous flag, set to true for not useing a configured useraccount")
	flag.BoolVar(&debug, "d", false, "debug flag, set to true for debug output")
}

// check generic error checker function
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var pc *client.Client
	flag.Parse()
	log.Printf("expire set to: %s\n", expire)

	log.Printf("debug %t\n", debug)
	if debug {
		client.Debug = true
	}
	action := flag.Arg(0)
	log.Printf("action is: '%s'\n", action)
	// if the anonymous flag is set we will always use a new (unconfigured) client.
	if anonymous == true {
		log.Printf("anonymous flag set (true)")
		pc, err = client.New(
			client.SetExpire(expire),
		)
		check(err)
		log.Printf("Expire set to: %s\n", pc.Expire)
		goto SWITCH
	}

	if _, err := os.Stat(clientFile); !os.IsNotExist(err) {
		pc, err = client.RestoreClient(clientFile)
		check(err)
		//log.Printf("restored client from %s", clientFile)
		pc.Update(client.SetExpire(expire))
	} else {
		err = fmt.Errorf("client not configured, please setup first if you want to paste with your user account")
		log.Println(err)
		//action = "setup"
		pc, err = client.New()
		check(err)
		pc.Update(client.SetExpire(expire))
	}
SWITCH:
	switch action {
	case "add":
		file = flag.Arg(1)
		url, err := pc.NewPasteFromFile(file)
		check(err)
		fmt.Printf("%s\n", url)
	case "del":
		file = flag.Arg(1)
		url, err := pc.DeletePaste(file)
		check(err)
		fmt.Printf("%s\n", url)
	case "list":
		results, err := pc.ListPastes()
		check(err)
		fmt.Printf("%s\n", results)
	case "setup":
		if _, err := os.Stat(clientFile); !os.IsNotExist(err) {
			fmt.Print("Enter your pastebin username: ")
			fmt.Scanf("%s", &username)
			fmt.Printf("Enter password for the '%s'", username)
			bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
			check(err)
			fmt.Println("")
			password = string(bytePassword)
		}
		pc, err = client.New(
			client.SetUsername(username),
			client.SetPassword(password),
		)
		check(err)
		sessionKey, err = pc.Login()
		check(err)
		password = "" // delete password from memory after login
		// save the client
		err := client.SaveClient(pc, clientFile)
		check(err)
		log.Printf("client saved under '%s'\n", clientFile)
	}
}
