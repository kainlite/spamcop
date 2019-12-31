package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func randStr(strSize int, randType string) string {

	var dictionary string

	if randType == "alphanum" {
		dictionary = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}

	if randType == "alpha" {
		dictionary = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}

	if randType == "number" {
		dictionary = "0123456789"
	}

	var bytes = make([]byte, strSize)
	rand.Read(bytes)
	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}
	return string(bytes)
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailComposeScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	pageToken := ""
	for {
		req := srv.Users.Messages.List("me").Q("in:spam")
		if pageToken != "" {
			req.PageToken(pageToken)
		}
		r, err := req.Do()
		if err != nil {
			log.Fatalf("Unable to retrieve messages: %v", err)
		}

		log.Printf("Processing %v messages...\n", len(r.Messages))
		for _, m := range r.Messages {
			// msg, err := srv.Users.Messages.Get("me", m.Id).Do()
			msg, err := srv.Users.Messages.Get("me", m.Id).Format("raw").Do()
			if err != nil {
				log.Fatalf("Unable to retrieve message %v: %v", m.Id, err)
			}

			// New message for our gmail service to send
			var message gmail.Message
			boundary := randStr(32, "alphanum")
			// It needs to be decoded from URL encoding otherwise strage things can happen
			body, err := base64.URLEncoding.DecodeString(msg.Raw)

			messageBody := []byte("Content-Type: multipart/mixed; boundary=" + boundary + " \n" +
				"MIME-Version: 1.0\n" +
				// "To: " + "kainlite@gmail.com" + "\n" +
				"To: " + "lala@spam.spamcop.net" + "\n" +
				"From: " + "kainlite@gmail.com" + "\n" +
				"Subject: " + "Spam report" + "\n\n" +

				"--" + boundary + "\n" +
				"Content-Type: text/plain; charset=" + string('"') + "UTF-8" + string('"') + "\n" +
				"MIME-Version: 1.0\n" +
				"Content-Transfer-Encoding: 7bit\n\n" +
				"Spam report" + "\n\n" +
				"--" + boundary + "\n" +

				"Content-Type: " + "message/rfc822" + "; name=" + string('"') + "email.txt" + string('"') + " \n" +
				"MIME-Version: 1.0\n" +
				"Content-Transfer-Encoding: base64\n" +
				"Content-Disposition: attachment; filename=" + string('"') + "email.txt" + string('"') + " \n\n" +
				string(body) +
				"--" + boundary + "--")

			// see https://godoc.org/google.golang.org/api/gmail/v1#Message on .Raw
			// use URLEncoding here !! StdEncoding will be rejected by Google API

			message.Raw = base64.URLEncoding.EncodeToString(messageBody)

			// Send the message
			_, err = srv.Users.Messages.Send("me", &message).Do()

			if err != nil {
				log.Printf("Error: %v", err)
			} else {
				fmt.Println("Message sent!")

				// If everything went well until here, then delete the message
				if err := srv.Users.Messages.Delete("me", m.Id).Do(); err != nil {
					log.Fatalf("unable to delete message %v: %v", m.Id, err)
				}
				log.Printf("Deleted message %v.\n", m.Id)
			}
		}

		if r.NextPageToken == "" {
			break
		}
		pageToken = r.NextPageToken
	}
}
