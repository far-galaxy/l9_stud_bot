from telegram import (
    InlineKeyboardMarkup,
    InlineKeyboardButton,
    ReplyKeyboardMarkup,
    KeyboardButton,
)


class Keyboard:
    def confirm() -> InlineKeyboardMarkup:
        """Клавиатура Да/Нет"""
        buttons = [
            [
                InlineKeyboardButton("Да", callback_data="yes"),
                InlineKeyboardButton("Нет", callback_data="no"),
            ]
        ]
        return InlineKeyboardMarkup(buttons)

    def cancel() -> ReplyKeyboardMarkup:
        """Кнопка отмены"""
        buttons = [[KeyboardButton("Отмена")]]
        return ReplyKeyboardMarkup(
            buttons, resize_keyboard=True, one_time_keyboard=True
        )

    def menu() -> ReplyKeyboardMarkup:
        """Кнопка Главного меню"""
        buttons = [[KeyboardButton("Главное меню")]]
        return ReplyKeyboardMarkup(
            buttons, resize_keyboard=True, one_time_keyboard=True
        )
