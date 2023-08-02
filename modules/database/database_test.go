package database

import (
	"log"
	"testing"
)

var TestDB = DB{
	User:   "test",
	Pass:   "TESTpass1!",
	Schema: "testdb",
}

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
	_, err := Connect(TestDB)
	handleError(err)
	_, err = Connect(TestDB)
	handleError(err)
}
