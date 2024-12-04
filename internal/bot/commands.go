package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shrimpsizemoose/trekker/logger"

	"github.com/shrimpsizemoose/kanelbulle/internal/models"
)

const (
	operationTimeout = 5 * time.Second

	studentHelp = `Доступные команды:
/token - Получить токен для доступа к API
/help - Показать это сообщение`

	adminHelp = `Доступные команды:
/token - Получить токен для доступа к API
/lab add <course> <lab> score <score> deadline <date> -- Добавить лабораторную
/lab list <course> - Список лабораторных работ
/override set <course> <student> <lab> score <score> reason <reason> - Установить оценку вручную
/override list <course> - Список текущих оверрайдов
/new_course COURSE_CODE +список пар @tg_username и student.id по одной в каждой строке
/set_course <course> [comment] - Привязать чат к какому-то курсу
/map_student @username <student.name> - Привязать телеграмный айдишник к student.id
/help - Показать это сообщение

Примеры:
/lab add DE15 01s score 10 deadline "2024-12-01"
/lab list DE15
/override set DE15 01s student.name score 8 reason "Late submission accepted"
/override list DE15
/map_student @karkarkar kaggi.kar
/set_course DE15 "Дамокловы Экивоки 14+"
`
)

type commandHandler func(*tgbotapi.Message) error

func (b *Bot) routeStudentCommands(cmd string) (commandHandler, bool) {
	commands := map[string]commandHandler{
		"start": b.handleStart,
		"token": b.handleTokenCommand,
		"help":  b.handleHelp,
	}
	handler, found := commands[cmd]
	return handler, found
}

