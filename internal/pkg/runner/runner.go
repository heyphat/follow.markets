package runner

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	ta "github.com/itsphat/techan"
	"github.com/sdcoffey/big"

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
	maxSize = 500
)

func ChangeMaxSize(size int) {
	maxSize = size
}

type RunnerConfigs struct {
	Asset    AssetClass
	Market   MarketType
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
		Asset:    Crypto,
		Market:   Cash,
		LFrames:  lineFrames,
		IConfigs: tax.NewDefaultIndicatorConfigs(),
	}
}

type Fundamental struct {
	//cmcRank           int     `json:"cmc_rank"`
	MaxSupply         float64 `json:"max_supply"`
	TotalSupply       float64 `json:"total_supply"`
	CirculatingSupply float64 `json:"circulating_supply"`
}

type Runner struct {
	sync.Mutex

	name        string
	lines       map[time.Duration]*tax.Series
	configs     *RunnerConfigs
	fundamental *Fundamental
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

// SetFundamental set the fundamental values to the runner.
func (r *Runner) SetFundamental(fund *Fundamental) { r.fundamental = fund }

// GetLines returns a line of type tax.Series based on the given time frame.
func (r *Runner) GetLines(d time.Duration) (*tax.Series, bool) { k, v := r.lines[d]; return k, v }

// GetConfigs returns the runner's configurations.
func (r *Runner) GetConfigs() *RunnerConfigs { return r.configs }

// GetName returns the runner's name.
func (r *Runner) GetName() string { return r.name }

// GetUniqueName returns the unique name for the runner.
func (r *Runner) GetUniqueName(prefix ...string) string {
	var out string
	switch r.configs.Market {
	case Cash:
		out = r.name
	case Futures:
		out = r.name + "PERP"
	case Margin:
		out = r.name + "MARG"
	default:
		out = r.name + strconv.Itoa(int(time.Now().Unix()))
	}
	if len(prefix) > 0 {
		out = strings.Join(prefix, "-") + "-" + out
	}
	return out
}

// GetMarketType returns the runner market.
func (r *Runner) GetMarketType() MarketType { return r.configs.Market }

// GetCap return the current marketcap based on the current price of the runner.
func (r *Runner) GetCap() big.Decimal {
	if r == nil || r.fundamental == nil {
		return big.ZERO
	}
	if candle := r.LastCandle(r.SmallestFrame()); candle != nil {
		return candle.ClosePrice.Mul(big.NewDecimal(r.fundamental.TotalSupply))
	}
	return big.ZERO
}

// GetFloat returns the circulating supply of the runner.
func (r *Runner) GetFloat() big.Decimal {
	if r == nil || r.fundamental == nil {
		return big.ZERO
	}
	return big.NewDecimal(r.fundamental.CirculatingSupply)
}

// GetTotalSupply returns the total supply of the runner.
func (r *Runner) GetTotalSupply() big.Decimal {
	if r == nil || r.fundamental == nil {
		return big.ZERO
	}
	return big.NewDecimal(r.fundamental.TotalSupply)
}

// GetMaxSupply returns the max supply of the runner.
func (r *Runner) GetMaxSupply() big.Decimal {
	if r == nil || r.fundamental == nil {
		return big.ZERO
	}
	return big.NewDecimal(r.fundamental.MaxSupply)
}

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
