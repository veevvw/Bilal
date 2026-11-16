package bot

import (
	"ashokshau/tg-web/internal/commands"
	"ashokshau/tg-web/internal/db"
	"ashokshau/tg-web/internal/games"
	"ashokshau/tg-web/internal/utils"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
)

var gameEngine = games.New()
var rewardZone = time.FixedZone("Asia/Baghdad", 3*60*60)

func gamesPanel(chatID int64, enabled bool) (string, *td.ReplyMarkupInlineKeyboard) {
	t := func(ar, en string) string { return protectionText(chatID, ar, en) }
	status := t("مفعّلة", "Enabled")
	if !enabled {
		status = t("معطّلة بواسطة المشرف", "Disabled by administrator")
	}
	text := fmt.Sprintf(t("<b>ألعاب ShaHm</b>\n%s\n⭐ نقطة واحدة للإجابة الصحيحة.", "<b>ShaHm games</b>\n%s\n⭐ One point per correct answer."), status)
	return text, groupKeyboard(menuButton(t("🎮 الألعاب والجوائز", "🎮 Games & rewards"), "game:menu", td.ButtonStylePrimary{}),
		menuButton(t("🐱 كت", "🐱 Kat"), "game:kat", td.ButtonStyleSuccess{}), menuButton(t("🧠 مسابقة معلومات", "🧠 Trivia"), "game:quiz", td.ButtonStylePrimary{}),
		menuButton(t("🔢 تحدي الحساب", "🔢 Math challenge"), "game:math", td.ButtonStylePrimary{}), menuButton(t("🧩 ألغاز", "🧩 Riddles"), "game:riddle", td.ButtonStylePrimary{}),
		menuButton(t("💭 أسئلة", "💭 Questions"), "game:questions", td.ButtonStylePrimary{}), menuButton(t("⚡ تحديات", "⚡ Challenges"), "game:challenge", td.ButtonStylePrimary{}),
		menuButton(t("🎁 هديتي اليومية", "🎁 Daily gift"), "game:daily", td.ButtonStyleSuccess{}), menuButton(t("🎉 صندوق اليوم", "🎉 Daily box"), "game:box", td.ButtonStyleSuccess{}),
		menuButton(t("⭐ نقاطي ولقبي", "⭐ My points & title"), "game:points", td.ButtonStylePrimary{}), menuButton(t("🏆 ترتيب المجموعة", "🏆 Leaderboard"), "game:top", td.ButtonStylePrimary{}),
		menuButton(t("⚙ تفعيل/تعطيل الألعاب", "⚙ Toggle games"), "game:toggle", td.ButtonStyleDanger{}), menuButton(t("⛔ إنهاء الجولة للمشرف", "⛔ Stop round (admin)"), "game:stop", td.ButtonStyleDanger{}),
		menuButton(t("🛡 الحماية", "🛡 Protection"), "hub:protection", td.ButtonStyleSuccess{}), menuButton(t("⬅ الاوامر", "⬅ Commands"), "hub:home", td.ButtonStylePrimary{}))
}
func gameNavigation(chatID int64) *td.ReplyMarkupInlineKeyboard {
	return groupKeyboard(menuButton("✦ ShaHm ✦", "game:menu", td.ButtonStylePrimary{}), menuButton(protectionText(chatID, "🎮 الألعاب والجوائز", "🎮 Games & rewards"), "game:menu", td.ButtonStyleSuccess{}), menuButton(protectionText(chatID, "⬅ الاوامر", "⬅ Commands"), "hub:home", td.ButtonStylePrimary{}))
}
func gameMember(c *td.Client, chatID, userID int64) bool {
	if chatID >= 0 || userID <= 0 {
		return false
	}
	chat, chatErr := c.GetChat(chatID)
	if chatErr != nil || !utils.IsGroup(chat) {
		return false
	}
	member, err := c.GetChatMember(chatID, &td.MessageSenderUser{UserId: userID})
	if err != nil || member == nil {
		return false
	}
	switch status := member.Status.(type) {
	case *td.ChatMemberStatusCreator, *td.ChatMemberStatusAdministrator, *td.ChatMemberStatusMember:
		return true
	case *td.ChatMemberStatusRestricted:
		return status.IsMember
	}
	return false
}
func gamesMenuCallback(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	return handleGameAction(c, nil, cb, "menu")
}
func gamesCommand(action string) func(*td.Client, *td.Message) error {
	return func(c *td.Client, m *td.Message) error { return handleGameAction(c, m, nil, action) }
}
func gameCallback(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	return handleGameAction(c, nil, cb, strings.TrimPrefix(cb.DataString(), "game:"))
}

