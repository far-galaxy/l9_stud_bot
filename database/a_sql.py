from mysql.connector import connect
from mysql.connector.cursor_cext import CMySQLCursor
import random


class Database:
    """Модуль для mysql-connector"""

    def __init__(self, host: str, user: str, password: str):
        """Подключение к серверу MySQL"""
        self.database = connect(host=host, user=user, password=password)
        self.cursor = self.database.cursor()

    def execute(self, query: str, commit=False) -> CMySQLCursor:
        """Выполнить SQL запрос
        Примечание: в целях безопасности функция игнорирует запросы DROP и TRUNCATE

        Args:
            :query: текст запроса
            :commit: [optional] сохранить изменения
        Returns:
            :cursor: объект курсора
        """
        if (
            query.lower().find("drop") == -1
            and query.lower().find("truncate") == -1
        ):
            print(query)
            self.cursor.execute(query)
            if commit:
                self.database.commit()
        return self.cursor

    def executeFile(self, filename: str, commit=False) -> CMySQLCursor:
        """Выполнить запрос из .sql файла

        Args:
            :filename: название файла (без расширения)
            :commit: [optional] сохранить изменения
        Returns:
            :cursor: объект курсора
        """

        with open(f'database/{filename}.sql', encoding='utf-8') as f:
            query = f.read()
            return self.execute(query, commit).fetchall()

    def initDatabase(self, name: str):
        """Создать базу данных, если таковая отсутствует,
        и переключиться на неё для использования в дальнейших запросах
        Args:
            :name: название базы данных
        """

        self.execute(f"CREATE DATABASE IF NOT EXISTS {name};")
        self.execute(f"USE {name};")

    def initTable(self, name: str, head: str):
        """Создать таблицу, если таковая отсутствует

        TODO: вырезать эту функцию, поскольку теперь БД инициализирутся
        из файла

        Args:
            :name: название таблицы
            :head: двумерный список, в строках которых описаны столбцы таблицы
        """
        query = f"CREATE TABLE IF NOT EXISTS `{name}` ("
        query += ", ".join([" ".join(i) for i in head])
        query += ");"
        self.execute(query)

    def insert(self, name: str, values: dict):
        """Вставить значение в таблицу

        Args:
            :name: название таблицы
            :values: словарь их названий столбцов и их значений
        """
        query = f"INSERT IGNORE INTO `{name}` ("
        query += ", ".join(values) + ") VALUES ("
        query += (
            ", ".join(
                [
                    f'"{i}"' if (i != None) else "NULL"
                    for i in values.values()
                ]
            )
            + ");"
        )
        self.execute(query, commit=True)

    def get(self, name: str, condition=None, columns=None) -> list:
        """Получить данные из таблицу по запросу вида:

        :SELECT columns FROM name WHERE condition:

        Args:
            :name: название таблицы
            :condition: SQL условие для выборки, для получения всех строк оставить None
            :columns: [optional] список столбцов, которые необходимо выдать, для всех столбцов оставить None
        """
        query = "SELECT " + (', '.join(columns) if columns != None else "*")
        query += f" FROM `{name}`"
        query += f" WHERE {condition};" if condition != None else ";"
        result = self.execute(query).fetchall()
        return result

    def update(self, name: str, condition: str, new: str):
        """Обновить данные в строке

        Args:
                :name: название таблицы
                :condition: SQL условие для выборки строки
                :new: SQL условия для замены значений столбцов
        """
        query = f"UPDATE {name}"
        query += f" SET {new} WHERE {condition};"
        self.execute(query, commit=True)

    def newID(self, name: str, id_name: str) -> str:
        """Сгенерировать уникальный ID из 9 цифр

        Args:
            :name: название таблицы пользователей
            :id_name: название столбца уникальных ID
        Returns:
            :someID: строка с уникальным ID
        """
        someID = random.randint(100000000, 999999999)

        result = self.get(name, f"{id_name} = {someID}")

        exist = result != []
        if not exist:
            return str(someID)
        else:
            self.newID()
