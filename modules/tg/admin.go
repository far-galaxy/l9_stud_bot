package tg

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/api"
	"stud.l9labs.ru/bot/modules/database"
)

var AdminKey = []string{"scream", "stat", "update"}

func (bot *Bot) AdminHandle(msg *tgbotapi.Message) (tgbotapi.Message, error) {
	if strings.Contains(msg.Text, "/scream") {
		return bot.Scream(msg)
	} else if strings.Contains(msg.Text, "/stat") {
		return bot.Stat()
	} else if strings.Contains(msg.Text, "/update") {
		return bot.Update()
	}

	return nilMsg, nil
}

// Принудительное обновление всех .ics файлов
func (bot *Bot) Update() (tgbotapi.Message, error) {
	admin := database.TgUser{
		ChatID: bot.TestUser,
	}
	if err := UpdateICS(bot); err != nil {
		return bot.SendMsg(&admin, err.Error(), nil)
	}

	return bot.SendMsg(&admin, "Календари обновлены", nil)
}

func UpdateICS(bot *Bot, tsh ...database.ShedulesInUser) error {
	var ics []database.ICalendar
	if len(tsh) > 0 {
		q := database.ICalendar{
			IsGroup:   true,
			SheduleID: tsh[0].SheduleID,
		}
		if err := bot.DB.UseBool("IsGroup").
			Find(&ics, q); err != nil {
			return err
		}
	} else {
		if err := bot.DB.Find(&ics); err != nil {
			return err
		}
	}
	for _, i := range ics {
		sh := database.Schedule{
			IsGroup:    i.IsGroup,
			ScheduleID: i.SheduleID,
		}
		lessons, err := api.GetSemesterLessons(bot.DB, sh)
		if err != nil {
			log.Println(err)
		}
		var userSchedule database.ShedulesInUser
		if _, err := bot.DB.Where("l9id = ?", i.L9ID).Get(&userSchedule); err != nil {
			return err
		}
		if err := bot.CreateICSFile(lessons, userSchedule, i.ID); err != nil {
			log.Println(err)
		}
	}

	return nil
}

// Рассылка
func (bot *Bot) Scream(msg *tgbotapi.Message) (tgbotapi.Message, error) {
	var users []database.TgUser
	if err := bot.DB.Where("tgid > 0").Find(&users); err != nil {
		return nilMsg, err
	}
	scream := tgbotapi.NewMessage(
		0,
		strings.TrimPrefix(msg.Text, "/scream"),
	)
	for i, u := range users {
		scream.ChatID = u.ChatID
		if _, err := bot.TG.Send(scream); err != nil {
			bot.CheckBlocked(err, users[i])
		}
	}
	scream.ChatID = bot.TestUser
	scream.Text = "Сообщения отправлены"

	return bot.TG.Send(scream)
}

// Статистика
func (bot *Bot) Stat() (tgbotapi.Message, error) {
	total, err := bot.DB.Count(database.TgUser{})
	if err != nil {
		return nilMsg, err
	}

	active, err := bot.DB.Count(database.ShedulesInUser{})
	if err != nil {
		return nilMsg, err
	}

	txt := fmt.Sprintf("Build: %s\n\n", bot.Build)
	txt += fmt.Sprintf("Текущая сессия:\nСообщений: %d\nНажатий на кнопки: %d\n\n", bot.Messages, bot.Callbacks)
	txt += fmt.Sprintf("Всего пользователей: %d\nАктивных пользователей: %d\n\nСтатистика по группам:\n", total, active)

	res, err := bot.DB.Query("SELECT G.GroupName, COUNT(U.L9ID) AS UserCount " +
		"FROM `Group` G LEFT JOIN ShedulesInUser U ON G.GroupID = U.SheduleID " +
		"GROUP BY G.GroupName HAVING UserCount > 0 ORDER BY UserCount DESC;")
	if err != nil {
		return nilMsg, err
	}

	for _, r := range res {
		txt += fmt.Sprintf("%s | %s\n", r["GroupName"], r["UserCount"])
	}
	stat := tgbotapi.NewMessage(bot.TestUser, txt)

	return bot.TG.Send(stat)
}