func gameTitle(points int, language string) string {
	if points >= 400 {
		return games.Text{"🏆 بطل المجموعة", "🏆 Group champion"}.In(language)
	}
	if points >= 150 {
		return games.Text{"🥇 نجم الألعاب", "🥇 Game star"}.In(language)
	}
	if points >= 50 {
		return games.Text{"🥈 لاعب مميز", "🥈 Skilled player"}.In(language)
	}
	return games.Text{"🌱 لاعب جديد", "🌱 New player"}.In(language)
}
func handleGameAction(c *td.Client, m *td.Message, cb *td.UpdateNewCallbackQuery, action string) error {
	var chatID, userID int64
	if m != nil {
		chatID, userID = m.ChatId, m.SenderID()
	} else {
		chatID, userID = cb.ChatId, cb.SenderUserId
	}
	t := func(ar, en string) string { return protectionText(chatID, ar, en) }
	notice := func(ar, en string) error {
		if cb != nil {
			return cb.Answer(c, t(ar, en), &td.CallbackAnswerOpts{ShowAlert: true})
		}
		_, err := utils.ReplyBotText(c, m, t(ar, en), nil)
		return err
	}
	if !gameMember(c, chatID, userID) {
		return notice("الألعاب للأعضاء داخل المجموعة. استخدم حسابك الشخصي للمشاركة.", "Games require current group membership and a personal account.")
	}
	enabled, err := db.Instance.GamesEnabled(chatID)
	if err != nil {
		return notice("تعذر تحميل إعدادات الألعاب. حاول مجدداً.", "Unable to load game settings. Please retry.")
	}
	activeRound := ""
	show := func(text string, keyboard *td.ReplyMarkupInlineKeyboard) error {
		if cb != nil {
			_ = cb.Answer(c, "", nil)
			_, err := utils.EditControlMessage(c, chatID, cb.MessageId, text, &td.EditMessageTextOpts{ReplyMarkup: keyboard})
			if err == nil {
				gameEngine.UnbindMessage(chatID, cb.MessageId)
				if activeRound != "" {
					gameEngine.BindMessage(chatID, activeRound, cb.MessageId)
				}
			}
			return err
		}
		message, err := utils.ReplyControlMessage(c, m, text, &td.SendMessageOpts{ReplyMarkup: keyboard})
		if err == nil && message != nil && activeRound != "" {
			gameEngine.BindMessage(chatID, activeRound, message.Id)
		}
		return err
	}
	if action == "toggle" || action == "stop" {
		if protectionAccess(c, chatID, userID, false) != nil {
			return notice("هذا الزر للمشرفين فقط.", "This control is for administrators only.")
		}
		if action == "toggle" {
			if err = db.Instance.SetGamesEnabled(chatID, !enabled); err != nil {
				return notice("تعذر حفظ الإعداد.", "Unable to save setting.")
			}
			enabled = !enabled
		}
		if action == "stop" || !enabled {
			gameEngine.Stop(chatID)
		}
		text, k := gamesPanel(chatID, enabled)
		return show(text, k)
	}
	if action == "menu" {
		text, k := gamesPanel(chatID, enabled)
		return show(text, k)
	}
	if !enabled {
		return notice("الألعاب معطّلة في هذه المجموعة. يمكن للمشرف تفعيلها من اللوحة.", "Games are disabled here. An administrator can enable them from the panel.")
	}
	name := t("عضو", "Member")
	if user, e := c.GetUser(userID); e == nil {
		name = user.FirstName
	}
	now := time.Now()
	day := now.In(rewardZone).Format("2006-01-02")
	switch action {
	case "kat", "questions", "challenge":
		prompt, e := gameEngine.PromptKind(chatID, userID, action, now)
		if e != nil {
			return notice("انتظر قليلاً قبل السؤال التالي.", "Please wait before the next prompt.")
		}
		promptText := prompt.In(chatLanguage(chatID))
		title := t("🐱 كت", "🐱 Kat")
		if action == "questions" {
			title = t("💭 أسئلة", "💭 Questions")
			promptText = strings.TrimPrefix(strings.TrimPrefix(promptText, "كت: "), "Kat: ")
		}
		if action == "challenge" {
			title = t("⚡ تحديات", "⚡ Challenges")
		}
		text := html.EscapeString(promptText)
		if action == "kat" {
			return show(text, sourceChannelKeyboard(chatID, false))
		}
		keyboard := groupKeyboard(menuButton(title, "game:"+action, td.ButtonStylePrimary{}), menuButton(t("🔄 التالي", "🔄 Next"), "game:"+action, td.ButtonStyleSuccess{}), menuButton(t("🎮 الألعاب", "🎮 Games"), "game:menu", td.ButtonStylePrimary{}), menuButton(t("⬅ الاوامر", "⬅ Commands"), "hub:home", td.ButtonStylePrimary{}))
		return show(text, keyboard)
	case "quiz", "math", "riddle":
		round, e := gameEngine.Start(chatID, action, now)
		if e != nil && !errors.Is(e, games.ErrActive) {
			return notice("تعذر بدء الجولة.", "Unable to start the round.")
		}
		activeRound = round.ID
		text := html.EscapeString(round.Question.Title.In(chatLanguage(chatID))) + "\n\n" + t("⭐ نقطة واحدة", "⭐ 1 point")
		buttons := []td.InlineKeyboardButton{}
		for i, option := range round.Question.Options {
			buttons = append(buttons, menuButton(option.In(chatLanguage(chatID)), fmt.Sprintf("game:answer:%s:%d", round.ID, i), td.ButtonStylePrimary{}))
		}
		buttons = append(buttons, menuButton(t("🎮 الألعاب", "🎮 Games"), "game:menu", td.ButtonStyleSuccess{}), menuButton(t("⬅ الاوامر", "⬅ Commands"), "hub:home", td.ButtonStylePrimary{}))
		return show(text, groupKeyboard(menuButton("🧠 ShaHm", "game:menu", td.ButtonStylePrimary{}), buttons...))
	case "daily", "box", "katclaim":
		points := 20
		kind := "daily"
		if action == "box" {
			points = 5 + games.Random(11)
			kind = "box"
		}
		if action == "katclaim" {
			points = 1
			kind = "kat"
		}
		awarded, e := db.Instance.AwardGamePoints(chatID, userID, name, kind+":"+day, points, false)
		if e != nil {
			return notice("تعذر حفظ النقاط. حاول مجدداً.", "Unable to save points. Please retry.")
		}
		if !awarded {
			return notice("استلمت هذه المكافأة اليوم. تتجدد عند منتصف الليل بتوقيت بغداد.", "You already claimed this reward today. It resets at midnight Baghdad time.")
		}
		return notice(fmt.Sprintf("⭐ +%d", points), fmt.Sprintf("⭐ +%d", points))
	case "points":
		player, e := db.Instance.GetGamePlayer(chatID, userID)
		if e != nil {
			return notice("تعذر تحميل نقاطك.", "Unable to load your points.")
		}
		// Personal balances use an alert, so another member's panel is not overwritten.
		return notice(fmt.Sprintf("%s\nنقاطك: %d\nانتصاراتك: %d\nلقبك: %s", name, player.Points, player.Wins, gameTitle(player.Points, "ar")), fmt.Sprintf("%s\nPoints: %d\nWins: %d\nTitle: %s", name, player.Points, player.Wins, gameTitle(player.Points, "en")))
	case "top":
		players, e := db.Instance.GameLeaderboard(chatID)
		if e != nil {
			return notice("تعذر تحميل الترتيب.", "Unable to load the leaderboard.")
		}
		text := t("<b>🏆 ترتيب المجموعة</b>", "<b>🏆 Group leaderboard</b>")
		for i, p := range players {
			text += fmt.Sprintf("\n%d. %s — %d ⭐ · %s", i+1, html.EscapeString(p.Name), p.Points, gameTitle(p.Points, chatLanguage(chatID)))
		}
		if len(players) == 0 {
			text += "\n" + t("ابدأ اللعب لتظهر أول نتيجة.", "Play to record the first score.")
		}
		return show(text, gameNavigation(chatID))
	default:
		parts := strings.Split(action, ":")
		var choice int
		var roundID string
		if m != nil && len(parts) == 2 && parts[0] == "textanswer" {
			round, ok := gameEngine.ReplyRound(chatID, m.ReplyToMessageID())
			if !ok || round.ID != parts[1] {
				return nil
			}
			roundID = round.ID
			choice = round.Question.Choice(m.Text())
		} else {
			if len(parts) != 3 || parts[0] != "answer" || cb == nil {
				return nil
			}
			n, e := strconv.Atoi(parts[2])
			if e != nil {
				return nil
			}
			choice = n
			roundID = parts[1]
		}
		round, e := gameEngine.Answer(chatID, roundID, userID, choice, now)
		switch {
		case errors.Is(e, games.ErrExpired):
			return notice("انتهت الجولة أو أوقفها المشرف. افتح مسابقة جديدة.", "The round expired or was stopped. Start a new quiz.")
		case errors.Is(e, games.ErrWrong):
			return notice("إجابة غير صحيحة.", "Incorrect answer.")
		case errors.Is(e, games.ErrAttempted):
			return notice("استخدمت محاولتك في هذه الجولة.", "You already used your attempt this round.")
		case errors.Is(e, games.ErrWon):
			return notice("سبقك عضو إلى الإجابة الصحيحة.", "Another member already won this round.")
		case e != nil:
			return notice("تعذر التحقق من الإجابة.", "Unable to check the answer.")
		}
		awarded, e := db.Instance.AwardGamePoints(chatID, userID, name, "win:"+round.ID, games.AnswerReward, true)
		if e != nil {
			return notice("تعذر حفظ النقطة. حاول مجدداً.", "Unable to save your point. Please retry.")
		}
		gameEngine.Finish(chatID, round.ID, userID)
		if !awarded {
			return notice("تم احتساب جائزتك سابقاً.", "Your prize was already recorded.")
		}
		text := fmt.Sprintf(t("<b>🏆 %s</b>\n%s\n⭐ +1", "<b>🏆 %s</b>\n%s\n⭐ +1"), html.EscapeString(name), html.EscapeString(round.Question.Options[round.Question.Correct].In(chatLanguage(chatID))))
		return show(text, gameNavigation(chatID))
	}
}

func gameReplyFilter(m *td.Message) bool {
	if m == nil || m.IsOutgoing || m.EditDate > 0 || m.IsChannelPost || strings.TrimSpace(m.Text()) == "" {
		return false
	}
	reply, ok := m.ReplyTo.(*td.MessageReplyToMessage)
	if !ok || reply.MessageId == 0 || (reply.ChatId != 0 && reply.ChatId != m.ChatId) {
		return false
	}
	if _, command := commands.Resolve(m.Text()); command || strings.HasPrefix(strings.TrimSpace(m.Text()), "/") {
		return false
	}
	_, ok = gameEngine.ReplyRound(m.ChatId, reply.MessageId)
	return ok
}
func gameReplyHandler(c *td.Client, m *td.Message) error {
	if !requireSourceSubscription(c, m) {
		return td.EndGroups
	}
	round, ok := gameEngine.ReplyRound(m.ChatId, m.ReplyToMessageID())
	if !ok {
		return td.ContinueGroups
	}
	if err := handleGameAction(c, m, nil, "textanswer:"+round.ID); err != nil {
		return err
	}
	return td.EndGroups
}
