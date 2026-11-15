package xapi

import (
	"math"
	"testing"

	"github.com/chromedp/cdproto/cdp"
)

func TestFormatWallTime(t *testing.T) {
	tests := []struct {
		name string
		in   cdp.TimeSinceEpoch
		want string
	}{
		{name: "unset", in: 0, want: ""},
		{name: "negative", in: -1, want: ""},
		{name: "nan", in: cdp.TimeSinceEpoch(math.NaN()), want: ""},
		{name: "whole seconds", in: 1700000000, want: "2023-11-14T22:13:20Z"},
		{name: "fractional seconds", in: 1700000000.5, want: "2023-11-14T22:13:20.5Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatWallTime(tt.in); got != tt.want {
				t.Errorf("formatWallTime(%v) = %q, want %q", float64(tt.in), got, tt.want)
			}
		})
	}
}
