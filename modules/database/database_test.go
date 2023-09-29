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

func TestInitLog(t *testing.T) {
	mainLog := InitLog("log")
	log.SetOutput(mainLog)
	log.Println("testing")
	t.Log("ok")
}

func TestConnect(t *testing.T) {
	_, err := Connect(TestDB, InitLog("sql"))
	handleError(err)
	_, err = Connect(TestDB, InitLog("sql"))
	handleError(err)
	t.Log("ok")
}
