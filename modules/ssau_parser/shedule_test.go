package ssau_parser

import (
	"log"
	"testing"
)

func TestDownload(t *testing.T) {
	HeadURL = "http://127.0.0.1:5000"
	sh := WeekShedule{}
	err := sh.Download("/rasp?groupId=802440189", 4, false)
	handleError(err)
	// Ошибка в адресе
	err = sh.Download("/oops", 4, false)
	handleError(err)
	// Ошибка во время парсинга
	err = sh.Download("/rasp?groupId=123", 3, false)
	handleError(err)

	// Тестирование DownloadById с отсутствующими полями
	sh = WeekShedule{
		IsGroup: false,
		Week:    4,
	}
	err = sh.DownloadById(true)
	handleError(err)

	sh = WeekShedule{
		SheduleId: 802440189,
		IsGroup:   false,
	}
	err = sh.DownloadById(true)
	handleError(err)
}

func TestSheduleCompare(t *testing.T) {
	HeadURL = "http://127.0.0.1:5000"
	sh := WeekShedule{
		SheduleId: 802440189,
		IsGroup:   true,
		Week:      4,
	}
	err := sh.DownloadById(true)
	handleError(err)

	new_sh := WeekShedule{
		SheduleId: 802440189,
		IsGroup:   true,
		Week:      8,
	}
	err = new_sh.DownloadById(true)
	handleError(err)

	add, del := Compare(new_sh.Uncovered, sh.Uncovered)
	log.Println(add, del)
}
