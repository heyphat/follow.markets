package runner

import (
	"fmt"
	"sync"
	"time"

	ta "github.com/itsphat/techan"

	tax "follow.market/internal/pkg/techanex"
)

var (
	acceptedFrames = [...]time.Duration{
		time.Minute,
		3 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		60 * time.Minute,
		4 * time.Hour,
		24 * time.Hour,
	}
)

type RunnerConfigs struct {
	LFrames  []time.Duration
	IConfigs *tax.IndicatorConfigs
}

func NewRunnerDefaultConfigs() *RunnerConfigs {
	lineFrames := []time.Duration{
		time.Minute,
		5 * time.Minute,
		15 * time.Minute,
	}
	return &RunnerConfigs{
		LFrames:  lineFrames,
		IConfigs: tax.NewIndicatorDefaultConfigs(),
	}
}

type Runner struct {
	sync.Mutex

	name    string
	lines   map[time.Duration]*tax.Series
	configs *RunnerConfigs
}

func NewRunner(name string, configs *RunnerConfigs) *Runner {
	if configs == nil {
		configs = NewRunnerDefaultConfigs()
	}
	lines := make(map[time.Duration]*tax.Series, len(configs.LFrames))
	lines[time.Minute] = tax.NewSeries(configs.IConfigs)
	for _, frame := range configs.LFrames {
		lines[frame] = tax.NewSeries(configs.IConfigs)
	}
	return &Runner{
		name:    name,
		lines:   lines,
		configs: configs,
	}
}

func (r *Runner) validateFrame(d time.Duration) bool {
	for _, duration := range acceptedFrames {
		if duration == d {
			return true
		}
	}
	return false
}

// GetLines returns a line of type tax.Series based on the given time frame.
func (r *Runner) GetLines(d time.Duration) (*tax.Series, bool) { k, v := r.lines[d]; return k, v }

// GetConfigs returns the runner's configurations
func (r *Runner) GetConfigs() *RunnerConfigs { return r.configs }

// GetName return the runner's name
func (r *Runner) GetName() string { return r.name }

// SyncCandle aggregate the lines with the values of the given candle.
// The syncing process will be different for different lines based on its frame.
// The given candle's period should always be the latest candle broadcasted by
// a market data providor. For example: Binance, AlpacaMarkets, FTX.
func (r *Runner) SyncCandle(c *ta.Candle) bool {
	if c == nil {
		panic(fmt.Errorf("error syncing candle: cannle cannot be nil"))
	}
	for frame, series := range r.lines {
		if !series.SyncCandle(c, &frame) {
			return false
		}
	}
	return true
}

// AddNewLine adds a new price timeseries and indicator timeseries to the runner.
// The newly created timeseries will be aggregated from the 1-minute time series
// if the candles is not given (nil). Otherwise, it uses the given candles.
func (r *Runner) AddNewLine(d *time.Duration, candles *ta.TimeSeries) bool {
	r.Lock()
	defer r.Unlock()

	for _, duration := range r.configs.LFrames {
		if *d == duration {
			return true
		}
	}
	if !r.validateFrame(*d) {
		return false
	}
	line, ok := r.GetLines(time.Minute)
	if !ok {
		return ok
	}
	newCandles := line.Candles
	if candles != nil {
		newCandles = candles
	}
	series := tax.NewSeries(r.configs.IConfigs)
	if !series.SyncCandles(newCandles, d) {
		return false
	}
	r.lines[*d] = series
	r.configs.LFrames = append(r.configs.LFrames, *d)
	return true
}

// Initialize initializes a time series of the 1-minute bar candles. It's used for the
// performance purpose on syncing runner with market data.
func (r *Runner) Initialize(series *ta.TimeSeries) bool {
	for d, line := range r.lines {
		if !line.SyncCandles(series, &d) {
			return false
		}
	}
	return true
}
