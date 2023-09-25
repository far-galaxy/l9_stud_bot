package ssauparser

import (
	"log"
	"testing"
)

func TestDownload(t *testing.T) {
	HeadURL = "http://127.0.0.1:5000"
	sh := WeekShedule{}
	err := sh.Download("/rasp?groupId=100000000", 1, false)
	handleError(err)
	// Ошибка в адресе
	err = sh.Download("/oops", 4, false)
	handleError(err)
	// Ошибка во время парсинга
	err = sh.Download("/rasp?groupId=123456789", 1, false)
	handleError(err)

	// Тестирование DownloadById с отсутствующими полями
	sh = WeekShedule{
		IsGroup: false,
		Week:    4,
	}
	err = sh.DownloadByID(true)
	handleError(err)

	sh = WeekShedule{
		SheduleID: 123456789,
		IsGroup:   false,
	}
	err = sh.DownloadByID(true)
	handleError(err)
	t.Log("ok")
}

func TestSheduleCompare(t *testing.T) {
	HeadURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleID: 123456789,
		IsGroup:   true,
		Week:      6,
	}
	err := sh.DownloadByID(true)
	handleError(err)

	newSh := WeekShedule{
		SheduleID: 123456789,
		IsGroup:   true,
		Week:      7,
	}
	err = newSh.DownloadByID(true)
	handleError(err)

	add, del := Compare(newSh.Uncovered, sh.Uncovered)
	log.Println(add, del)
	t.Log("ok")
}
