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