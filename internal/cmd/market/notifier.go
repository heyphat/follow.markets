package market

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	tele "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"follow.markets/pkg/config"
	"follow.markets/pkg/log"
	"follow.markets/pkg/util"
)

type notifier struct {
	sync.Mutex
	connected bool
	bot       *tele.BotAPI
	notis     *sync.Map
	chatIDs   []int64

	// shared properties with other market participants
	logger       *log.Logger
	communicator *communicator
}

type notification struct {
	id       string
	lastSent time.Time
}

func newNotifier(participants *sharedParticipants, configs *config.Configs) (*notifier, error) {
	if configs == nil || participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants or configs")
	}
	var chatIDs []int64
	for _, id := range configs.Market.Notifier.Telegram.ChatIDs {
		if iid, err := strconv.Atoi(id); err != nil {
			return nil, err
		} else {
			chatIDs = append(chatIDs, int64(iid))
		}
	}
	bot, err := tele.NewBotAPI(configs.Market.Notifier.Telegram.BotToken)
	if err != nil {
		return nil, err
	}
	return &notifier{
		connected: false,
		bot:       bot,
		notis:     &sync.Map{},
		chatIDs:   chatIDs,

		logger:       participants.logger,
		communicator: participants.communicator,
	}, nil
}

// connect connects the notifier to other market participants py listening to
// decicated channels for the communication.
func (n *notifier) connect() {
	n.Lock()
	defer n.Unlock()
	if n.connected {
		return
	}
	go func() {
		for msg := range n.communicator.evaluator2Notifier {
			go n.processEvaluatorRequest(msg)
		}
	}()
	go func() {
		for msg := range n.communicator.trader2Notifier {
			go n.processTraderRequest(msg)
		}
	}()
	go n.await()
	n.connected = true
}

// await awaits for message from user to add chatID.
func (n *notifier) await() {
	updates := n.bot.GetUpdatesChan(tele.NewUpdate(0))
	for update := range updates {
		if update.Message == nil {
			continue
		}
		go n.addChatIDs([]int64{update.Message.Chat.ID})
		msg := tele.NewMessage(update.Message.Chat.ID, fmt.Sprintf("You're all set. Your chatID is %d.", update.Message.Chat.ID))
		msg.ReplyToMessageID = update.Message.MessageID
		n.bot.Send(msg)
	}
}

// isConnected returns true if the notifier is connected to the system, false otherwise.
func (n *notifier) isConnected() bool { return n.connected }

// addChatIDs adds new chat ids to the system if not initialized
func (n *notifier) addChatIDs(cids []int64) {
	n.Lock()
	defer n.Unlock()
	for _, cid := range cids {
		if !util.Int64SliceContains(n.chatIDs, cid) {
			n.chatIDs = append(n.chatIDs, cid)
		}
	}
}

// getNotifications returns a list of notifications the notifier has sent.
func (n *notifier) getNotifications() map[string]time.Time {
	out := make(map[string]time.Time)
	n.notis.Range(func(k, v interface{}) bool {
		out[k.(string)] = v.(notification).lastSent
		return true
	})
	return out
}

// this method processes requests from evaluator, it sends notifications to user
// based on the set of rules specified on the signal.
func (n *notifier) processEvaluatorRequest(msg *message) {
	if msg.request.what.runner == nil || msg.request.what.signal == nil {
		return
	}
	s := msg.request.what.signal
	r := msg.request.what.runner
	id := r.GetUniqueName() + "-" + s.Name
	mess := id + "\n" + s.Description()
	if s.IsOnetime() {
		n.notify(mess, s.OwnerID)
		return
	}
	if val, ok := n.notis.Load(id); !ok {
		n.notify(mess, s.OwnerID)
		n.notis.Store(id,
			notification{
				id:       id,
				lastSent: time.Now().Add(-time.Minute),
			})
	} else {
		if s.ShouldSend(val.(notification).lastSent) {
			n.notify(mess, s.OwnerID)
			n.notis.Store(id,
				notification{
					id:       id,
					lastSent: time.Now().Add(-time.Minute),
				})
		}
	}
}

// this method processes requests from trader, it sends notifications to user
// about trade activities.
func (n *notifier) processTraderRequest(msg *message) {
	if msg.request.what.dynamic == nil {
		return
	}
	mess := msg.request.what.dynamic.(string)
	if msg.request.what.signal == nil {
		n.notify(mess, nil)
		return
	}
	n.notify(mess, msg.request.what.signal.OwnerID)
}

// notify sends tele message to all chatIDs for a given content if the given `cid`
// is missing, otherwise only send to `cid`.
func (n *notifier) notify(content string, cid *int64) {
	if cid != nil {
		message := tele.NewMessage(*cid, content)
		n.bot.Send(message)
		return
	}
	for _, cid := range n.chatIDs {
		message := tele.NewMessage(cid, content)
		n.bot.Send(message)
	}
}

// generates a new log with the format for the notifier
func (n *notifier) newLog(name, message string) string {
	return fmt.Sprintf("[notifier] %s: %s", name, message)
}
