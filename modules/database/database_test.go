package database

import (
	"log"
	"testing"
)

// Вывод некритических ошибок тестирования в консоль
func handleError(err error) {
	if err != nil {
		log.Println(err)
	}
}

func TestCreateLog(t *testing.T) {
	CreateLog("log")
}

func TestConnect(t *testing.T) {
	_, err := Connect("test", "TESTpass1!", "testdb")
	handleError(err)
	_, err = Connect("test", "wrongpass", "testdb")
	handleError(err)
}
