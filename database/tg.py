from .l9 import L9_DB
import telegram


class TG_DB:
    """Класс взаимодействия с БД пользователей бота в Telegram"""

    def __init__(self, l9lk: L9_DB):
        self.l9lk = l9lk
        self.db = l9lk.db
        self.db.executeFile('tg')

    def getTagM(self, query: telegram.Message) -> (str, str, str):
        """Получить тэг и l9Id пользователя из сообщения"""

        tgId = query.from_user.id
        name = f'{query.from_user.first_name or ""} {query.from_user.last_name or ""}'

        l9Id = self.db.get('tg_users', f"tgId = {tgId}", ["l9Id"])
        if l9Id == []:
            l9Id = self.l9lk.initUser(0)
            user = {"l9Id": l9Id, "tgId": tgId, "name": name}
            self.db.insert('tg_users', user)
        else:
            l9Id = l9Id[0][0]

        tag = self.db.get('tg_users', f"tgId = {tgId}", ["posTag"])[0][0]

        return tag, l9Id, f'{tgId}\t{tag}\t{name}\tM:{query.text}'

    def getTagC(self, query: telegram.CallbackQuery) -> (str, str, str):
        """Получить тэг и l9Id пользователя из CallbackQuery (кнопки)"""

        tgId = query.from_user.id
        name = f'{query.from_user.first_name or ""} {query.from_user.last_name or ""}'

        l9Id = self.db.get('tg_users', f"tgId = {tgId}", ["l9Id"])
        if l9Id == []:
            l9Id = self.l9lk.initUser(0)
            user = {"l9Id": l9Id, "tgId": tgId, "name": name}
            self.db.insert('tg_users', user)
        else:
            l9Id = l9Id[0][0]

        tag = self.db.get('tg_users', f"tgId = {tgId}", ["posTag"])[0][0]

        return tag, l9Id, f'{tgId}\t{tag}\t{name}\tC:{query.data}'

    def changeTag(self, tgId: int, tag: str) -> None:
        """Сменить тэг пользователя"""
        self.db.update('tg_users', f"tgId = {tgId}", f"posTag = '{tag}'")
