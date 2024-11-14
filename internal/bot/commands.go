package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shrimpsizemoose/trekker/logger"
)

const (
	studentHelp = `Доступные команды:
/token - Получить токен для доступа к API
/help - Показать это сообщение`

	adminHelp = `Доступные команды:
/token - Получить токен для доступа к API
/lab add <course> <lab> --score <score> --deadline <date> - Добавить лабораторную
/lab list <course> - Список лабораторных работ
/override set <course> <student> <lab> --score <score> --reason <reason> - Установить оценку вручную
/override list <course> - Список текущих оверрайдов
/help - Показать это сообщение

Примеры:
/lab add DE15 01s --score 10 --deadline "2024-12-01"
/lab list DE15
/override set DE15 student.name 01s --score 8 --reason "Late submission accepted"
/override list DE15`
)

type commandHandler func(*tgbotapi.Message) error

func (b *Bot) routeStudentCommands(cmd string) (commandHandler, bool) {
	commands := map[string]commandHandler{
		"start": b.handleStart,
		"token": b.handleToken,
		"help":  b.handleHelp,
	}
	handler, found := commands[cmd]
	return handler, found
}

func (b *Bot) routeAdminCommands(cmd string) (commandHandler, bool) {
	commands := map[string]commandHandler{
		"lab":      b.handleLab,
		"override": b.handleOverride,
	}
	handler, found := commands[cmd]
	return handler, found
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if !msg.IsCommand() {
		b.sendHelp(msg.Chat.ID)
		return
	}

	cmd := msg.Command()

	if handler, ok := b.routeStudentCommands(cmd); ok {
		if err := handler(msg); err != nil {
			logger.Error.Printf("Command error: %v", err)
			b.sendMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err))
		}
		return
	}

	if b.admins[msg.From.ID] {
		if handler, ok := b.routeAdminCommands(cmd); ok {
			if err := handler(msg); err != nil {
				logger.Error.Printf("Command error: %v", err)
				b.sendMessage(msg.Chat.ID, fmt.Sprintf("Error: %v", err))
			}
		}
		return
	}

	b.sendHelp(msg.Chat.ID)
}

func (b *Bot) handleHelp(msg *tgbotapi.Message) error {
	var text string
	if b.admins[msg.From.ID] {
		text = adminHelp
	} else {
		text = studentHelp
	}

	return b.sendMessage(msg.Chat.ID, text)
}

func (b *Bot) sendHelp(chatID int64) error {
	return b.sendMessage(chatID, "Используйте команды для взаимодействия с ботом. Отправьте /help для списка команд.")
}

func (b *Bot) handleStart(msg *tgbotapi.Message) error {
	text := "Привет! Я помогу тебе с курсом.\n\n"
	if b.admins[msg.From.ID] {
		text += "Ты администратор курса. Используй /help для списка команд."
	} else {
		text += "Используй /token чтобы получить токен."
	}

	return b.sendMessage(msg.Chat.ID, text)
}

func (b *Bot) handleToken(msg *tgbotapi.Message) error {
	// TODO: Генерация и сохранение токена в Redis
	return nil
}

func (b *Bot) handleLab(msg *tgbotapi.Message) error {
	// Пример: /lab add DE15 lab01 --score 10 --deadline "2024-12-01"
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 4 {
		return fmt.Errorf("usage: /lab add <course> <lab> --score <score> --deadline <deadline>")
	}

	// TODO: Парсинг аргументов и создание LabScore
	return nil
}

func (b *Bot) handleOverride(msg *tgbotapi.Message) error {
	// Пример: /override set DE15 student.name lab01 --score 8 --reason "Late submission"
	return nil
}

func (b *Bot) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	return err
}
