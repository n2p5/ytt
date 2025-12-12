package youtube

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DownloadTranscript downloads the transcript for a video and saves it to the output directory.
func (c *Client) DownloadTranscript(videoID, outputDir string) error {
	videoCall := c.Service.Videos.List([]string{"snippet"}).Id(videoID)
	videoResponse, err := videoCall.Do()
	if err != nil {
		return fmt.Errorf("error retrieving video details: %w", err)
	}

	if len(videoResponse.Items) == 0 {
		return fmt.Errorf("video %s not found", videoID)
	}

	videoTitle := videoResponse.Items[0].Snippet.Title
	sanitizedTitle := SanitizeFilename(videoTitle)

	captionsCall := c.Service.Captions.List([]string{"snippet"}, videoID)
	captionsResponse, err := captionsCall.Do()
	if err != nil {
		return fmt.Errorf("error retrieving captions list: %w", err)
	}

	if len(captionsResponse.Items) == 0 {
		return fmt.Errorf("no captions found for video %s", videoID)
	}

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

	downloadCall := c.Service.Captions.Download(captionID)
	resp, err := downloadCall.Download()
	if err != nil {
		return fmt.Errorf("error downloading captions: %w", err)
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.txt", videoID, sanitizedTitle)
	outputPath := filepath.Join(outputDir, filename)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outputFile.Close()

	fmt.Fprintf(os.Stderr, "Downloading transcript for video: %s\n", videoTitle)
	fmt.Fprintf(os.Stderr, "Saving to: %s\n", outputPath)

	if _, err := io.Copy(outputFile, resp.Body); err != nil {
		return fmt.Errorf("error writing transcript: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Transcript saved successfully!\n")
	return nil
}

// SanitizeFilename removes or replaces characters that are invalid in filenames.
func SanitizeFilename(filename string) string {
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := reg.ReplaceAllString(filename, "_")

	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}

	sanitized = strings.Trim(sanitized, " .")

	return sanitized
}
