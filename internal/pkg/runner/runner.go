package runner

import (
	"fmt"
	"sort"
	"sync"
	"time"

	ta "github.com/itsphat/techan"

	tax "follow.markets/internal/pkg/techanex"
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
		2 * time.Hour,
		4 * time.Hour,
		24 * time.Hour,
	}
	maxSize = 1000
)

func ChangeMaxSize(size int) {
	maxSize = size
}

type RunnerConfigs struct {
	LFrames  []time.Duration
	IConfigs tax.IndicatorConfigs
}

func NewRunnerDefaultConfigs() *RunnerConfigs {
	lineFrames := []time.Duration{
		time.Minute,
		//3 * time.Minute,
		5 * time.Minute,
		//10 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		60 * time.Minute,
		//2 * time.Hour,
		4 * time.Hour,
		24 * time.Hour,
	}
	return &RunnerConfigs{
		LFrames:  lineFrames,
		IConfigs: tax.NewDefaultIndicatorConfigs(),
	}
}

type Runner struct {
	sync.Mutex

	name    string
	lines   map[time.Duration]*tax.Series
	configs *RunnerConfigs
}

func NewRunner(name string, configs *RunnerConfigs) *Runner {
	if configs == nil || len(configs.LFrames) == 0 {
		configs = NewRunnerDefaultConfigs()
	}
	lines := make(map[time.Duration]*tax.Series, len(configs.LFrames))
	//lines[time.Minute] = tax.NewSeries(configs.IConfigs)
	for _, frame := range configs.LFrames {
		lines[frame] = tax.NewSeries(configs.IConfigs)
	}
	return &Runner{
		name:    name,
		lines:   lines,
		configs: configs,
	}
}

// validateFrame returns true if the given duration for a line is acceptable.
//func (r *Runner) validateFrame(d time.Duration) bool {
//	for _, duration := range acceptedFrames {
//		if duration == d {
//			return true
//		}
//	}
//	return false
//}

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
		series.Shrink(maxSize)
	}
	return true
}

// SmallestFrame return the smallest time duration of the line that the runner is holding.
func (r *Runner) SmallestFrame() time.Duration {
	frames := r.configs.LFrames
	sort.Slice(frames, func(i, j int) bool {
		return frames[i] < frames[j]
	})
	return frames[0]
}

// LastCandle returns last candle on the given time frame of the runner.
func (r *Runner) LastCandle(d time.Duration) *ta.Candle {
	line, ok := r.GetLines(d)
	if !ok || line == nil {
		return nil
	}
	return line.Candles.LastCandle()
}

func (r *Runner) LastIndicator(d time.Duration) *tax.Indicator {
	line, ok := r.GetLines(d)
	if !ok || line == nil {
		return nil
	}
	return line.Indicators.LastIndicator()
}

// Initialize initializes a time series with the given candle series. It's used for the
// the first time of initializing the series.
func (r *Runner) Initialize(series *ta.TimeSeries, d *time.Duration) bool {
	line, ok := r.GetLines(*d)
	if !ok || line == nil {
		return false
	}
	return line.SyncCandles(series, d)
}

// Validate the given frame
func ValidateFrame(d time.Duration) bool {
	for _, duration := range acceptedFrames {
		if duration == d {
			return true
		}
	}
	return false
}
