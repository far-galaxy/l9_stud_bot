package ssau_parser

import (
	"log"
	"testing"
)

func TestParse(t *testing.T) {
	headURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleId: 802440189,
		IsGroup:   true,
		Week:      3,
	}
	err := sh.DownloadById(false)
	handleError(err)

	// Ошибки в скелете расписания
	for i := 1; i < 6; i++ {
		sh := WeekShedule{
			SheduleId: 123,
			IsGroup:   true,
			Week:      i,
		}
		err = sh.DownloadById(false)
		handleError(err)
	}

	// Ошибки внутри пар
	for i := 2; i < 3; i++ {
		sh := WeekShedule{
			SheduleId: 62806001,
			IsGroup:   false,
			Week:      i,
		}
		err = sh.DownloadById(false)
		handleError(err)
		log.Println(sh.FullName)
	}
}