func (b *Bot) routeAdminCommands(cmd string) (commandHandler, bool) {
	commands := map[string]commandHandler{
		"lab":         b.handleLab,
		"override":    b.handleOverride,
		"set_course":  b.handleSetCourseCommand,
		"map_student": b.handleMapStudentCommand,
		"new_course":  b.handleNewCourseCommand,
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
	if msg.Chat.Type != "private" {
		return nil
	}

	var response strings.Builder
	// TODO: add emoji
	response.WriteString("Привет!\n\n")

	if b.admins[msg.From.ID] {
		b.sendMessage(msg.Chat.ID, "Ты администратор бота. Команды:\n"+adminHelp)
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	info, err := b.tokenManager.FetchStudentCourseInfo(ctx, msg.From.UserName)
	if err == nil {
		response.WriteString(fmt.Sprintf("Вы студент курса `%s`.\n\n%s", info.Course, studentHelp))
	}

	return b.sendMessage(msg.Chat.ID, response.String())

}

func (b *Bot) handleTokenCommand(msg *tgbotapi.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	if msg.Chat.Type != "private" {
		// delMsg := tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID)
		// if _, err := b.api.Request(delMsg); err != nil {
		// 	return fmt.Errorf("failed to delete message: %w", err)
		// }
		// return fmt.Errorf("Команда /token имеет смысл только в приватном чате")

		warnMsg := tgbotapi.NewMessage(msg.Chat.ID, "Команда /token имеет смысл только в приватном чате")
		warnMsg.ReplyToMessageID = msg.MessageID
		reply, err := b.api.Send(warnMsg)
		if err == nil {
			go func() {
				time.Sleep(12 * time.Second)
				delWarn := tgbotapi.NewDeleteMessage(msg.Chat.ID, reply.MessageID)
				if _, err := b.api.Request(delWarn); err != nil {
					logger.Error.Printf("Failed to delete warning message: %v", err)
				}
				delMsg := tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID)
				if _, err := b.api.Request(delMsg); err != nil {
					logger.Error.Printf("failed to delete message: %v", err)
				}
			}()
		}
		return nil
	}

	info, err := b.tokenManager.FetchStudentCourseInfo(ctx, msg.From.UserName)
	if err != nil {
		return fmt.Errorf("Не признал: %w", err)
	}

	// mapping, err := b.tokenManager.FetchCourseMappingByChatID(ctx, msg.Chat.ID)
	// if err != nil {
	// 	return fmt.Errorf("failed to determine course: %w", err)
	// }

	// studentID, err := b.tokenManager.FetchStudentIDByTelegram(ctx, mapping.Course, msg.From.UserName)
	// if err != nil {
	// 	return fmt.Errorf("failed to get student ID: %w", err)
	// }

	tokenInfo, isNewToken, err := b.tokenManager.FetchOrCreateStudentToken(ctx, info.Course, info.StudentID)
	if err != nil {
		return fmt.Errorf("failed to get/create token: %w", err)
	}

	if isNewToken {
		go b.notifyAdminsAboutNewToken(info.Course, info.StudentID, tokenInfo.Token)
	}

	text := fmt.Sprintf(
		"🧩 %s\nStudent: `%s`\nToken: `%s`",
		info.Course,
		info.StudentID,
		tokenInfo.Token,
	)

	if err := b.sendMessageMarkdown(msg.From.ID, text); err != nil {
		return fmt.Errorf("failed to send token: %w", err)
	}

	return nil
}

func (b *Bot) notifyAdminsAboutNewToken(course, student, token string) {
	message := fmt.Sprintf(
		"🔐 New token created\nCourse: %s\nStudent: `%s`\nToken: `%s`",
		course,
		student,
		token,
	)

	for _, adminID := range b.config.Bot.AdminIDs {
		go func(id int64) {
			if err := b.sendMessageMarkdown(id, message); err != nil {
				logger.Error.Printf("Failed to notify admin %d: %v", id, err)
			}
		}(adminID)
	}
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

func isActiveMember(member tgbotapi.ChatMember) bool {
	// Не активен, если:
	// - покинул чат
	// - был исключён
	if member.HasLeft() || member.WasKicked() {
		return false
	}
	// Активен, если:
	// - создатель
	// - администратор
	// - обычный участник
	// - с ограниченными правами
	return member.Status == "member" ||
		member.Status == "administrator" ||
		member.Status == "creator" ||
		member.Status == "restricted"
}

// handleSetCourseCommand привязывает чат к курсу чтобы отслеживать join/leave события
func (b *Bot) handleSetCourseCommand(msg *tgbotapi.Message) error {
	if msg.Chat.Type == "private" {
		return fmt.Errorf("Эту команду имеет смысл выполнять только в групповых чатах")
	}

	args := strings.Fields(msg.CommandArguments())
	if len(args) < 1 {
		return fmt.Errorf("использование: /set_course <course> [comment]")
	}

	course := strings.TrimSpace(args[0])
	comment := ""
	if len(args) > 1 {
		comment = strings.Trim(strings.TrimSpace(args[1]), `"'`)
	}
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	// FIXME: сейчас это не так работает, маппинг на студента приходит другой
	mapping := &models.ChatCourseMapping{
		Course:          course,
		Name:            msg.Chat.Title,
		Comment:         comment,
		AssociationTime: time.Now().UTC(),
		RegisteredBy:    msg.From.ID,
	}

	if err := b.tokenManager.AssociateChatWithCourse(ctx, msg.Chat.ID, mapping); err != nil {
		return fmt.Errorf("Не смог сассоциировать курс с этим чатом: %w", err)
	}

	students, err := b.tokenManager.FetchCourseStudents(ctx, course)
	if err != nil {
		return fmt.Errorf("failed to fetch course data: %w", err)
	}
	if len(students) == 0 {
		return fmt.Errorf("course %s not found", course)
	}

	if err := b.sendMessage(msg.Chat.ID, "⏳ Processing group members..."); err != nil {
		logger.Error.Printf("Failed to send status message: %v", err)
	}

	b.notifyAdminsAboutNewCourseChatAssociation(msg.Chat.Title, course, comment, msg.From.UserName)

	statusMsg := fmt.Sprintf(
		"✅ Chat successfully associated with course %s\n"+
			"🔍 Verifying student memberships...",
		course,
	)
	if err := b.sendMessage(msg.Chat.ID, statusMsg); err != nil {
		logger.Error.Printf("Failed to send status message: %v", err)
	}

	var presentStudents, missingStudents []string

	for username := range students {
		// for username, userID := range students {
		member, err := b.api.GetChatMember(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: msg.Chat.ID,
				UserID: msg.From.ID, // userID,
			},
		})

		if err != nil {
			logger.Error.Printf("Failed to check member %s: %v", username, err)
			missingStudents = append(missingStudents, "@"+username+" (error)")
			continue
		}

		if isActiveMember(member) {
			presentStudents = append(
				presentStudents,
				fmt.Sprintf("@%s (%s)", username, member.Status),
			)
		} else {
			missingStudents = append(
				missingStudents,
				fmt.Sprintf("@%s (%s)", username, member.Status),
			)

		}
	}

	report := fmt.Sprintf(
		"✅ Chat associated with course %s\n\n"+
			"📊 Student status:\n"+
			"✅ Present: %d students\n"+
			"❌ Missing: %d students\n\n",
		course,
		len(presentStudents),
		len(missingStudents),
	)

	if len(presentStudents) > 0 {
		report += "\nПосчитал:\n" + strings.Join(missingStudents, "\n")
	}

	if len(missingStudents) > 0 {
		report += "\n\nMissig/inactive:\n" + strings.Join(missingStudents, "\n")
	}

	if err := b.editMessage(msg.Chat.ID, msg.MessageID, report); err != nil {
		logger.Error.Printf("Failed to update status message: %v", err)
	}

	return nil
}

func (b *Bot) notifyAdminsAboutNewCourseChatAssociation(chatTitle, course, comment, username string) {
	message := fmt.Sprintf(
		"🔄 Course association updated\n"+
			"Chat: %s\n"+
			"Course: %s\n"+
			"Comment: %s\n"+
			"Updated by: @%s",
		chatTitle,
		course,
		comment,
		username,
	)

	for _, adminID := range b.config.Bot.AdminIDs {
		go func(id int64) {
			if err := b.sendMessage(id, message); err != nil {
				logger.Error.Printf("Failed to notify admin %d: %v", id, err)
			}
		}(adminID)
	}
}

