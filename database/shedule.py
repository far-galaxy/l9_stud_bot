from .l9 import L9_DB
from .a_ssau_parser import *
import telegram
from configparser import ConfigParser
import datetime


class Shedule_DB:
    """Класс взаимодействия с базой расписания"""

    def __init__(self, l9lk: L9_DB, first_week):
        self.l9lk = l9lk
        self.db = l9lk.db
        self.first_week = first_week
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

    def loadShedule(self, groupId: str, date: datetime.datetime):
        """Загрузка расписания"""
        week = date.isocalendar()[1] - self.first_week

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

    def getGroups(self, l9Id: str):
        groups = self.db.execute(
            (
                f'SELECT g.groupId, groupName FROM '
                f'`groups_users` AS gu JOIN `groups` AS g '
                'ON gu.groupId=g.groupId WHERE '
                f'l9Id = {l9Id}'
            )
        ).fetchall()

        return groups if groups != [] else None

    def getLesson(self, lessonId):
        icons = {'other': '📙', 'lect': '📗', 'lab': '📘', 'pract': '📕'}

        lesson = self.db.get('lessons', f'lessonId = {lessonId}')

        if lesson != []:
            lesson = lesson[0]

            teacher = None
            if lesson[12] != None:
                teacher = self.db.get(
                    'teachers', f'teacherId = {lesson[12]}'
                )

            if teacher != None and teacher != []:
                info = teacher[0]
                teacher = f"{info[1]} {info[2][0]}.{info[3][0]}."

            json_lesson = {
                'numInDay': lesson[5],
                'type': icons[lesson[7]],
                'name': lesson[8],
                'place': lesson[13],
                'teacher': teacher,
                'add_info': lesson[14],
                'begin': lesson[10],
                'end': lesson[11],
            }

            return json_lesson

        else:
            return {'empty'}

    def strLesson(self, lessonIds):
        lesson = [self.getLesson(i) for i in lessonIds]
        begin = lesson[0]['begin']
        end = lesson[0]['end']
        text = "\n📆  %02i:%02i - %02i:%02i" % (
            begin.hour,
            begin.minute,
            end.hour,
            end.minute,
        )

        for l in lesson:
            add_info = "" if l['add_info'] == None else "\n" + l['add_info']
            teacher = "" if l['teacher'] == None else "\n👤  " + l['teacher']
            place = "" if l['place'] == None else f"\n🧭 {l['place']}"
            text += f"\n{l['type']} {l['name']}{place}{teacher}{add_info}\n"
        return text

    def nearLesson(self, date: datetime.datetime, l9Id: str, retry=False):
        str_time = date.isoformat(sep=' ')
        groupIds = self.getGroups(l9Id)

        if groupIds != None:
            second_gr = (
                f' OR groupId = {groupIds[1][0]}'
                if len(groupIds) == 2
                else ''
            )
            lessonId = self.db.get(
                'lessons',
                f"(groupId = {groupIds[0][0]}{second_gr}) AND `end` > '{str_time}' "
                'ORDER BY `begin` LIMIT 2',
                ['lessonId, begin'],
            )

            if lessonId != []:
                if len(lessonId) == 2 and lessonId[0][1] == lessonId[1][1]:
                    return self.strLesson([lessonId[0][0], lessonId[1][0]])
                else:
                    return self.strLesson([lessonId[0][0]])

            elif not retry:
                for groupId in [i for i in groupIds if i[0] > 1000]:
                    self.loadShedule(
                        groupId[0], date + datetime.timedelta(days=7)
                    )
                return self.nearLesson(date, l9Id, retry=True)
