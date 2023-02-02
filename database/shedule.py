from .l9 import L9_DB
from .a_ssau_parser import *
import telegram
from configparser import ConfigParser
import datetime


class Shedule_DB:
    """Класс взаимодействия с базой расписания"""

    def __init__(self, l9lk: L9_DB):
        self.l9lk = l9lk
        self.db = l9lk.db
        self.db.executeFile('shedule')

    def checkGroupExists(self, groupName: str, l9Id: str) -> str:
        """Проверка наличия группы в БД и на сайте, а также проверка,
        что пользователь ещё не подключен к группе

        'OK;N_группы;Назв-е_спец-сти' - пользователь успешно подключен \n
        'Exists' - пользователь уже подключен к данной группе \n
        'ssau.ru/groupId=*' - группа отсутствует в базе, но есть на сайте \n
        'Error' - ошибка на стороне сайта
        'Empty' - группа нигде не обнаружена
        """

        groupIdInDB = self.l9lk.db.get(
            'groups',
            f'groupName LIKE "{groupName}%"',
            ['groupId', 'groupName', 'specName'],
        )

        if groupIdInDB != []:
            groupIdInDB = groupIdInDB[0]

            exists = self.l9lk.db.get(
                'groups_users',
                f'l9Id = {l9Id} AND groupId = {groupIdInDB[0]}',
            )
            if exists == []:
                self.l9lk.db.insert(
                    'groups_users',
                    {'l9Id': l9Id, 'groupId': groupIdInDB[0]},
                )
                return f'OK;{groupIdInDB[1]};{groupIdInDB[2]}'

            else:
                return 'Exists'

        else:
            group = findInRasp(groupName)
            if group != None:
                group_url = f'ssau.ru/{group["url"][2:]}'
                gr_num = group["text"]
                groupId = group["id"]

                return group_url

            elif group == 'Error':
                return 'Error'

            else:
                return 'Empty'

    def loadShedule(self, groupId, date, first_week):
        week = date.isocalendar()[1] - first_week

        self.db.execute(
            f'DELETE FROM `lessons` WHERE WEEK(`begin`, 1) = {date.isocalendar()[1]} AND groupId = {groupId};'
        )

        t_info = self.db.get('teachers', None, teacher_columns)
        t_info = [dict(zip(teacher_columns, i)) for i in t_info]
        lessons, teachers = parseWeek(groupId, week, t_info)

        g = getGroupInfo(groupId)
        self.db.insert('groups', g)

        for t in teachers:
            self.l9lk.db.insert('teachers', t)

        for l in lessons:
            self.l9lk.db.insert('lessons', l)
