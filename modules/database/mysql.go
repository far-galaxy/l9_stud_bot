package database

import (
	"log"
	"math/rand"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
	xlog "xorm.io/xorm/log"
	"xorm.io/xorm/names"
)

func Connect(user, pass, db string) (*xorm.Engine, error) {
	engine, err := xorm.NewEngine("mysql", user+":"+pass+"@tcp(localhost:3306)/"+db+"?charset=utf8")
	if err != nil {
		return nil, err
	}
	sqlLogger := xlog.NewSimpleLogger(CreateLog("sql"))
	engine.SetLogger(sqlLogger)
	engine.ShowSQL(true)
	engine.SetMapper(names.SameMapper{})

	err = engine.Sync(&User{}, &TgUser{}, &Group{}, &Lesson{}, &Teacher{}, &ShedulesInUser{})
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func GenerateID(engine *xorm.Engine) (int64, error) {
	id := rand.Int63n(899999999) + 100000000

	exists, err := engine.ID(id).Exist(&User{})
	if err != nil {
		return 0, err
	}

	if exists {
		return GenerateID(engine)
	} else {
		return id, nil
	}
}

func CreateLog(name string) *os.File {
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		err = os.Mkdir("logs", os.ModePerm)
		if err != nil {
			log.Fatal("Fail to create log folder")
		}
	}
	fileName := "./logs/" + name + ".log"
	logFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal("Fail to open tg.log file")
	}
	return logFile
}
