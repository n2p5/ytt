package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Client wraps the YouTube API service.
type Client struct {
	Service *youtube.Service
}

// NewClient creates a new YouTube API client using OAuth2 credentials.
func NewClient(oauthPath, tokenPath string) (*Client, error) {
	ctx := context.Background()

	b, err := os.ReadFile(oauthPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %w", err)
	}

	config, err := google.ConfigFromJSON(b, youtube.YoutubeReadonlyScope, youtube.YoutubeForceSslScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file: %w", err)
	}

	httpClient, err := getHTTPClient(config, tokenPath)
	if err != nil {
		return nil, err
	}

	service, err := youtube.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("unable to create YouTube service: %w", err)
	}

	return &Client{Service: service}, nil
}

// Authenticate forces a new OAuth flow and saves the token.
func Authenticate(oauthPath, tokenPath string) error {
	b, err := os.ReadFile(oauthPath)
	if err != nil {
		return fmt.Errorf("unable to read client secret file: %w", err)
	}

	config, err := google.ConfigFromJSON(b, youtube.YoutubeReadonlyScope, youtube.YoutubeForceSslScope)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file: %w", err)
	}

	tok, err := getTokenFromWeb(config)
	if err != nil {
		return err
	}

	return saveToken(tokenPath, tok)
}

func getHTTPClient(config *oauth2.Config, tokenPath string) (*http.Client, error) {
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenPath, tok); err != nil {
			return nil, err
		}
	} else if tok.Expiry.Before(time.Now()) {
		ctx := context.Background()
		tokenSource := config.TokenSource(ctx, tok)
		newTok, err := tokenSource.Token()
		if err != nil {
			log.Printf("Token refresh failed: %v", err)
			log.Println("Re-authenticating...")
			tok, err = getTokenFromWeb(config)
			if err != nil {
				return nil, err
			}
			if err := saveToken(tokenPath, tok); err != nil {
				return nil, err
			}
		} else if newTok.AccessToken != tok.AccessToken {
			tok = newTok
			if err := saveToken(tokenPath, tok); err != nil {
				return nil, err
			}
		}
	}
	return config.Client(context.Background(), tok), nil
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	codeChan := make(chan string)

	server := &http.Server{Addr: ":8080"}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintf(w, "Authorization successful! You can close this tab.")
			codeChan <- code
		} else {
			fmt.Fprintf(w, "Authorization failed: no code received")
			codeChan <- ""
		}
	})
	server.Handler = mux

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	config.RedirectURL = "http://localhost:8080"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Fprintf(os.Stderr, "Opening browser for authorization...\nIf it doesn't open automatically, go to: %v\n", authURL)

	go func() {
		time.Sleep(1 * time.Second)
		fmt.Fprintf(os.Stderr, "Browser should open automatically. If not, please visit the URL above.\n")
	}()

	authCode := <-codeChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	if authCode == "" {
		return nil, fmt.Errorf("authorization failed")
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

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

func saveToken(path string, token *oauth2.Token) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("unable to create token directory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
