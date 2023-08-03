package database

import "time"

type User struct {
	L9Id int64 `xorm:"pk"`
}

type Position string

const (
	NotStarted Position = "not_started"
	Ready      Position = "ready"
)

type TgUser struct {
	L9Id   int64 `xorm:"pk"`
	TgId   int64
	Name   string
	PosTag Position
}

type ShedulesInUser struct {
	UID       int64 `xorm:"pk autoincr"` // Не забывать про автоинкремент!!!
	L9Id      int64
	IsTeacher bool
	SheduleId int64
}

type Group struct {
	GroupId   int64  `xorm:"pk"`
	GroupName string // Полный номер группы
	SpecName  string // Шифр и название специальности
}

type Teacher struct {
	TeacherId int64  `xorm:"pk"`
	FirstName string // Фамилия
	LastName  string // Имя, отчество и прочие окончания
	SpecName  string // Место работы
}

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
