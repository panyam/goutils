package utils

import (
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	DATE_FORMAT     = "2006-01-02"
	DATETIME_FORMAT = "2006-01-02 03-04-05"
)

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

func ParseDate(str string) time.Time {
	str = strings.Replace(str, "_", "-", -1)
	if result, err := time.Parse(DATE_FORMAT, str); err != nil {
		log.Print("Error: ", err)
		log.Fatalf("Invalid date string: %s", str)
	} else {
		return result
	}
	return time.Now().UTC()
}

func ParseTime(str string) time.Time {
	str = strings.Replace(str, "_", "-", -1)
	if result, err := time.Parse(DATETIME_FORMAT, str); err != nil {
		log.Fatalf("Invalid time string: %s", str)
	} else {
		return result
	}
	return time.Now().UTC()
}

func FormatDate(t time.Time) string {
	return fmt.Sprintf("%04d_%02d_%02d", t.Year(), t.Month(), t.Day())
}

func FormatTime(t time.Time) string {
	return fmt.Sprintf("%04d_%02d_%02d %02d-%02d-%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
