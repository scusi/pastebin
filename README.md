# pastebin

a pastebin commandline tool, can paste files onto pastebin.

supports pastebin.com user accounts. when configured with credentials to pastebin.com
it can list your pastes, add new ones and delete existing ones.

[![GoDoc](https://godoc.org/github.com/scusi/pastebin?status.svg)](https://godoc.org/github.com/scusi/pastebin)

## Install

```go get github.com/scusi/pastebin```

## Configure

```pastebin setup```

configuration file will be saved under $HOME/.pastebin
**Your password will not be saved to disk at any time.**
The supplied password is just used once to login to pastebin.com and retrieve a valid _api_user_key_.
The _api_user_key_ is saved to the config file along with the supplied username.

After setup your account is used for all further actions.

## Usage

```
pastebin [-a -d -c /path/to/file -e expireValue] action arg

-a anonymous flag, post as guest if true
-d debug flag, output debug info if true
-c file to read/write client configuration to
-e expire value, can be 10M,1H,1D,1W,2W,1M,6M,1Y,N

action can be: add, list, del
arg us the filename to add or the code to delete
```

Paste a file 

```pastebin add test.txt```

If you have configured your client 'test.txt' will be posted as the configured user.
If you have not configured your client beforehand you post as a guest.

You can also post as a guest after you have configured your client by useing the -a switch.

```pastebin -a add test.txt```

The above command would post as a guest, even when your client is confirued to use a user account.

sessionkey to use
```
