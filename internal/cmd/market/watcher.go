package market

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"follow.markets/internal/pkg/runner"
	"follow.markets/pkg/log"
	ta "github.com/itsphat/techan"

	tax "follow.markets/internal/pkg/techanex"
)

type watcher struct {
	sync.Mutex
	connected bool
	runners   *sync.Map

	// shared properties with other market participants
	logger       *log.Logger
	provider     *provider
	communicator *communicator
}

type wmember struct {
	runner *runner.Runner
	bChann chan *ta.Candle
	tChann chan *tax.Trade
}

func newWatcher(participants *sharedParticipants) (*watcher, error) {
	if participants == nil || participants.communicator == nil || participants.logger == nil {
		return nil, errors.New("missing shared participants")
	}
	return &watcher{
		connected: false,
		runners:   &sync.Map{},

		logger:       participants.logger,
		provider:     participants.provider,
		communicator: participants.communicator,
	}, nil
}

// isConnected returns true when the watcher is connected to other market participants, false otherwise.
func (w *watcher) isConnected() bool { return w.connected }

// Get return a runner which is watching on the watchlist
func (w *watcher) get(name string) *runner.Runner {
	if m, ok := w.runners.Load(name); ok {
		return m.(wmember).runner
	}
	return nil
}

// watchlist returns a watchlist where tickers are being closely monitored and reported.
func (w *watcher) watchlist() []string {
	tickers := []string{}
	w.runners.Range(func(key, value interface{}) bool {
		tickers = append(tickers, key.(string))
		return true
	})
	return tickers
}

// isWatchingOn returns whether the ticker is on the watchlist or not.
func (w *watcher) isWatchingOn(ticker string) bool {
	valid := false
	w.runners.Range(func(key, value interface{}) bool {
		valid = key.(string) == ticker
		return !valid
	})
	return valid
}

// isSynced returns whether the ticker is correctly synced with the market data on time frame.
// It only checks if the last candle held timestamp is the latest one compared to the current time.
func (w *watcher) isSynced(ticker string, duration time.Duration) bool {
	if time.Now().Sub(time.Now().Truncate(duration)) <= time.Minute {
		return true
	}
	last := w.lastCandles(ticker)
	if len(last) == 0 {
		return false
	}
	for _, c := range last {
		if c.Period.End.Sub(c.Period.Start) != duration {
			continue
		}
		if time.Now().Truncate(duration).Unix() == c.Period.Start.Unix() {
			return true
		}
	}
	return false
}

// watch initializes the process to add a ticker to the watchlist. The process keep
// watching the ticker by comsuming the 1-minute candle and trade information boardcasted
// from the streamer.
func (w *watcher) watch(ticker string, rc *runner.RunnerConfigs) error {
	if !w.connected {
		w.connect()
	}
	if w.isWatchingOn(ticker) {
		return nil
	}
	m := wmember{
		runner: runner.NewRunner(ticker, rc),
		bChann: make(chan *ta.Candle, 3),
		tChann: make(chan *tax.Trade, 10),
	}
	for _, f := range m.runner.GetConfigs().LFrames {
		candles, err := w.provider.fetchBinanceKlinesV3(ticker, f, &fetchOptions{limit: 500})
		if err != nil {
			return err
		}
		if len(candles) == 0 {
			return errors.New(fmt.Sprintf("failed to fetch data for frame %v", f))
		}
		if !m.runner.Initialize(&ta.TimeSeries{Candles: candles}, &f) {
			return errors.New(fmt.Sprintf("failed to sync %v candles on initialization", f))
		}
	}
	w.Lock()
	defer w.Unlock()
	w.runners.Store(ticker, m)
	go w.await(m)
	w.logger.Info.Println(w.newLog(ticker, "started to watch"))
	return nil
}

// await will loop forever to receive streaming data from the streamer. This function is meant
// to run in a separate go routine. The watcher can close listening channels to stop watching when
// it receives drop signals from the market.
func (w *watcher) await(mem wmember) {
	for !w.registerStreamingChannel(mem) {
		w.logger.Error.Println(w.newLog(mem.runner.GetName(), "failed to register streaming data"))
	}
	go func() {
		for msg := range mem.bChann {
			if !mem.runner.SyncCandle(msg) {
				w.logger.Error.Println(w.newLog(mem.runner.GetName(), "failed to sync new candle on watching"))
				continue
			}
			w.communicator.watcher2Evaluator <- w.communicator.newMessage(mem, nil)
		}
	}()
	go func() {
		for _ = range mem.tChann {
			//w.logger.Info.Println(msg)
		}
	}()
}

// lastCandles returns all last candles from all time frames of a member in the watchlist
func (w *watcher) lastCandles(ticker string) []*ta.Candle {
	candles := make([]*ta.Candle, 0)
	r := w.get(ticker)
	if r == nil {
		return candles
	}
	for _, d := range r.GetConfigs().LFrames {
		c := r.LastCandle(d)
		if r != nil {
			candles = append(candles, c)
		}
	}
	return candles
}

// lastIndicators return all last indicators from all time frames a member in the watchlist
func (w *watcher) lastIndicators(ticker string) []*tax.Indicator {
	inds := make([]*tax.Indicator, 0)
	r := w.get(ticker)
	if r == nil {
		return inds
	}
	for _, d := range r.GetConfigs().LFrames {
		c := r.LastIndicator(d)
		if r != nil {
			inds = append(inds, c)
		}
	}
	return inds
}

// connect connects the watcher to other market participants py listening to
// decicated channels for the communication.
func (w *watcher) connect() {
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
func (w *watcher) registerStreamingChannel(mem wmember) bool {
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

// This processes the request from the streamer, currently the streamer only requests
// for the `mem` channels in order to reinitialize the streaming data if necessary.
func (w *watcher) processStreamerRequest(msg *message) {
	if mem, ok := w.runners.Load(msg.request.what.(string)); ok && msg.response != nil {
		msg.response <- w.communicator.newPayload(mem)
		close(msg.response)
	}
}

// generates a new log with the format for the watcher
func (w *watcher) newLog(ticker, message string) string {
	return fmt.Sprintf("[watcher] %s: %s", ticker, message)
}
