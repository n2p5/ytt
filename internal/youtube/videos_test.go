package youtube

import "testing"

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     int
	}{
		{"seconds only", "PT30S", 30},
		{"minutes only", "PT5M", 300},
		{"hours only", "PT2H", 7200},
		{"minutes and seconds", "PT10M30S", 630},
		{"hours and minutes", "PT1H30M", 5400},
		{"hours minutes seconds", "PT1H30M45S", 5445},
		{"zero duration", "PT0S", 0},
		{"empty string", "", 0},
		{"missing PT prefix", "1H30M", 0},
		{"invalid format", "P1D", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDuration(tt.duration)
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %d, want %d", tt.duration, got, tt.want)
			}
		})
	}
}

func TestIsShort(t *testing.T) {
	tests := []struct {
		name               string
		duration           string
		minDurationSeconds int
		want               bool
	}{
		{"under threshold", "PT30S", 60, true},
		{"at threshold", "PT60S", 60, false},
		{"over threshold", "PT2M", 60, false},
		{"zero duration", "PT0S", 60, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isShort(tt.duration, tt.minDurationSeconds)
			if got != tt.want {
				t.Errorf("isShort(%q, %d) = %v, want %v", tt.duration, tt.minDurationSeconds, got, tt.want)
			}
		})
	}
}
