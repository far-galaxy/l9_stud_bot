package tg

import (
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"stud.l9labs.ru/bot/modules/api"
	"stud.l9labs.ru/bot/modules/database"
	"stud.l9labs.ru/bot/modules/htmlschedule"
)

// Получить расписание на неделю
//
// При week == -1 неделя определяется автоматически
func (bot *Bot) GetWeekSummary(
	now time.Time,
	shedule database.Schedule,
	week int,
	caption string,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {

	if _, err := bot.ActShedule(&shedule); err != nil {
		return nilMsg, err
	}

	if week != -1 && week != 0 {
		week += bot.Week
	}
	isCompleted, err := api.CheckWeek(bot.DB, now, &week, shedule)
	week -= bot.Week
	if err != nil {
		return nilMsg, err
	}
	if isCompleted {
		caption = "На этой неделе больше занятий нет\n" +
			"На фото расписание следующей недели"
	}

	var image database.File
	var cols []string
	if !shedule.IsPersonal {
		image = database.File{
			TgId:       shedule.TgUser.TgId,
			FileType:   database.Photo,
			IsPersonal: false,
			IsGroup:    shedule.IsGroup,
			SheduleId:  shedule.ScheduleID,
			Week:       week,
		}
		cols = []string{"IsPersonal", "IsGroup"}
	} else {
		image = database.File{
			TgId:       shedule.TgUser.TgId,
			FileType:   database.Photo,
			IsPersonal: true,
			Week:       week,
		}
		cols = []string{"IsPersonal"}
	}
	has, err := bot.DB.UseBool(cols...).Get(&image)
	if err != nil {
		return nilMsg, err
	}

	// Получаем дату обновления расписания
	lastUpd, err := api.GetLastUpdate(bot.DB, shedule)
	if err != nil {
		return nilMsg, err
	}

	if !has || image.LastUpd.Before(lastUpd) {
		// Если картинки нет, или она устарела
		if has {
			if _, err := bot.DB.Delete(&image); err != nil {
				return nilMsg, err
			}
		}

		gen, err := bot.SendMsg(shedule.TgUser, "Создание изображения...\n", nil)
		if err != nil {
			return nilMsg, err
		}

		img, err := htmlschedule.CreateWeekImg(
			bot.DB,
			bot.WkPath,
			now,
			shedule.TgUser,
			shedule,
			week,
			bot.Week,
			caption,
			editMsg...,
		)

		if err := bot.DelMsg(gen); err != nil {
			return nilMsg, err
		}
		if err != nil {
			markup := SummaryKeyboard(
				Week,
				shedule,
				week,
				false,
			)
			if strings.Contains(err.Error(), "no lessons") {
				return bot.SendMsg(
					shedule.TgUser,
					fmt.Sprintf("На %d неделе занятий нет", week),
					markup,
				)
			}
			fmt.Println(err)

			return bot.SendMsg(shedule.TgUser, "Возникла ошибка при создании изображения", markup)
		}

		return bot.SendWeekImg(shedule, img, caption, week, now, editMsg...)
	}
	// Если всё есть, скидываем, что есть
	markup := tgbotapi.InlineKeyboardMarkup{}
	if (caption == "" || (caption != "" && isCompleted)) && shedule.TgUser.TgId > 0 {
		connectButton := !shedule.IsPersonal && !bot.IsThereUserShedule(shedule.TgUser)
		markup = SummaryKeyboard(
			Week,
			shedule,
			week,
			connectButton,
		)
	}

	return bot.EditOrSend(shedule.TgUser.TgId, caption, image.FileId, markup, editMsg...)
}

func (bot *Bot) SendWeekImg(
	shedule database.Schedule,
	img tgbotapi.FileBytes,
	caption string,
	week int,
	now time.Time,
	editMsg ...tgbotapi.Message,
) (
	tgbotapi.Message,
	error,
) {
	user := shedule.TgUser
	photo := tgbotapi.NewPhoto(user.TgId, img)
	photo.Caption = caption
	isCompleted := strings.Contains(caption, "На этой неделе больше занятий нет")
	connectButton := !shedule.IsPersonal && !bot.IsThereUserShedule(user)
	if (caption == "" || isCompleted) && user.TgId > 0 {
		photo.ReplyMarkup = SummaryKeyboard(
			Week,
			shedule,
			week,
			connectButton,
		)
	}
	resp, err := bot.TG.Send(photo)
	if err != nil {
		fmt.Println(err)

		return bot.SendMsg(shedule.TgUser, "Возникла ошибка при отправке изображения", nil)
	}
	file := database.File{
		FileId:     resp.Photo[0].FileID,
		FileType:   database.Photo,
		TgId:       user.TgId,
		IsPersonal: shedule.IsPersonal,
		IsGroup:    shedule.IsGroup,
		SheduleId:  shedule.ScheduleID,
		Week:       week,
		LastUpd:    now,
	}
	_, err = bot.DB.InsertOne(file)

	if len(editMsg) != 0 {
		err = bot.DelMsg(editMsg[0])
	}

	return nilMsg, err
}
