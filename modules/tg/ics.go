package tg

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/api"
	"stud.l9labs.ru/bot/modules/database"
)

type LessonStr struct {
	TypeIcon    string
	TypeStr     string
	Name        string
	Begin       time.Time
	End         time.Time
	SubGroup    int64
	TeacherName string
	Place       string
	Comment     string
}

// Создание и отправка .ics файла с расписанием для приложений календаря
func (bot *Bot) CreateICS(shedule database.Schedule, query ...tgbotapi.CallbackQuery) error {
	if _, err := bot.ActShedule(&shedule); err != nil {
		return err
	}
	if !shedule.IsGroup {
		_, err := bot.SendMsg(
			shedule.TgUser,
			"Скачивание .ics для преподавателей пока недоступно (:",
			nil,
		)

		return err
	}

	var ics database.ICalendar
	if shedule.IsPersonal {
		ics = database.ICalendar{
			IsPersonal: true,
			L9ID:       shedule.TgUser.L9ID,
		}
	} else {
		ics = database.ICalendar{
			IsPersonal: false,
			IsGroup:    shedule.IsGroup,
			SheduleID:  shedule.ScheduleID,
		}
	}

	exists, err := bot.DB.UseBool("IsPersonal", "IsGroup").Get(&ics)
	if err != nil {
		return err
	}

	// Если .ics уже есть
	if exists {
		return bot.SendICS(shedule.TgUser, ics.ID, query)
	}

	lessons, err := api.GetSemesterLessons(bot.DB, shedule)
	if err != nil {
		return err
	}
	if len(lessons) == 0 {
		return nil
	}

	id, err := database.GenerateID(bot.DB, &database.ICalendar{})
	if err != nil {
		return err
	}
	ics.ID = id
	ics.IsGroup = shedule.IsGroup
	ics.SheduleID = shedule.ScheduleID
	if _, err := bot.DB.InsertOne(ics); err != nil {
		return err
	}

	var userSchedule database.ShedulesInUser
	if _, err := bot.DB.Where("l9id = ?", shedule.TgUser.L9ID).Get(&userSchedule); err != nil {
		return err
	}

	if err := bot.CreateICSFile(lessons, userSchedule, id); err != nil {
		return err
	}

	return bot.SendICS(shedule.TgUser, id, query)
}

// Сохраниение .ics в файл
func (bot *Bot) CreateICSFile(lessons []database.Lesson, shedule database.ShedulesInUser, id int64) error {
	txt, err := bot.GenerateICS(lessons, shedule)
	if err != nil {
		return err
	}

	path := "./shedules/ics/"
	fileName := fmt.Sprintf("%s%d.ics", path, id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}

	f, _ := os.Create(fileName)
	defer f.Close()
	if _, err := f.WriteString(txt); err != nil {
		return err
	}

	return nil
}

// Отправка сообщения с .ics
func (bot *Bot) SendICS(user *database.TgUser, id int64, query []tgbotapi.CallbackQuery) error {
	if _, err := bot.SendMsg(
		user,
		fmt.Sprintf(
			"📖 Инструкция по установке: https://stud.l9labs.ru/bot/ics\n\n"+
				"Ссылка для Календаря:\n"+
				"https://stud.l9labs.ru/ics/%d.ics\n\n"+
				"‼️ Файл по данной ссылке <b>не для скачивания</b> ‼️\n"+
				"Иначе не будет синхронизации\n\n ",
			id,
		),
		nil,
	); err != nil {
		return err
	}
	if len(query) != 0 {
		ans := tgbotapi.NewCallback(query[0].ID, "")
		if _, err := bot.TG.Request(ans); err != nil {
			return err
		}
	}

	return nil
}

// Создание непосредственно ICS файла
func (bot *Bot) GenerateICS(
	lessons []database.Lesson,
	shedule database.ShedulesInUser,
) (
	string,
	error,
) {
	var strLessons []LessonStr
	for _, lesson := range lessons {
		if lesson.Type == database.Window {
			continue
		}
		if lesson.Type == database.Military && !shedule.Military {
			continue
		}
		var teacherName string
		if lesson.StaffID != 0 {
			staff, err := api.GetStaff(bot.DB, lesson.StaffID)
			if err != nil {
				return "", err
			}
			teacherName = fmt.Sprintf("%s %s", staff.FirstName, staff.LastName)
		}

		l := LessonStr{
			TypeIcon:    Icons[lesson.Type],
			TypeStr:     Comm[lesson.Type],
			Name:        lesson.Name,
			Begin:       lesson.Begin.UTC(),
			End:         lesson.End.UTC(),
			SubGroup:    lesson.SubGroup,
			TeacherName: teacherName,
			Place:       lesson.Place,
			Comment:     lesson.Comment,
		}
		strLessons = append(strLessons, l)
	}

	tmpl, err := template.ParseFiles("templates/shedule.ics")
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, strLessons)
	if err != nil {
		return "", err
	}
	txt := rendered.String()

	return txt, nil
}
