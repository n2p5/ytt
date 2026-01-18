package youtube

import (
	"fmt"
	"os"
	"strings"
)

// VideoInfo represents metadata for a YouTube video.
type VideoInfo struct {
	VideoID   string `json:"video_id"`
	Title     string `json:"title"`
	ViewCount uint64 `json:"view_count"`
	Date      string `json:"published_at"`
}

// VideoDetails represents detailed metadata for a YouTube video.
type VideoDetails struct {
	VideoID      string   `json:"video_id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	ChannelID    string   `json:"channel_id"`
	ChannelTitle string   `json:"channel_title"`
	Duration     string   `json:"duration"`
	ViewCount    uint64   `json:"view_count"`
	LikeCount    uint64   `json:"like_count"`
	CommentCount uint64   `json:"comment_count"`
	PublishedAt  string   `json:"published_at"`
	Tags         []string `json:"tags,omitempty"`
}

// ListVideos retrieves all videos from a channel, filtering out shorts.
func (c *Client) ListVideos(channelID string, minDurationSeconds int) ([]VideoInfo, error) {
	debug := os.Getenv("YTT_DEBUG") != ""

	if channelID == "" {
		var err error
		channelID, err = c.getAuthenticatedChannelID()
		if err != nil {
			return nil, err
		}
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Channel ID: %s\n", channelID)
	}

	uploadsPlaylistID, err := c.getUploadsPlaylistID(channelID)
	if err != nil {
		return nil, err
	}
	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Uploads playlist ID: %s\n", uploadsPlaylistID)
	}

	videos := []VideoInfo{}
	nextPageToken := ""

	for {
		playlistCall := c.Service.PlaylistItems.List([]string{"snippet"}).
			PlaylistId(uploadsPlaylistID).
			MaxResults(50)
		if nextPageToken != "" {
			playlistCall = playlistCall.PageToken(nextPageToken)
		}

		playlistResponse, err := playlistCall.Do()
		if err != nil {
			return nil, fmt.Errorf("error retrieving playlist items: %w", err)
		}
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Playlist returned %d items\n", len(playlistResponse.Items))
		}

		var videoIDs []string
		for _, item := range playlistResponse.Items {
			videoIDs = append(videoIDs, item.Snippet.ResourceId.VideoId)
		}

		if len(videoIDs) > 0 {
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Fetching details for video IDs: %v\n", videoIDs)
			}
			videosCall := c.Service.Videos.List([]string{"snippet", "statistics", "contentDetails"}).
				Id(strings.Join(videoIDs, ","))
			videosResponse, err := videosCall.Do()
			if err != nil {
				return nil, fmt.Errorf("error retrieving video statistics: %w", err)
			}
			if debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] Videos.List returned %d items\n", len(videosResponse.Items))
			}

			for _, video := range videosResponse.Items {
				duration := ParseDuration(video.ContentDetails.Duration)
				if debug {
					fmt.Fprintf(os.Stderr, "[DEBUG] Video %s: duration=%s (%ds), minDuration=%d, isShort=%v\n",
						video.Id, video.ContentDetails.Duration, duration, minDurationSeconds, duration < minDurationSeconds)
				}
				if isShort(video.ContentDetails.Duration, minDurationSeconds) {
					continue
				}
				videos = append(videos, VideoInfo{
					VideoID:   video.Id,
					Title:     video.Snippet.Title,
					ViewCount: video.Statistics.ViewCount,
					Date:      video.Snippet.PublishedAt,
				})
			}
		}

		nextPageToken = playlistResponse.NextPageToken
		if nextPageToken == "" {
			break
		}
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Returning %d videos after filtering\n", len(videos))
	}
	return videos, nil
}

// GetVideoDetails retrieves detailed metadata for a single video.
func (c *Client) GetVideoDetails(videoID string) (*VideoDetails, error) {
	call := c.Service.Videos.List([]string{"snippet", "statistics", "contentDetails"}).Id(videoID)
	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("error retrieving video details: %w", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("video %s not found", videoID)
	}

	video := response.Items[0]
	return &VideoDetails{
		VideoID:      video.Id,
		Title:        video.Snippet.Title,
		Description:  video.Snippet.Description,
		ChannelID:    video.Snippet.ChannelId,
		ChannelTitle: video.Snippet.ChannelTitle,
		Duration:     video.ContentDetails.Duration,
		ViewCount:    video.Statistics.ViewCount,
		LikeCount:    video.Statistics.LikeCount,
		CommentCount: video.Statistics.CommentCount,
		PublishedAt:  video.Snippet.PublishedAt,
		Tags:         video.Snippet.Tags,
	}, nil
}

func (c *Client) getAuthenticatedChannelID() (string, error) {
	channelsCall := c.Service.Channels.List([]string{"id", "statistics"}).Mine(true)
	channelsResponse, err := channelsCall.Do()
	if err != nil {
		return "", fmt.Errorf("error retrieving user's channel: %w", err)
	}
	if len(channelsResponse.Items) == 0 {
		return "", fmt.Errorf("no channel found for authenticated user")
	}
	if os.Getenv("YTT_DEBUG") != "" {
		stats := channelsResponse.Items[0].Statistics
		fmt.Fprintf(os.Stderr, "[DEBUG] Channel stats: %d videos, %d subscribers\n",
			stats.VideoCount, stats.SubscriberCount)
	}
	return channelsResponse.Items[0].Id, nil
}

func (c *Client) getUploadsPlaylistID(channelID string) (string, error) {
	channelCall := c.Service.Channels.List([]string{"contentDetails"}).Id(channelID)
	channelResponse, err := channelCall.Do()
	if err != nil {
		return "", fmt.Errorf("error retrieving channel details: %w", err)
	}
	if len(channelResponse.Items) == 0 {
		return "", fmt.Errorf("channel %s not found", channelID)
	}
	return channelResponse.Items[0].ContentDetails.RelatedPlaylists.Uploads, nil
}

func isShort(duration string, minDurationSeconds int) bool {
	return ParseDuration(duration) < minDurationSeconds
}

// ParseDuration parses ISO 8601 duration and returns total seconds.
func ParseDuration(duration string) int {
	if !strings.HasPrefix(duration, "PT") {
		return 0
	}
	d := duration[2:]

	var hours, minutes, seconds int
	for len(d) > 0 {
		i := 0
		for i < len(d) && d[i] >= '0' && d[i] <= '9' {
			i++
		}
		if i == 0 || i >= len(d) {
			break
		}
		val := 0
		for j := 0; j < i; j++ {
			val = val*10 + int(d[j]-'0')
		}
		switch d[i] {
		case 'H':
			hours = val
		case 'M':
			minutes = val
		case 'S':
			seconds = val
		}
		d = d[i+1:]
	}

	return hours*3600 + minutes*60 + seconds
}
