package database

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
	xlog "xorm.io/xorm/log"
	"xorm.io/xorm/names"
)

type DB struct {
	User   string
	Pass   string
	Schema string
}

func Connect(db DB) (*xorm.Engine, error) {
	engine, err := xorm.NewEngine("mysql", fmt.Sprintf("%s:%s@tcp(localhost:3306)/%s?charset=utf8", db.User, db.Pass, db.Schema))
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
			log.Fatal("failed create log folder")
		}
	}
	fileName := "./logs/" + name + ".log"
	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		new := fmt.Sprintf("./logs/%s.before.%s.log", name, time.Now().Format("06-02-01-15-04-05"))
		err := os.Rename(fileName, new)
		if err != nil {
			log.Fatalf("failed rename %s file to %s", fileName, new)
		}
	}
	logFile, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("failed open %s.log file", name)
	}
	defer logFile.Close()
	return logFile
}
