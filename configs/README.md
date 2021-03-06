# Application configurations. See the `configs/configs.json` file.
1. `env`: the targeted environment when deploying the application, `development` or `production`. If it is `development`, some parts of the application won't be fully initialized to save API call credit. For example, `coinmarketcap` API calls.

3. `server`: this is the basic configuration of the application. The variables are self-explanatory. If you want to change the port where the application will be listening to, you can do it on `server.port`. 

4. `datadog`: if you want to monitor the application performance on datadog, you can set up a datadog agent and update the corresponding variables in this session. Otherwise, leaving it unchanged will be fine.

5. `market`: this is the main configuration you will need to understand and update for your system. The market consists of a few components which have their own configurations. 
    1. `base.crypto.quote_currency`: the basic information of the market, all the other agents can refer to. Usually `USDT`.
    2. `provider`: this agent manages all communications with market data providers. 
        1. `binance`: currently the application tracks Binance spot and futures markets,  you need to have a Binance account and obtain the keys, `api_key` and `secret_key`.
        2. `coinmarketcap`: coinmarketcap APIs provide the fundamental data for coins and tokens, for example: supply and circulation. If the coinmarketcap `api_key` is missing, you won't be able to configure signals based on fundamentals.
    3. `notifier`: this agent is responsible for communicating with users (sending and accepting messages/requests) on trading events via telegram bot. 
        1. `bot_token`: a telegram bot token. Ask the [BotFather](https://core.telegram.org/bots) for a telegram `bot_token` if you don't know how to get it yet. Then start a conversation with your bot after deploying the system. The bot'll notify you messages when your signals are successfully evaluated.
        2. `bot_password`: this password is to prevent others to access your bot. You can set it to anything, the bot will ask you for authorization when you start talking to it.
        3. `chat_ids`: a list of chatIDs. If you know your tele account chatID, you can configure it here. Otherwise, you can obtain it from the `bot`. If you set your chatID here, you will receive signals but it won't allow you to access the trader and interfere trades, you will need to authorize.
    4. `watcher`: this agent is responsible for watching the markets. It holds the market data of all runners on the watchlist.
        1. `watchlist`: a list of regex patterns that are matched against all tradable tickers on Binance spot and futures markets. The watchlist sample in `configs/configs.json` matches all `USDT`-based pairs. If you want to watch only 1 ticker, `BTCUSDT` for example, you can just add `BTCUSDT` to the watchlist.
        2. `runner`: when a ticker matches the `watchlist` patterns, the watcher initializes it as a `runner` and starts to watch on it. A runner consists of multiple timeseries of candles and indicators. One timeseries is associated with one timeframe. 
            1. `frames`: a list of timeframes you want the watcher to watch on. The supported frames are: `1m`, `3m`, `5m`, `10m`, `15m`, `30m`, `1h`, `2h`, `4h`, `1d`. The values must be in second. 
            2. `indicators`: a list of indicators. Refer to the `indicator` docs for a list of supported indicators. An indicator comes with a list of parameters, often be a list of window frames.
    5. `evaluator`: this agent is responsible for evaluating your signals. Refer to the `evaluator` [docs]() for more information about how to build a signal. A signal is a set of rules that are avaluated against the candles and indicators on a runner. Make sure that you set the right params for the `runner` before refering it to configure signals.
        1. `source_path`: the place to store all of your signals. All the signals in this directory will be evaluated every minutes (when a new candle formed) to all runners on the watchlist. 
    6. `tester`: this agent is responsible for testing your signals/strategies. Refer to the `tester` [docs](https://paxon.notion.site/Backtests-ac8e074b161e4994a3b5cea593130a3f) for more information about how to execute a backtest request. The process on the tester often happens independent of the other agents, it's a good idea to deploy tester separately. 
        1. `save_path`: the place to save all the test results.
        2. `init_balance`: the initialized balance before testing. You can also configure this in the NotionDB before executing a backtest request.
        2. `profit_margin`: the profit margin. You can also configure this in the NotionDB before executing a backtest request.
        4. `loss_tolerance`: the loss tolerance. You can also configure this in the NotionDB before executing a backtest request.
    7. `trader`: when evaluator completes its jobs with a valid signal, it will send the signal to trader to place trades. Trades will be carried out from placing a limit order (always limit order set by the signal) to converting base currency back to quote currency by the trader.
        1. `allowed`: is global variable to disable trader. Set it to `false`, trader won't be able to trade. You can also switch this on the tele bot via the notifier after authorizing identity or update the trader's configuration via trader APIs.
        2. `allowed_markets`: `CASH` or/and `FUTURES`. 
        3. `allowed_patterns`: if you want to trade only 1 ticker, say `BTCUSDT`, you can set it here. If you don't set anything, it won't trade. If you use the same regex as the watcher, It will trade the entire markets.
        4. `min_balance_to_trade`: the amount in dollar to place an order. With the `FUTURES` market, this amount will be multiplied by `max_leverage`. If you don't have enough this minimum balance on the quote currency, it won't trade.
        5. `max_leverage`: this is used for `FUTURES` market, it will leverage your money to place trades. `CASH` market will always have leverage of 1.
        6. `max_wait_to_fill` (in seconds): the trader will cancel an order after this amount of seconds if it hasn't been matched yet.
        7. `loss_tolerance`: the loss tolerance per trade based on the current best price and average filled price. Example, 0.01, 1% loss.
        8. `profit_margin`: the profit margin per trade based on the current best price and average filled price. Example, 0.02, 2% gain.
        9. `max_loss_per_trade`: this is used for `FUTURES` markets, since I want to set the absolute loss instead of ratio like `CASH` market.
        10. `min_profit_per_trade`: this is used for `FUTURES` markets, since I want to set absolute profit instead of ratio like `CASH` market.
5. `database`: this is optional on the system. I didn't want to use any database, but since the project grows bigger, some form of persistent datasource is required. It supports `mongodb` and `notion` at the moment. You can remove this session if you don't want to use db, and just want to track market via tele bot.
    1. `use`: scpecifies which type of db you want to initialize.
    2. `mongodb`: the configuration for mongodb.
    3. `notion`: the configuration for notiondb.
