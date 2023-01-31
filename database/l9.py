from .asql import Database


class L9_DB:
    """Класс взаимодействия с базой пользователей L9
    (перспектива для сайта)
    """

    def __init__(self, user, password):
        self.db = Database('localhost', user, password)
        self.db.initDatabase('l9_db')
        self.db.executeFile('l9')

    def countUsers(self) -> int:
        return len(self.db.get('users', None, ['l9Id']))

    def initUser(self, uid):
        result = self.db.get('users', f"l9Id = {uid}", ["l9Id"])
        if result == []:
            l9Id = self.db.newID('users', "l9Id")
            user = {"l9Id": l9Id}
            self.db.insert('users', user)
        else:
            l9Id = result[0][0]

        return l9Id
