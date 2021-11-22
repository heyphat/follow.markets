package market

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	tele "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"follow.market/internal/pkg/strategy"
	"follow.market/pkg/config"
	"follow.market/pkg/log"
	"follow.market/pkg/util"
)

type notifier struct {
	sync.Mutex
	connected bool
	bot       *tele.BotAPI
	chatIDs   []int64

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

func newNotifier(participants *sharedParticipants, configs *config.Configs) (*notifier, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	var chatIDs []int64
	for _, id := range configs.Telegram.ChatIDs {
		if iid, err := strconv.Atoi(id); err != nil {
			return nil, err
		} else {
			chatIDs = append(chatIDs, int64(iid))
		}
	}
	bot, err := tele.NewBotAPI(configs.Telegram.BotToken)
	if err != nil {
		return nil, err
	}
	return &notifier{
		connected: false,
		bot:       bot,
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

// add adds new chat ids to the system if not initialized
func (n *notifier) add(cids []int64) {
	n.Lock()
	defer n.Unlock()
	for _, cid := range cids {
		if !util.Int64SliceContains(n.chatIDs, cid) {
			n.chatIDs = append(n.chatIDs, cid)
		}
	}
}

func (n *notifier) processEvaluatorRequest(msg *message) {
	s := msg.request.what.(*strategy.Strategy)
	n.notify(s.Description())
}

func (n *notifier) notify(content string) {
	for _, cid := range n.chatIDs {
		message := tele.NewMessage(cid, content)
		n.bot.Send(message)
	}
}

func (n *notifier) newLog(name, message string) string {
	return fmt.Sprintf("[notifier] %s: %s", name, message)
}
