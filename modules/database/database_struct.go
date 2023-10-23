package database

import "time"

// Пользователь системы (задел под сайт)
type User struct {
	L9Id int64 `xorm:"pk"`
}

// Позиция пользователя в чате
type Position string

const (
	NotStarted Position = "not_started" // Только начал диалог с ботом
	Ready      Position = "ready"       // Готов к дальнейшим действиям
	Add        Position = "add"         // Подключает личное расписание
	Set        Position = "set"         // Устанавливает время
	Delete     Position = "del"         // Отключается от группы
)

// TODO: к v1 исправить названия

// Пользователь Telegram
type TgUser struct {
	L9Id   int64  `xorm:"pk"`
	TgId   int64  // TODO: переименовать в ChatID в v1
	Name   string // XXX: должно быть удалено в v1
	PosTag Position
}

// TODO: переименовать к v1 в Options

// Подключённое к пользователю расписание
type ShedulesInUser struct {
	UID       int64 `xorm:"pk autoincr"` // Не забывать про автоинкремент!!!
	L9Id      int64
	IsGroup   bool
	SheduleId int64
	Subgroup  int64
	NextNote  bool
	NextDay   bool
	NextWeek  bool
	First     bool
	FirstTime int `xorm:"default 45"`
	Military  bool
}

// Учебная группа
type Group struct {
	GroupId   int64  `xorm:"pk"`
	GroupName string // Полный номер группы
	SpecName  string // Шифр и название специальности
	LastUpd   time.Time
	LastCheck time.Time
}

// Преподаватель
type Teacher struct {
	TeacherId int64  `xorm:"pk"`
	FirstName string // Фамилия
	LastName  string // Имя, отчество и прочие окончания
	ShortName string // Инициалы
	SpecName  string // Место работы
	LastUpd   time.Time
	LastCheck time.Time
}

// Тип занятий
type Kind string

const (
	Lection    Kind = "lect"
	Practice   Kind = "pract"
	Lab        Kind = "lab"
	Other      Kind = "other"
	Military   Kind = "mil"
	Window     Kind = "window"
	Exam       Kind = "exam"
	Consult    Kind = "cons"
	CourseWork Kind = "kurs"
	Test       Kind = "test"
	Unknown    Kind = "unknown"
)

// Занятие
type Lesson struct {
	LessonId     int64 `xorm:"pk autoincr"`
	NumInShedule int
	Type         Kind
	Name         string
	GroupId      int64
	Begin        time.Time
	End          time.Time
	TeacherId    int64
	Place        string
	Comment      string
	SubGroup     int64
	Hash         string
}

// Файлы, залитые в Telegram
type FileType string

const (
	Photo FileType = "photo"
	ICS   FileType = "ics"
)

type File struct {
	Id int64 `xorm:"pk autoincr"`
	FileType
	FileId     string
	TgId       int64
	IsPersonal bool
	IsGroup    bool
	SheduleId  int64
	Week       int
	LastUpd    time.Time
}

// Самоуничтожающиеся сообщения
type TempMsg struct {
	ID        int64 `xorm:"pk autoincr"`
	TgId      int64
	MessageId int
	Destroy   time.Time
}

// Данные о чате, в который добавили бота
type GroupChatInfo struct {
	ID        int64 `xorm:"pk autoincr"`
	ChatID    int64 `xorm:"unique"`
	IsGroup   bool
	SheduleID int64
}
