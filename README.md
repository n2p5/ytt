# ytt

This is meant to be a placeholder for code I've generated with Claude Code to solve some problems I have when needing to interact with videos. 

Initially this just helps me get transcripts from youtube videos on channels I manage. You give it your video ID and it will put the transcript in the outputs folder. 

This does require setting up oauth to get it to work.

## Features

- **Transcript Extraction**: Download transcripts from YouTube videos you own
- **OAuth Authentication**: Secure authentication with YouTube API
- **Flexible Output**: Save transcripts to custom directories with meaningful filenames
- **Automatic File Naming**: Files saved as `{video_id}-{title}.txt`

## Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/n2p5/ytt
   cd ytt
   ```

2. **Install dependencies**
   ```bash
   go mod tidy
   ```

3. **Setup Google OAuth**
   - Create a Google Cloud Project and enable YouTube Data API v3
   - Create OAuth 2.0 credentials (Desktop application)
   - Add `http://localhost:8080` as an authorized redirect URI
   - Download the client secret JSON file
   - Place it in the `secrets/` directory (any filename ending in `.json`)

## Usage

### Extract transcript to default `outputs/` directory:
```bash
go run main.go <video_id>
```

### Extract transcript to custom directory:
```bash
go run main.go -output my_transcripts <video_id>
```

### Examples:
```bash
# Extract transcript for video abc123 to outputs/
go run main.go abc123

# Extract to custom directory
go run main.go -output ~/Downloads abc123

# Results in files like: abc123-My_Video_Title.txt
```

### Edge Cases

For video IDs that start with a dash (e.g., `-m8CDR_lHXo`), use `--` to separate flags from arguments:
```bash
go run main.go -- -m8CDR_lHXo
```

This tells the command parser that everything after `--` should be treated as arguments, not flags.

## First Run

On first run, the tool will:
1. Open your browser for OAuth authentication
2. Redirect to localhost:8080 (handled automatically)
3. Save authentication token for future use
4. Download and save the transcript

## Requirements

- Go 1.24.5+
- Google Cloud Project with YouTube Data API v3 enabled
- OAuth 2.0 credentials configured for desktop application
- Video must be owned by the authenticated account