func (b *Bot) handleMapStudentCommand(msg *tgbotapi.Message) error {
	args := strings.Fields(msg.CommandArguments())
	if len(args) != 2 {
		return b.sendMessage(msg.Chat.ID, "Использование:\n"+
			"/map_student @username student.name - ассоциировать студента @username со student.name")
	}

	tgUsername := strings.TrimPrefix(args[0], "@")
	if tgUsername == "" {
		return fmt.Errorf("invalid telegram username")
	}

	studentID := args[1]
	if !strings.Contains(studentID, ".") {
		return fmt.Errorf("Неправильный формат studentID, должно быть: firstname.lastname")
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	mapping, err := b.tokenManager.FetchCourseMappingByChatID(ctx, msg.Chat.ID)
	if err != nil {
		return fmt.Errorf("этот чат не сассоциирован ни с каким курсом: %v. Попросите админа сделать /set_course", err)
	}

	if err := b.tokenManager.SaveStudentTelegramMapping(ctx, mapping.Course, tgUsername, studentID); err != nil {
		return fmt.Errorf("failed to save student mapping: %w", err)
	}

	tokenInfo, isNewToken, err := b.tokenManager.FetchOrCreateStudentToken(ctx, mapping.Course, studentID)
	if err != nil {
		logger.Error.Printf("Failed to check existing token: %v", err)
	}

	var tokenStatus string
	if err == nil {
		if isNewToken {
			tokenStatus = "\nНовый токен сгенерирован тоже."
		} else {
			tokenStatus = fmt.Sprintf(
				"\nУ студента уже есть токен (запрошен, раз: %d, последний: %s)",
				tokenInfo.RequestCount,
				tokenInfo.LastRequestTime.Format("2006-01-02 15:04:05 MST"),
			)
		}
	}

	response := fmt.Sprintf(
		"✅ Student mapping created\n"+
			"Course: %s\n"+
			"Telegram: @%s\n"+
			"Student ID: %s%s",
		mapping.Course,
		tgUsername,
		studentID,
		tokenStatus,
	)

	if err := b.sendMessage(msg.Chat.ID, response); err != nil {
		return fmt.Errorf("failed to send confirmation message: %w", err)
	}

	notificationMsg := fmt.Sprintf(
		"👤 New student mapping\n"+
			"Course: %s\n"+
			"Telegram: @%s\n"+
			"Student ID: %s\n"+
			"Mapped by: @%s",
		mapping.Course,
		tgUsername,
		studentID,
		msg.From.UserName,
	)

	for _, adminID := range b.config.Bot.AdminIDs {
		if adminID != msg.From.ID {
			go func(id int64) {
				if err := b.sendMessage(id, notificationMsg); err != nil {
					logger.Error.Printf("Failed to notify admin %d: %v", id, err)
				}
			}(adminID)
		}
	}

	return nil
}

func (b *Bot) handleNewCourseCommand(msg *tgbotapi.Message) error {
	lines := strings.Split(msg.Text, "\n")
	if len(lines) < 2 {
		return fmt.Errorf("Использование:\n/new_course COURSE_CODE\n@username1 student1.name\n@username2 student2.name")
	}

	course := strings.TrimSpace(strings.TrimPrefix(lines[0], "/new_course"))
	if course == "" {
		return fmt.Errorf("Нужен 'код' курса")
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	var processedStudents []string
	var errors []string

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			errors = append(errors, fmt.Sprintf("Invalid line format: %s", line))
			continue
		}

		tgUsername := strings.TrimPrefix(parts[0], "@")
		studentID := parts[1]

		if !strings.Contains(studentID, ".") {
			errors = append(errors, fmt.Sprintf("Invalid student ID format: %s", studentID))
			continue
		}

		if err := b.tokenManager.SaveStudentTelegramMapping(ctx, course, tgUsername, studentID); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to map @%s to %s: %v", tgUsername, studentID, err))
			continue
		}

		courseInfo := &models.StudentCourseInfo{
			StudentID: studentID,
			Course:    course,
		}
		if err := b.tokenManager.SaveStudentCourseInfo(ctx, tgUsername, courseInfo); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to associate %s with course: %v", studentID, err))
			continue
		}

		processedStudents = append(processedStudents, fmt.Sprintf("@%s -> %s", tgUsername, studentID))
	}

	response := formatNewCourseResponse(course, processedStudents, errors)
	return b.sendMessage(msg.Chat.ID, response)
}

func formatNewCourseResponse(course string, processed []string, errors []string) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("📚 Course %s setup", course))

	if len(processed) > 0 {
		parts = append(parts, "\n✅ Mapped students:", strings.Join(processed, "\n"))
	}
	if len(errors) > 0 {
		parts = append(parts, "\n⚠️ Errors:", strings.Join(errors, "\n"))
	}

	// parts = append(parts, "\nℹ️ Use /set_course in target chat to activate the course")

	return strings.Join(parts, "\n")
}

func (b *Bot) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) sendMessageMarkdown(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) editMessage(chatID int64, msgID int, editText string) error {
	edit := tgbotapi.NewEditMessageText(chatID, msgID, editText)
	_, err := b.api.Send(edit)
	return err
}
