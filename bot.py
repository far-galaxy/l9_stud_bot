from database.l9 import L9_DB
from database.tg import TG_DB
from database.shedule import Shedule_DB
from utils.config import *
from utils.stuff import *
import telegram
from tg.keyboards import Keyboard
import logging
from logging.handlers import TimedRotatingFileHandler as TRFL
import configparser
import datetime

logger = logging.getLogger('bot')


def initLogger():
    if not os.path.isdir(f'logs/bot'):
        os.makedirs(f'logs/bot')

    f_handler = TRFL(f'./logs/bot/log', when='midnight', encoding="utf-8")

    f_format = logging.Formatter(
        '%(asctime)s - %(levelname)s - %(message)s',
        datefmt='%d-%b-%y %H:%M:%S',
    )
    f_handler.setFormatter(f_format)
    f_handler.setLevel(logging.INFO)
    logger.addHandler(f_handler)

    c_handler = logging.StreamHandler()
    c_format = logging.Formatter('%(levelname)s : %(message)s')
    c_handler.setFormatter(c_format)
    logger.addHandler(c_handler)
    logger.setLevel(logging.DEBUG)


class Bot:
    def __init__(
        self,
        token: str,
        db: L9_DB,
        tg_db: TG_DB,
        shedule: Shedule_DB,
        limit=150,
    ):
        self.l9lk = db
        self.tg_db = tg_db
        self.shedule = shedule
        self.tg = telegram.Bot(token)
        self.limit = limit
        self.udpate_id = None
        self.isWork = True

    def answer(self, query: telegram.CallbackQuery, text=None, alert=False):
        try:
            query.answer(text, alert)
        except telegram.error.BadRequest:
            pass

    def edit(self, query: telegram.CallbackQuery, text=None, markup=None):
        if isinstance(text, str):
            try:
                query.edit_message_text(text)
            except telegram.error.BadRequest:
                pass

        if isinstance(markup, telegram.ReplyKeyboardMarkup):
            try:
                query.edit_message_reply_markup(text)
            except telegram.error.BadRequest:
                pass

    def checkMessages(self):
        """Проверка и обработка входящих сообщений"""

        updates = self.tg.get_updates(offset=self.udpate_id, timeout=5)
        for update in updates:
            self.udpate_id = update.update_id + 1

            if update.callback_query:
                query = update.callback_query
                tag, l9Id, log = self.tg_db.getTagC(query)
                logger.info(log)
                tgId = query.from_user.id

                if 'conf' in tag:
                    self.confirmGroup(query, tag, l9Id)

            if update.message:
                query = update.message
                tag, l9Id, log = self.tg_db.getTagM(query)
                logger.info(log)
                tgId = query.from_user.id

                if tag == 'not_started':
                    self.start(query)

                elif query.text == 'Отмена':
                    if self.shedule.getGroups(l9Id) != None:
                        self.tg_db.changeTag(tgId, 'ready')
                        self.tg.sendMessage(
                            tgId,
                            loc['etc']['cancel'],
                            reply_markup=Keyboard.menu(),
                        )
                    else:
                        self.tg_db.changeTag(tgId, 'add')
                        self.tg.sendMessage(
                            tgId,
                            loc['etc']['need_group'],
                            reply_markup=Keyboard.menu(),
                        )

                elif tag == 'add':
                    self.addGroup(l9Id, query)

                elif query.text == 'Главное меню':
                    now = query.date
                    # now = datetime.datetime(2023, 2, 6, 14, 0)
                    pairs = self.shedule.nearLesson(now, l9Id)
                    if pairs != None:
                        pair = pairs[0][0][1]
                        if pair.date() > now.date():
                            text = loc['shedule']['next_days']
                            day = datetime.timedelta(days=1)
                            if pair.date() - now.date() == day:
                                text += f" {loc['shedule']['tomorrow']}:\n"
                            else:
                                text += (
                                    f' {pair.day} {month[pair.month-1]}:\n'
                                )
                        elif pair.time() > now.time():
                            text = f"{loc['shedule']['today']}:\n"
                        else:
                            text = f"{loc['shedule']['now']}:\n"
                        text += self.shedule.strLesson(
                            [p[0] for p in pairs[0]]
                        )

                        if len(pairs) == 2 and pair.date() == now.date():
                            text += f"\n{loc['shedule']['next']}:\n"
                            text += self.shedule.strLesson(
                                [p[0] for p in pairs[1]]
                            )
                        elif pair.date() == now.date():
                            text += f"\n{loc['shedule']['next_empty']}"
                        # TODO: Добавить смайликов
                        self.tg.sendMessage(
                            tgId,
                            text,
                            reply_markup=Keyboard.menu(),
                        )
                    else:
                        self.tg.sendMessage(
                            tgId,
                            loc['shedule']['no_shedule'],
                            reply_markup=Keyboard.menu(),
                        )

                else:
                    self.tg.sendMessage(
                        tgId,
                        loc['etc']['oops'],
                        reply_markup=Keyboard.menu(),
                    )

    def start(self, query: telegram.Message):
        """Обработка нового пользователя"""

        # Проверка лимита пользователей и обработка лишних
        count = self.l9lk.countUsers()
        tgId = query.from_user.id

        if count >= self.limit:
            self.tg.sendMessage(
                tgId,
                loc['etc']['overlimit'],
            )

        else:
            self.tg_db.changeTag(tgId, 'add')
            self.tg.sendMessage(
                tgId,
                loc['etc']['hello'],
            )

    def addGroup(self, l9Id: int, query: telegram.Message):
        """Процесс добавления группы"""

        groupName = query.text
        tgId = query.from_user.id

        result = self.shedule.checkGroupExists(groupName, l9Id)
        if 'OK' in result:
            _, groupName, specName = result.split(';')
            self.tg_db.changeTag(tgId, 'ready')
            self.tg.sendMessage(
                tgId,
                loc['group']['connected'] % (groupName, specName),
                reply_markup=Keyboard.menu(),
            )

        elif result == 'Exists':
            self.tg.sendMessage(
                tgId,
                loc['group']['exists'],
                reply_markup=Keyboard.cancel(),
            )

        elif result == 'Error':
            self.tg.sendMessage(
                tgId,
                loc['group']['error'],
                reply_markup=Keyboard.cancel(),
            )

        elif 'ssau.ru' in result:
            self.tg_db.changeTag(tgId, f'conf_{result[21:]}')
            self.tg.sendMessage(
                tgId,
                loc['group']['checkShedule'] % (result),
                reply_markup=Keyboard.confirm(),
            )

        else:
            self.tg.sendMessage(
                tgId,
                loc['group']['empty'],
                reply_markup=Keyboard.cancel(),
            )

    def confirmGroup(
        self, query: telegram.CallbackQuery, tag: str, l9Id: str
    ):
        """Процесс подтверждения группы и загрузка расписания"""
        tgId = query.from_user.id
        self.answer(query)
        if query.data == 'yes':
            self.edit(query, loc['group']['loading'])
            self.shedule.loadShedule(tag[5:], query.message.date)
            self.shedule.db.insert(
                'groups_users',
                {'l9Id': l9Id, 'groupId': tag[5:]},
            )
            self.edit(query, loc['group']['loaded'], Keyboard.menu())
            self.tg_db.changeTag(tgId, 'ready')
        else:
            self.edit(query, loc['group']['nogroup'], Keyboard.cancel())
            self.tg_db.changeTag(tgId, 'add')


if __name__ == "__main__":
    initLogger()
    logger.info("Start bot")

    loc = configparser.ConfigParser()
    loc.read('./locale/ru.ini', encoding='utf-8')

    config = loadJSON("config")
    l9lk = L9_DB(**config['sql'])
    tg_db = TG_DB(l9lk)
    shedule = Shedule_DB(l9lk, config['first_week'])
    bot = Bot(
        config['tg']['token'], l9lk, tg_db, shedule, config['tg']['limit']
    )

    logger.info("Bot ready!")

    while bot.isWork:
        msgs = bot.checkMessages()
