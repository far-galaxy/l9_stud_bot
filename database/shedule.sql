-- Учебные группы
CREATE TABLE IF NOT EXISTS `groups` (
	`groupId`	bigint		NOT NULL,
	-- Идентификатор группы, соответствует ID в расписании на сайте

	`groupName`	varchar(4)	DEFAULT		'0000',
	-- Учебный номер (наименование) группы

	`specName`	text,
	-- Код и название направления подготовки

	PRIMARY KEY				(`groupId`)
	);


-- Сведения о группах пользователя и настройках каждой группы
CREATE TABLE IF NOT EXISTS `groups_users` (
	`guId`		bigint		NOT NULL	AUTO_INCREMENT,
	-- (service) Идентификатор сведения, устанавливается автоматически

	`l9Id`		bigint		NOT NULL,
	-- Идентификатор пользователя системы

	`groupId`	bigint		NOT NULL,
	-- ID группы, которой принадлежит пользователь
	
	`firstTime`	int			DEFAULT		'45',
	-- Время в минутах, за которое приходит уведомление о начале занятий

	`firstNote`	tinyint		DEFAULT		'1',
	-- Состояние уведомлений о начале занятий:
	-- 0 - выключены
	-- ненулевое значение - включены

	`nextNote`	tinyint		DEFAULT		'1',
	-- Состояние уведомлений о первой или следующей паре
	-- 0 - выключены
	-- ненулевое значение - включены

	PRIMARY KEY				(`guId`),
	KEY			`guid_idx`	(`l9Id`),
	KEY			`gid_idx`	(`groupId`),
	CONSTRAINT	`gr_gu`		FOREIGN KEY	(`groupId`)		REFERENCES	`groups`	(`groupId`)	ON DELETE CASCADE	ON UPDATE CASCADE,
	CONSTRAINT	`l9_gu`		FOREIGN KEY	(`l9Id`)		REFERENCES	`users`		(`l9Id`)	ON DELETE CASCADE	ON UPDATE CASCADE
	);
	
-- Преподаватели
CREATE TABLE IF NOT EXISTS `teachers` (
	`teacherId`	bigint		NOT NULL,
	-- Идентификатор преподавателя, соответствует ID на сайте

	`surname`	varchar(45)	DEFAULT		'Brzęczyszczykiewicz',
	`name`		varchar(45)	DEFAULT		'Grzegorz',
	`midname`	varchar(45)	DEFAULT		'Chrząszczyżewoszywicz',
	-- ФИО преподавателя

	PRIMARY KEY				(`teacherId`)
	);

-- Занятия
CREATE TABLE IF NOT EXISTS `lessons` (
	`lessonId`	bigint		NOT NULL	AUTO_INCREMENT,
	-- (service) Идентификатор занятия, устанавливается автоматически

	`addedBy`	varchar(4)	DEFAULT		'ssau',
	-- Источник информации о занятии:
	-- 'ssau' - сайт Университета
	-- 'lk' - Личный кабинет сайта Университета
	-- '`l9Id`' - добавлено пользователем

	`cancelled`	bigint		DEFAULT		'0',
	-- Отметка, является ли занятие отменённым
	-- '0' - занятие НЕ отменено
	-- '`l9Id`' - занятие отменено пользователем
	
	`migrated`	bigint		DEFAULT		'0',
	-- Отметка, является ли занятие перенесённым
	-- '0' - занятие НЕ перенесено
	-- '`lessonId`' - занятие перенесено на другое время

	`numInDay`	int 		DEFAULT		'1',
	-- Порядковый номер занятия в текущем дне
	
	`numInShedule`	int 		DEFAULT		'1',
	-- Порядковый номер занятия относительно расписания на неделю

	`type`		char(5)		DEFAULT		'other',
	-- Тип занятия:
	-- 'lect' 	- лекция
	-- 'pract' 	- практика (семинар)
	-- 'lab' 	- лабораторная работа
	-- 'other'	- прочие

	`name`		text,
	-- Название занятия

	`groupId`	bigint		NOT NULL,
	-- ID учебной группы

	`begin`		datetime	NOT NULL,
	`end`		datetime	NOT NULL,
	-- Начало и конец занятия

	`teacherId`	bigint		DEFAULT		NULL,
	-- (опционально) ID преподавателя

	`place`		text,
	-- (опционально) Учебная аудитория

	`addInfo`	text,
	-- (опционально) Дополнительная информация

	PRIMARY KEY				(`lessonId`),
	KEY			`gr_l_idx`	(`groupId`),
	KEY			`t_l_idx`	(`teacherId`),
	CONSTRAINT	`group_l`	FOREIGN KEY	(`groupId`)		REFERENCES	`groups`	(`groupId`)		ON DELETE RESTRICT	ON UPDATE RESTRICT,
	CONSTRAINT	`teach_l`	FOREIGN KEY	(`teacherId`)	REFERENCES	`teachers`	(`teacherId`)	ON DELETE SET NULL	ON UPDATE CASCADE
	);