-- Сведения о пользователях бота Telegram
CREATE TABLE IF NOT EXISTS `tg_users` (
	`l9Id`		bigint		NOT NULL,
	-- Идентификатор пользователя системы

	`tgId`		bigint		NOT NULL,
	-- ID пользователя в Telegram

	`name`		TEXT,
	-- (optional) Имя пользователя в Telegram

	`posTag`	varchar(30)	DEFAULT		'not_started',
	-- Позиция пользователя в диалоге с ботом:
	-- (default) not_started - только что вступил в диалог
	-- add - добавляет группу
	-- conf_{groupId} - пользователь на стадии подтверждения выбранной группы
	-- ready - готов к работе

	PRIMARY KEY				(`l9Id`),
	CONSTRAINT	`l9_tg`		FOREIGN KEY	(`l9Id`)		REFERENCES	`users`		(`l9Id`)	ON DELETE CASCADE	ON UPDATE CASCADE
	);