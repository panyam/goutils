package utils

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// Our Default Date and DateTime formats
const (
	DEFAULT_DATE_FORMAT     = "2006-01-02"
	DEFAULT_DATETIME_FORMAT = "2006-01-02 03-04-05"
)

// A quick way to check if
func NeedsRefresh(refresh_type int32, last_refreshed_at time.Time, now time.Time) bool {
	if refresh_type <= 0 {
		return true
	}
	diff := now.Sub(last_refreshed_at)
	td := time.Duration(int64(refresh_type) * 1000000000)
	needsit := diff > td
	if needsit {
		log.Println("Diff: ", diff, "RT: ", refresh_type, "TD: ", td, "LastRef: ", last_refreshed_at, "Now: ", now)
	}
	return needsit
}

func DefaultParseDate(str string) time.Time {
	str = strings.Replace(str, "_", "-", -1)
	if result, err := time.Parse(DEFAULT_DATE_FORMAT, str); err != nil {
		log.Print("Error: ", err)
		log.Fatalf("Invalid date string: %s", str)
	} else {
		return result
	}
	return time.Now().UTC()
}

func DefaultParseTime(str string) time.Time {
	str = strings.Replace(str, "_", "-", -1)
	if result, err := time.Parse(DEFAULT_DATETIME_FORMAT, str); err != nil {
		log.Fatalf("Invalid time string: %s", str)
	} else {
		return result
	}
	return time.Now().UTC()
}

func DefaultFormatDate(t time.Time) string {
	return fmt.Sprintf("%04d_%02d_%02d", t.Year(), t.Month(), t.Day())
}

func DefaultFormatTime(t time.Time) string {
	return fmt.Sprintf("%04d_%02d_%02d %02d-%02d-%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
