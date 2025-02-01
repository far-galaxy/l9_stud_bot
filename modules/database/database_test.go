package database

import (
	"log"
	"testing"
	"time"
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
	mainLog := InitLog("log", time.Hour*24*14)
	log.SetOutput(mainLog)
	log.Println("testing")
	t.Log("ok")
}

func TestConnect(t *testing.T) {
	_, err := Connect(TestDB, InitLog("sql", time.Hour*24*14))
	handleError(err)
	_, err = Connect(TestDB, InitLog("sql", time.Hour*24*14))
	handleError(err)
	t.Log("ok")
}
