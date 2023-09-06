package tg

import (
	"fmt"
	"strings"

	"git.l9labs.ru/anufriev.g.a/l9_stud_bot/modules/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var AdminKey = []string{"scream", "stat"}

func (bot *Bot) AdminHandle(msg *tgbotapi.Message) (tgbotapi.Message, error) {
	nilMsg := tgbotapi.Message{}
	if strings.Contains(msg.Text, "/scream") {
		return bot.Scream(msg)
	} else if strings.Contains(msg.Text, "/stat") {
		return bot.Stat(msg)
	}
	return nilMsg, nil
}

func (bot *Bot) Scream(msg *tgbotapi.Message) (tgbotapi.Message, error) {
	nilMsg := tgbotapi.Message{}
	var users []database.TgUser
	if err := bot.DB.Where("tgid > 0").Find(&users); err != nil {
		return nilMsg, err
	}
	scream := tgbotapi.NewMessage(
		0,
		strings.TrimPrefix(msg.Text, "/scream"),
	)
	for _, u := range users {
		scream.ChatID = u.TgId
		if _, err := bot.TG.Send(scream); err != nil {
			if !strings.Contains(err.Error(), "blocked by user") {
				bot.Debug.Println(err)
			}
		}
	}
	scream.ChatID = bot.TestUser
	scream.Text = "Сообщения отправлены"
	return bot.TG.Send(scream)
}

func (bot *Bot) Stat(msg *tgbotapi.Message) (tgbotapi.Message, error) {
	nilMsg := tgbotapi.Message{}

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

	res, err := bot.DB.Query("SELECT G.GroupName, COUNT(U.L9Id) AS UserCount " +
		"FROM `Group` G LEFT JOIN ShedulesInUser U ON G.GroupId = U.SheduleId " +
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
