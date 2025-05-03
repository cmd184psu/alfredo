package alfredo

import (
	"fmt"
	"testing"
	"time"
)

func TestCalculateETARaw(t *testing.T) {
	type args struct {
		remaining    int64
		rateOfChange float64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "Zero rate of change",
			args: args{
				remaining:    100,
				rateOfChange: 0,
			},
			want: 0,
		},
		{
			name: "Normal case",
			args: args{
				remaining:    100,
				rateOfChange: 10,
			},
			want: 10,
		},
		{
			name: "Rate of change greater than remaining",
			args: args{
				remaining:    5,
				rateOfChange: 10,
			},
			want: 1,
		},
		{
			name: "Remaining is zero",
			args: args{
				remaining:    0,
				rateOfChange: 10,
			},
			want: 0,
		},
		{
			name: "Rate of change is one",
			args: args{
				remaining:    100,
				rateOfChange: 1,
			},
			want: 100,
		},
		{
			name: "Measuring using same numbers as status_test",
			args: args{
				remaining:    6000000,
				rateOfChange: CalculateRateOfChange(3000000, time.Now().Unix()-3600, time.Now().Unix()), // Fixed timestamp for consistent testing
			},
			want: 7200,
		},
		{
			name: "Measuring using 50% in 1 hr",
			args: args{
				remaining:    1000000,
				rateOfChange: CalculateRateOfChange(1000000, time.Now().Unix()-3600, time.Now().Unix()), // Fixed timestamp for consistent testing
			},
			want: 3600,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateETARaw(tt.args.remaining, tt.args.rateOfChange)
			fmt.Printf("eta (raw): %d (hr: %s, projected: %s)\n", got, HumanReadableSeconds(got), SecondsToTimestamp(time.Now().Unix()+got))
			if got != tt.want {
				t.Errorf("CalculateETARaw() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateRateOfChange(t *testing.T) {
	type args struct {
		unitsProcessed int64
		startTime      int64
		endTime        int64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "Zero units processed",
			args: args{
				unitsProcessed: 0,
				startTime:      time.Now().Unix() - 10,
				endTime:        time.Now().Unix(),
			},
			want: 0,
		},
		{
			name: "Zero start time",
			args: args{
				unitsProcessed: 500,
				startTime:      0,
				endTime:        0,
			},
			want: 0,
		},
		{
			name: "Normal case",
			args: args{
				unitsProcessed: 500,
				startTime:      time.Now().Unix() - 10,
				endTime:        time.Now().Unix(),

			},
			want: 50, // Assuming 500 units processed in 10 seconds
		},
		{
			name: "Rate of change less than 1",
			args: args{
				unitsProcessed: 1,
				startTime:      time.Now().Unix() - 10, // Fixed timestamp for consistent testing
				endTime:        time.Now().Unix(),
			},
			want: 0.1, // Minimum rate of change is 1
		},
		{
			name: "Large units processed",
			args: args{
				unitsProcessed: 900000,
				startTime:      time.Now().Unix() - 100, // Fixed timestamp for consistent testing
				endTime:        time.Now().Unix(),
			},
			want: 9000, // Assuming 900000 units processed in 100 seconds
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateRateOfChange(tt.args.unitsProcessed, tt.args.startTime, tt.args.endTime); got != tt.want {
				t.Errorf("CalculateRateOfChange() = %v, want %v", got, tt.want)
			}
		})
	}
}
