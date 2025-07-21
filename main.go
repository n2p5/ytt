package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func main() {
	var outputDir string
	flag.StringVar(&outputDir, "output", "outputs", "Output directory for transcript files")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("Usage: go run main.go [-output <dir>] <video_id>")
	}

	videoID := args[0]

	ctx := context.Background()

	// Read the client secret file
	b, err := os.ReadFile("secrets/oauth.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// Parse the credentials
	config, err := google.ConfigFromJSON(b, youtube.YoutubeReadonlyScope, youtube.YoutubeForceSslScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// Get the OAuth2 client
	client := getClient(config)

	// Create YouTube service
	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create YouTube service: %v", err)
	}

	// Create output directory if it doesn't exist
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	// Get video captions
	err = getTranscript(service, videoID, outputDir)
	if err != nil {
		log.Fatalf("Error getting transcript: %v", err)
	}
}

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "secrets/token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	// Create a channel to receive the authorization code
	codeChan := make(chan string)
	
	// Start a local server to handle the OAuth callback
	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintf(w, "Authorization successful! You can close this tab.")
			codeChan <- code
		} else {
			fmt.Fprintf(w, "Authorization failed: no code received")
			codeChan <- ""
		}
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Update redirect URI to use localhost:8080
	config.RedirectURL = "http://localhost:8080"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Opening browser for authorization...\nIf it doesn't open automatically, go to: %v\n", authURL)

	// Try to open the browser automatically
	go func() {
		time.Sleep(1 * time.Second)
		fmt.Printf("Browser should open automatically. If not, please visit the URL above.\n")
	}()

	// Wait for the authorization code
	authCode := <-codeChan
	
	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	if authCode == "" {
		log.Fatal("Authorization failed")
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
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

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getTranscript(service *youtube.Service, videoID, outputDir string) error {
	// First, get video details to fetch the title
	videoCall := service.Videos.List([]string{"snippet"}).Id(videoID)
	videoResponse, err := videoCall.Do()
	if err != nil {
		return fmt.Errorf("error retrieving video details: %v", err)
	}

	if len(videoResponse.Items) == 0 {
		return fmt.Errorf("video %s not found", videoID)
	}

	videoTitle := videoResponse.Items[0].Snippet.Title
	sanitizedTitle := sanitizeFilename(videoTitle)
	
	// Get the list of caption tracks for the video
	captionsCall := service.Captions.List([]string{"snippet"}, videoID)
	captionsResponse, err := captionsCall.Do()
	if err != nil {
		return fmt.Errorf("error retrieving captions list: %v", err)
	}

	if len(captionsResponse.Items) == 0 {
		return fmt.Errorf("no captions found for video %s", videoID)
	}

	// Find the first available caption track (preferably auto-generated or English)
	var captionID string
	for _, caption := range captionsResponse.Items {
		if caption.Snippet.Language == "en" || caption.Snippet.Language == "" {
			captionID = caption.Id
			break
		}
	}

	if captionID == "" {
		captionID = captionsResponse.Items[0].Id
	}

	// Download the caption track
	downloadCall := service.Captions.Download(captionID)
	resp, err := downloadCall.Download()
	if err != nil {
		return fmt.Errorf("error downloading captions: %v", err)
	}
	defer resp.Body.Close()

	// Create output filename
	filename := fmt.Sprintf("%s-%s.txt", videoID, sanitizedTitle)
	outputPath := filepath.Join(outputDir, filename)

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outputFile.Close()

	// Read and write the transcript to file
	buf := make([]byte, 1024)
	fmt.Printf("Downloading transcript for video: %s\n", videoTitle)
	fmt.Printf("Saving to: %s\n", outputPath)
	
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			outputFile.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	fmt.Printf("Transcript saved successfully!\n")
	return nil
}

func sanitizeFilename(filename string) string {
	// Remove or replace characters that are invalid in filenames
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := reg.ReplaceAllString(filename, "_")
	
	// Limit length to avoid filesystem issues
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	
	// Remove leading/trailing spaces and dots
	sanitized = strings.Trim(sanitized, " .")
	
	return sanitized
}
