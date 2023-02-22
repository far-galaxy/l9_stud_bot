package database

type User struct {
	L9Id int64 `xorm:"pk"`
}

type TgUser struct {
	L9Id   int64 `xorm:"pk"`
	TgId   int64
	Name   string
	PosTag string
}
