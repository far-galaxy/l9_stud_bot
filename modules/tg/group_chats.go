package tg

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/database"
)

func (bot *Bot) HandleGroup(msg *tgbotapi.Message, now time.Time) (tgbotapi.Message, error) {
	/*
		group := database.GroupChatInfo{
			ChatID: msg.Chat.ID,
		}
		if _, err := bot.DB.Get(&group); err != nil {
			return nilMsg, err
		}
	*/
	bot.Debug.Printf("ChatCommand [%d] %s", msg.Chat.ID, msg.Text)

	// TODO: добавить игнор команд для групповых чатов в ЛС
	/*
		if strings.Contains(msg.Text, "/add") {
			cmd := strings.Split(msg.Text, " ")
			if len(cmd) == 1 {
				ans := tgbotapi.NewMessage(
					group.ChatID,
					"Необходимо указать запрос прямо в команде!\n"+
						"Например, /add@l9_stud_bot 2405-240502D",
				)

				return bot.TG.Send(ans)
			}
			fakeUser := database.TgUser{
				TgID:   group.ChatID,
				PosTag: database.Ready,
			}

			return bot.Find(now, &fakeUser, cmd[1])
		}*/
	if KeywordContains(msg.Text, []string{"/group", "/staff"}) {
		fakeUser := database.TgUser{
			ChatID: msg.Chat.ID,
			PosTag: database.Ready,
		}

		return bot.GetSheduleFromCmd(now, &fakeUser, msg.Text)
	}
	ans := tgbotapi.NewMessage(msg.Chat.ID, "Исполняю команду, которую не знаю, как исполнять")
	ans.ReplyMarkup = CancelKey()

	return bot.TG.Send(ans)
}

func (bot *Bot) ChatActions(update tgbotapi.Update) (tgbotapi.Message, error) {
	action := update.MyChatMember

	if action.NewChatMember.Status == "member" &&
		action.OldChatMember.Status != "administrator" {
		msg := tgbotapi.NewMessage(
			action.Chat.ID,
			"Всем привет! Теперь вы можете посмотреть расписание прямо в чате :)\n"+
				fmt.Sprintf("Просто начни писать @%s, а далее введи и выбери нужную группу или преподавателя\n\n", bot.Name)+
				"Подробнее: https://youtube.com/shorts/oCPIsoOILYU",
		)
		/*
			group := database.GroupChatInfo{
				ChatID: action.Chat.ID,
			}
			if _, err := bot.DB.InsertOne(group); err != nil {
				return nilMsg, err
			}
		*/

		return bot.TG.Send(msg)
	} else if action.NewChatMember.IsAdministrator() {
		msg := tgbotapi.NewMessage(
			action.Chat.ID,
			"Не думаю, что есть необходимость делать меня администратором\n"+
				"А то вдруг начнётся восстание машин и я случайно поудаляю ваши чаты (:\n\n"+
				"Отменить действие можно в меню <b>Управление группой -> Администраторы</b>",
		)
		msg.ParseMode = tgbotapi.ModeHTML

		return bot.TG.Send(msg)
	}

	return nilMsg, nil
}
