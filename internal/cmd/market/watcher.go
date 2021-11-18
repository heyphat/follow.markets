package market

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"follow.market/internal/pkg/runner"
	"follow.market/pkg/log"
	ta "github.com/itsphat/techan"

	tax "follow.market/internal/pkg/techanex"
)

//type WatcherConfigs struct {
//	Communicator *Communicator
//}

type Watcher struct {
	sync.Mutex
	connected bool
	runners   *sync.Map

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

type member struct {
	runner *runner.Runner
	bChann chan *ta.Candle
	tChann chan *tax.Trade
}

func newWatcher(participants *sharedParticipants) (*Watcher, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	return &Watcher{
		connected: false,
		runners:   &sync.Map{},

		logger:       participants.logger,
		provider:     participants.provider,
		communicator: participants.communicator,
	}, nil
}

// IsConnected return whether the watcher is connected to other market participants.
func (w *Watcher) IsConnected() bool { return w.connected }

// Get return a runner which is watching on the watchlist
func (w *Watcher) Get(name string) *runner.Runner {
	if m, ok := w.runners.Load(name); ok {
		return m.(member).runner
	}
	return nil
}

// Watchlist returns a watchlist where tickers are being closely monitored and reported.
func (w *Watcher) Watchlist() []string {
	tickers := []string{}
	w.runners.Range(func(key, value interface{}) bool {
		tickers = append(tickers, key.(string))
		return true
	})
	return tickers
}

// IsWatchingOn returns whether the ticker is on the watchlist or not.
func (w *Watcher) IsWatchingOn(ticker string) bool {
	valid := false
	w.runners.Range(func(key, value interface{}) bool {
		valid = key.(string) == ticker
		return !valid
	})
	return valid
}

// Watch initializes the process to add a ticker to the watchlist. The process keep
// watching the ticker by comsuming the 1-minute candle and trade information boardcasted
// from the streamer.
func (w *Watcher) Watch(ticker string) error {
	if !w.connected {
		w.connect()
	}
	if w.IsWatchingOn(ticker) {
		return nil
	}
	m := member{
		runner: runner.NewRunner(ticker, nil),
		bChann: make(chan *ta.Candle, 10),
		tChann: make(chan *tax.Trade, 10),
	}
	candles, err := w.provider.fetchBinanceKlines(ticker, time.Minute)
	if err != nil {
		return err
	}
	if !m.runner.Initialize(&ta.TimeSeries{Candles: candles}) {
		return errors.New("failed to sync candles on initialization")
	}
	w.runners.Store(ticker, m)
	go w.watch(m)
	return nil
}

func (w *Watcher) watch(mem member) {
	for !w.registerStreamingChannel(mem) {
		w.logger.Error.Println(w.newLog(mem.runner.GetName(), "failed to register streaming data"))
	}
	go func() {
		for msg := range mem.bChann {
			if ok := mem.runner.SyncCandle(msg); ok {
				w.logger.Error.Println(w.newLog(mem.runner.GetName(), "failed to sync new candle on watching"))
				continue
			}
		}
	}()
	go func() {
		for msg := range mem.tChann {
			w.logger.Info.Println(msg)
		}
	}()
}

// Connect connects the watcher to other market participants py listening to
// a decicated channels for the communication.
func (w *Watcher) connect() {
	w.Lock()
	defer w.Unlock()
	if w.connected {
		return
	}
	go func() {
		for msg := range w.communicator.streamer2Watcher {
			go w.processStreamerRequest(msg)
		}
	}()
	w.connected = true
}

// registerStreamingChannel registers the runners with the streamer in order to
// recevie and consume candles broadcasted by data providor. Every time the Watch
// method is called and the ticker is vallid, it will invoke this method.
func (w *Watcher) registerStreamingChannel(mem member) bool {
	doneStreamingRegister := false
	var maxTries int
	for !doneStreamingRegister && maxTries <= 3 {
		resC := make(chan *payload)
		w.communicator.watcher2Streamer <- w.communicator.newMessage(mem, resC)
		doneStreamingRegister = (<-resC).what.(bool)
		maxTries++
	}
	return doneStreamingRegister
}

func (w *Watcher) processStreamerRequest(msg *message) {
	if mem, ok := w.runners.Load(msg.request.what.(string)); ok && msg.response != nil {
		msg.response <- w.communicator.newPayload(mem)
		close(msg.response)
	}
}

func (w *Watcher) newLog(ticker, message string) string {
	return fmt.Sprintf("[watcher] %s: %s", ticker, message)
}
