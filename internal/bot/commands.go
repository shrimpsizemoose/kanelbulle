package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shrimpsizemoose/trekker/logger"

	"github.com/shrimpsizemoose/kanelbulle/internal/models"
)

const (
	studentHelp = `Доступные команды:
/token - Получить токен для доступа к API
/help - Показать это сообщение`

	adminHelp = `Доступные команды:
/token - Получить токен для доступа к API
/lab add <course> <lab> score <score> deadline <date> -- Добавить лабораторную
/lab list <course> - Список лабораторных работ
/override set <course> <student> <lab> score <score> reason <reason> - Установить оценку вручную
/override list <course> - Список текущих оверрайдов
/help - Показать это сообщение

Примеры:
/lab add DE15 01s score 10 deadline "2024-12-01"
/lab list DE15
/override set DE15 01s student.name score 8 reason "Late submission accepted"
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
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 1 {
		return b.sendMessage(msg.Chat.ID, "Использование:\n"+
			"/lab add <course> <lab> score <score> deadline <date> - Добавить лабораторную\n"+
			"/lab list <course> - Показать список лабораторных")
	}

	switch args[0] {
	case "add":
		return b.handleLabAdd(msg.Chat.ID, args[1:])
	case "list":
		if len(args) < 2 {
			return fmt.Errorf("укажи курс: /lab list DE15")
		}
		return b.handleLabList(msg.Chat.ID, args[1])
	default:
		return fmt.Errorf("неизвестная подкоманда: %s", args[0])
	}
}

func (b *Bot) handleLabAdd(chatID int64, args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("использование: add <course> <lab> score <score> deadline <date>")
	}

	course := args[0]
	lab := args[1]

	var score int
	var deadline time.Time
	var err error

	for i := 2; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return fmt.Errorf("пропущено значение для %s", args[i])
		}

		switch args[i] {
		case "score":
			score, err = strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("некорректная оценка: %v", err)
			}
		case "deadline":
			deadline, err = time.Parse("2006-01-02", args[i+1])
			if err != nil {
				return fmt.Errorf("некорректная дата (используйте YYYY-MM-DD): %v", err)
			}
			deadline = time.Date(
				deadline.Year(),
				deadline.Month(),
				deadline.Day(),
				23, 59, 59, 0,
				deadline.Location(),
			)
		default:
			return fmt.Errorf("неизвестный параметр: %s", args[i])
		}
	}

	labScore := models.LabScore{
		Lab:       lab,
		Course:    course,
		BaseScore: score,
		Deadline:  deadline.Unix(),
	}

	existing, err := b.store.GetLabScore(course, lab)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования лабы %s/%s: %v", course, lab, err)
	}

	err = b.store.CreateLabScore(labScore)
	if err != nil {
		return fmt.Errorf("ошибка сохранения: %v", err)
	}

	action := "добавлена"
	if existing != nil {
		action = "обновлена"
	}

	return b.sendMessage(chatID, fmt.Sprintf("✅ Лабораторная %s для курса %s %s:\n"+
		"Баллы: %d\n"+
		"Дедлайн: %s %s",
		lab,
		course,
		action,
		score,
		deadline.Format("2006-01-02 15:04"),
		deadline.Location().String(),
	))
}

func (b *Bot) handleLabList(chatID int64, course string) error {
	labs, err := b.store.ListLabScores(course)
	if err != nil {
		return fmt.Errorf("ошибка получения списка лаб: %v", err)
	}

	if len(labs) == 0 {
		return b.sendMessage(chatID, "Лабораторные работы не найдены")
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Лабораторные работы курса %s:\n\n", course))
	for _, lab := range labs {
		deadline := time.Unix(lab.Deadline, 0)
		msg.WriteString(fmt.Sprintf("📝 %s (баллы: %d)\n"+
			"📅 %s UTC\n\n",
			lab.Lab,
			lab.BaseScore,
			deadline.UTC().Format("2006-Jan-02 Mon 15:04"),
		))
	}

	return b.sendMessage(chatID, msg.String())
}

func (b *Bot) handleOverride(msg *tgbotapi.Message) error {
	// Пример: /override set DE15 student.name lab01 --score 8 --reason "Late submission"
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 1 {
		return b.sendMessage(msg.Chat.ID, "Использование:\n"+
			"/override set <course> <lab> <student> <lab> score <score> reason <reason> - Установить оценку вручную\n"+
			"/override list <course> - Список текущих оверрайдов для конкретног окурса")
	}

	switch args[0] {
	case "set":
		return b.handleOverrideSet(msg.Chat.ID, args[1:])
	case "list":
		if len(args) < 2 {
			return fmt.Errorf("укажи курс: /override list DE15")
		}
		return b.handleOverrideList(msg.Chat.ID, args[1])
	default:
		return fmt.Errorf("неизвестная подкоманда: %s", args[0])
	}
}

func (b *Bot) handleOverrideSet(chatID int64, args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("использование: set <course> <lab> <student> score <score> reason <reason>")
	}

	course := args[0]
	lab := args[1]
	student := args[2]
	score, err := strconv.Atoi(args[4])
	if err != nil {
		return fmt.Errorf("некорректная оценка: %v", err)
	}
	reason := strings.Join(args[6+1:], " ")

	scoreOverride := models.ScoreOverride{
		Student: student,
		Lab:     lab,
		Score:   score,
		Course:  course,
		Reason:  reason,
	}

	existing, err := b.store.GetScoreOverride(course, lab, student)
	if err != nil {
		return fmt.Errorf("ошибка проверки существования оверрайда %s/%s/%s: %v", course, lab, student, err)
	}

	err = b.store.CreateScoreOverride(scoreOverride)
	if err != nil {
		return fmt.Errorf("ошибка сохранения: %v", err)
	}

	action := "добавлен"
	if existing != nil {
		action = "обновлён"
	}

	return b.sendMessage(chatID, fmt.Sprintf("✅ Оверрайд для студента %s/%s/%s %s:\n"+
		"Баллы: %d\n"+
		"Причина: %s",
		action,
		course, lab, student,
		score,
		reason,
	))
}

func (b *Bot) handleOverrideList(chatID int64, course string) error {
	overrides, err := b.store.ListCourseScoreOverrides(course)
	if err != nil {
		return fmt.Errorf("ошибка получения списка оверрайдов: %v", err)
	}

	if len(overrides) == 0 {
		return b.sendMessage(chatID, "Оверрайды не найдены")
	}

	baseScores := map[string]int{}
	scores, err := b.store.ListLabScores(course)
	if err != nil {
		return fmt.Errorf("ошибка при запросе скоров лаб: %v", err)
	}
	for _, score := range scores {
		baseScores[score.Lab] = score.BaseScore
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Оверрайды курса %s:\n\n", course))
	for _, override := range overrides {
		msg.WriteString(fmt.Sprintf(
			"👉🏻 %s: за лабу %s ставим %d\nБазовый скор за эту лабу: %d\n❓(%s)\n\n",
			override.Student,
			override.Lab,
			override.Score,
			baseScores[override.Lab],
			override.Reason,
		))
	}

	return b.sendMessage(chatID, msg.String())
}

func (b *Bot) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	return err
}
