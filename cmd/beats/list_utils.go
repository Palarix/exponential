package main

import (
	"time"

	"github.com/dustin/go-humanize"
)

var listMagnitudes = []humanize.RelTimeMagnitude{
	{D: time.Second, Format: "now", DivBy: time.Second},
	{D: 2 * time.Second, Format: "1s %s", DivBy: 1},
	{D: time.Minute, Format: "%ds %s", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "1m %s", DivBy: 1},
	{D: time.Hour, Format: "%dm %s", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "1h %s", DivBy: 1},
	{D: 24 * time.Hour, Format: "%dh %s", DivBy: time.Hour},
	{D: 2 * 24 * time.Hour, Format: "1d %s", DivBy: 1},
	{D: 7 * 24 * time.Hour, Format: "%dd %s", DivBy: 24 * time.Hour},
	{D: 2 * 7 * 24 * time.Hour, Format: "1w %s", DivBy: 1},
	{D: 30 * 24 * time.Hour, Format: "%dw %s", DivBy: 7 * 24 * time.Hour},
	{D: 2 * 30 * 24 * time.Hour, Format: "1mo %s", DivBy: 1},
	{D: 365 * 24 * time.Hour, Format: "%dmo %s", DivBy: 30 * 24 * time.Hour},
	{D: 2 * 365 * 24 * time.Hour, Format: "1y %s", DivBy: 1},
	{D: 100 * 365 * 24 * time.Hour, Format: "%dy %s", DivBy: 365 * 24 * time.Hour},
}
