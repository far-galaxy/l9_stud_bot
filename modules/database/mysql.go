package database

import (
	"log"
	"math/rand"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

func Connect(user, pass, db string) *xorm.Engine {
	engine, err := xorm.NewEngine("mysql", user+":"+pass+"@tcp(localhost:3306)/"+db+"?charset=utf8")
	if err != nil {
		log.Fatal(err)
	}

	engine.ShowSQL(true)
	engine.SetMapper(names.SameMapper{})

	err = engine.Sync(&User{}, &TgUser{}, &Group{}, &Lesson{})
	if err != nil {
		log.Fatal(err)
	}
	return engine
}

func GenerateID(engine *xorm.Engine) int64 {
	id := rand.Int63n(899999999) + 100000000

	exists, err := engine.ID(id).Exist(&User{})
	if err != nil {
		log.Fatal(err)
	}

	if exists {
		return GenerateID(engine)
	} else {
		return id
	}
}
