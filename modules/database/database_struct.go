package database

import "time"

type User struct {
	L9Id int64 `xorm:"pk"`
}

type TgUser struct {
	L9Id   int64 `xorm:"pk"`
	TgId   int64
	Name   string
	PosTag string
}

type ShedulesInUser struct {
	UID       int64 `xorm:"pk"`
	L9Id      int64
	IsTeacher bool
	SheduleId int64
}

type Group struct {
	GroupId   int64 `xorm:"pk"`
	GroupName string
	SpecName  string
}

type Teacher struct {
	TeacherId int64 `xorm:"pk"`
	LastName  string
	FirstName string
	MidName   string
	SpecName  string
}

type Lesson struct {
	LessonId  int64 `xorm:"pk autoincr"`
	Type      string
	Name      string
	GroupId   int64
	Begin     time.Time
	End       time.Time
	TeacherId int64
	Place     string
	Comment   string
	SubGroup  string
}
