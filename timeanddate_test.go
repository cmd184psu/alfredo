package alfredo

import (
	"testing"
	"time"
)

func TestGetFirstOfMonthTimestamp(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "First day of the current month",
			want: time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02T15:04:05.000Z"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFirstOfMonthTimestamp(); got != tt.want {
				t.Errorf("GetFirstOfMonthTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}
