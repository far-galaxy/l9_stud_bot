package database

import "time"

// Пользователь системы (задел под сайт)
type User struct {
	L9Id int64 `xorm:"pk"`
}

type Position string

const (
	NotStarted  Position = "not_started"      // Только начал диалог с ботом
	Ready       Position = "ready"            // Готов к дальнейшим действиям
	Add         Position = "add"              // Подключает личное расписание
	SelAddGroup Position = "select_add_group" // Выбирает группу в личное расписание
	SelAddStaff Position = "select_add_staff" // Выбирает преподавателя в личное расписание
	SelSeeGroup Position = "select_see_group" // Выбирает группу для автономной карточки
	SelSeeStaff Position = "select_see_staff" // Выбирает преподавателя для автономной карточки
)

// Пользователь Telegram
type TgUser struct {
	L9Id   int64 `xorm:"pk"`
	TgId   int64
	Name   string
	PosTag Position
}

// Подключённое к пользователю расписание
type ShedulesInUser struct {
	UID       int64 `xorm:"pk autoincr"` // Не забывать про автоинкремент!!!
	L9Id      int64
	IsGroup   bool
	SheduleId int64
	Subgroup  int64
	NextNote  bool
}

// Учебная группа
type Group struct {
	GroupId   int64  `xorm:"pk"`
	GroupName string // Полный номер группы
	SpecName  string // Шифр и название специальности
	LastUpd   time.Time
}

// Преподаватель
type Teacher struct {
	TeacherId int64  `xorm:"pk"`
	FirstName string // Фамилия
	LastName  string // Имя, отчество и прочие окончания
	ShortName string // Инициалы
	SpecName  string // Место работы
	LastUpd   time.Time
}

// Занятие
type Lesson struct {
	LessonId     int64 `xorm:"pk autoincr"`
	NumInShedule int
	Type         string
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

// Файлы, залитые в Telegream
type File struct {
	Id         int64 `xorm:"pk autoincr"`
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
