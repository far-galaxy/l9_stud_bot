package database

import (
	"math/rand"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

func Connect(user, pass, db string) (*xorm.Engine, error) {
	engine, err := xorm.NewEngine("mysql", user+":"+pass+"@tcp(localhost:3306)/"+db+"?charset=utf8")
	if err != nil {
		return nil, err
	}

	engine.ShowSQL(true)
	engine.SetMapper(names.SameMapper{})

	err = engine.Sync(&User{}, &TgUser{}, &Group{}, &Lesson{})
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
