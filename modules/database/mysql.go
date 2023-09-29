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

type LogFiles struct {
	DebugFile *os.File
	ErrorFile *os.File
	TgLogFile *os.File
	DBLogFile *os.File
}

func OpenLogs() (files LogFiles) {
	return LogFiles{
		DebugFile: CreateLog("messages"),
		ErrorFile: CreateLog("error"),
		TgLogFile: CreateLog("tg"),
		DBLogFile: CreateLog("sql"),
	}
}

func (files *LogFiles) CloseAll() {
	files.DebugFile.Close()
	files.ErrorFile.Close()
	files.TgLogFile.Close()
	files.DBLogFile.Close()
}

func Connect(db DB, logger *os.File) (*xorm.Engine, error) {
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
		&Teacher{},
		&ShedulesInUser{},
		&File{},
		&TempMsg{},
	)
	if err != nil {
		return nil, err
	}

	return engine, nil
}

func GenerateID(engine *xorm.Engine) (int64, error) {
	id := rand.Int63n(899999999) + 100000000 // #nosec G404

	exists, err := engine.ID(id).Exist(&User{})
	if err != nil {
		return 0, err
	}

	if exists {
		return GenerateID(engine)
	}

	return id, nil
}

// Инициализация логгера
//
// # Каждые 24 часа будет создаваться новый файл, логи старше 14 дней удаляются
//
// Также создаётся симлинк актуального лога name.log
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

// TODO: изобрести раздорбление логов по дате
func CreateLog(name string) *os.File {
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		err = os.Mkdir("logs", os.ModePerm)
		if err != nil {
			log.Fatal("failed create log folder")
		}
	}
	fileName := "./logs/" + name + ".log"
	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		newFile := fmt.Sprintf("./logs/%s.before.%s.log", name, time.Now().Format("06-02-01-15-04-05"))
		err := os.Rename(fileName, newFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	logFile, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("failed open %s.log file: %s", name, err)
	}
	//defer logFile.Close()
	return logFile
}
