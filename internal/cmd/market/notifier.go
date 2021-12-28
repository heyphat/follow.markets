package market

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	tele "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"follow.markets/internal/pkg/strategy"
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
	provider     *provider
	communicator *communicator
}

type nmember struct {
	id       string
	lastSent time.Time
}

func newNotifier(participants *sharedParticipants, configs *config.Configs) (*notifier, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
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
		provider:     participants.provider,
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
	n.connected = true
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

func (n *notifier) processEvaluatorRequest(msg *message) {
	s := msg.request.what.(*strategy.Signal)
	mess := msg.request.id + "\n" + s.Description()
	if s.IsOnetime() {
		n.notify(mess)
		return
	}
	if val, ok := n.notis.Load(msg.request.id); !ok {
		n.notify(mess)
		n.notis.Store(msg.request.id,
			nmember{id: msg.request.id, lastSent: time.Now().Add(-time.Minute)})
	} else {
		if s.ShouldSend(val.(nmember).lastSent) {
			n.notify(mess)
			n.notis.Store(msg.request.id,
				nmember{id: msg.request.id, lastSent: time.Now().Add(-time.Minute)})
		}
	}
}

// notify sends tele message to all chatIDs for a given content.
func (n *notifier) notify(content string) {
	for _, cid := range n.chatIDs {
		message := tele.NewMessage(cid, content)
		n.bot.Send(message)
	}
}

func (n *notifier) newLog(name, message string) string {
	return fmt.Sprintf("[notifier] %s: %s", name, message)
}
