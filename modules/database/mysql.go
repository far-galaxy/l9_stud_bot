package database

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql" // Иначе не работает
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"xorm.io/xorm"
	xlog "xorm.io/xorm/log"
	"xorm.io/xorm/names"
)

type DB struct {
	User   string
	Pass   string
	Schema string
}

// Подключение к базе данных
//
// Пока доступен только mysql, но xorm умеет и в другие БД
func Connect(db DB, logger *rotatelogs.RotateLogs) (*xorm.Engine, error) {
	engine, err := xorm.NewEngine(
		"mysql",
		fmt.Sprintf(
			"%s:%s@tcp(localhost:3306)/%s?charset=utf8",
			db.User, db.Pass, db.Schema),
	)
	if err != nil {
		return nil, err
	}
	sqlLogger := xlog.NewSimpleLogger(logger)
	engine.SetLogger(sqlLogger)
	engine.ShowSQL(true)
	engine.SetMapper(names.SameMapper{})

	err = engine.Sync(
		&User{},
		&TgUser{},
		&Group{},
		&Lesson{},
		&Staff{},
		&ShedulesInUser{},
		&File{},
		&TempMsg{},
		&GroupChatInfo{},
		&ICalendar{},
	)
	if err != nil {
		return nil, err
	}

	return engine, nil
}

// Генерация уникального номера для таблицы table
func GenerateID(engine *xorm.Engine, table any) (int64, error) {
	id := rand.Int63n(899999999) + 100000000 // #nosec G404

	exists, err := engine.ID(id).Get(table)
	if err != nil {
		return 0, err
	}

	if exists {
		return GenerateID(engine, table)
	}

	return id, nil
}

// Инициализация логгера
//
// Каждые 24 часа будет создаваться новый файл, логи старше 14 дней удаляются.
// Также создаётся симлинк актуального лога "name.log"
func InitLog(name string) *rotatelogs.RotateLogs {
	// Создание папки с логами
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		err = os.Mkdir("logs", os.ModePerm)
		if err != nil {
			log.Fatal("failed create logs folder")
		}
	}
	// Определяем абсолютный путь, иначе не заработает симлинк
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	abspath, err := filepath.Abs(filepath.Dir(ex))
	if err != nil {
		log.Fatal(err)
	}
	// Непосредственно файл лога
	path := fmt.Sprintf("%s/logs/%s.log", abspath, name)
	writer, err := rotatelogs.New(
		fmt.Sprintf("%s.%s", path, "%Y-%m-%d.%H%M%S"),
		rotatelogs.WithLinkName(fmt.Sprintf("%s/logs/%s.log", abspath, name)),
		rotatelogs.WithMaxAge(time.Hour*24*14),
		rotatelogs.WithRotationTime(time.Hour*24),
	)
	if err != nil {
		log.Fatalf("failed to Initialize Log File %s: %s", name, err)
	}

	return writer
}
