package parser

import (
	"log"
	"testing"
)

func TestParse(t *testing.T) {
	HeadURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleID: 111111111,
		IsGroup:   true,
		Week:      1,
	}
	err := sh.DownloadByID(false)
	handleError(err)

	// Ошибки в скелете расписания
	for i := 1; i < 5; i++ {
		sh := WeekShedule{
			SheduleID: 123456789,
			IsGroup:   true,
			Week:      i,
		}
		err = sh.DownloadByID(false)
		handleError(err)
	}

	// Ошибки внутри пар
	for i := 2; i < 3; i++ {
		sh := WeekShedule{
			SheduleID: 5,
			IsGroup:   false,
			Week:      i,
		}
		err = sh.DownloadByID(false)
		handleError(err)
		log.Println(sh.FullName)
	}
	t.Log("ok")
}
