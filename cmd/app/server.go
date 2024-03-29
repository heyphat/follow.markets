package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/handlers"

	"gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

func main() {
	tracer.Start(
		tracer.WithEnv(configs.Datadog.Env),
		tracer.WithService(configs.Datadog.Service),
		tracer.WithServiceVersion(configs.Datadog.Version),
	)
	defer tracer.Stop()
	err := profiler.Start(
		profiler.WithEnv(configs.Datadog.Env),
		profiler.WithService(configs.Datadog.Service),
		profiler.WithVersion(configs.Datadog.Version))
	if err != nil {
		logger.Error.Printf("failed to init to a datadog profiler with err: %s", err.Error())
	}
	defer profiler.Stop()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	errChan := make(chan struct{}, 1)

	limitClient := 5
	if configs.Server.Limit > 0 {
		limitClient = configs.Server.Limit
	}
	mw := Compose(LimitClients(limitClient), Logging())
	serverHTTP := &http.Server{
		Handler:      Mux(mw),
		Addr:         fmt.Sprintf(":%d", configs.Server.Port),
		WriteTimeout: time.Duration(configs.Server.Timeout.Write) * time.Second,
		ReadTimeout:  time.Duration(configs.Server.Timeout.Read) * time.Second,
		IdleTimeout:  time.Duration(configs.Server.Timeout.Idle) * time.Second,
	}
	go func() {
		if err := serverHTTP.ListenAndServe(); err != nil {
			logger.Error.Fatalln(err)
			errChan <- struct{}{}
		}
		logger.Info.Printf("Server started... listening :%d\n", configs.Server.Port)
	}()

	select {
	case <-stop:
		break
	case <-errChan:
		break
	}

	logger.Info.Println("Server stopped...")
	serverHTTP.Shutdown(context.Background())
}

// Func is like a middleware
type Func func(http.Handler) http.Handler

// Compose will apply g, then f.
func Compose(f, g Func) Func {
	return func(next http.Handler) http.Handler {
		return f(g(next))
	}
}

// LimitClients Only n simultaneous requests.
func LimitClients(n int) Func {
	sema := make(chan struct{}, n)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sema <- struct{}{}
			defer func() {
				<-sema
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func Logging() Func {
	return func(next http.Handler) http.Handler {
		return handlers.LoggingHandler(os.Stdout, next)
	}
}

func Mux(middleware Func) *mux.Router {
	router := mux.NewRouter()

	// check if server is alive
	router.Handle("/ping",
		middleware(http.HandlerFunc(pong))).Methods("GET")

	// watcher endpoints
	router.Handle("/watcher/watchlist",
		middleware(http.HandlerFunc(watchlist))).Methods("GET")
	router.Handle("/watcher/last/{ticker}",
		middleware(http.HandlerFunc(last))).Methods("GET")
	router.Handle("/watcher/is_synced/{ticker}/{frame}",
		middleware(http.HandlerFunc(synced))).Methods("GET")
	router.Handle("/watcher/watch/{ticker}",
		middleware(http.HandlerFunc(watch))).Methods("POST")
	router.Handle("/watcher/drop/{ticker}",
		middleware(http.HandlerFunc(dropRunner))).Methods("POST")

	// evaluator endpoints
	router.Handle("/evaluator/list",
		middleware(http.HandlerFunc(listSignals))).Methods("GET")
	router.Handle("/evaluator/add/{patterns}",
		middleware(http.HandlerFunc(addSignal))).Methods("POST")
	router.Handle("/evaluator/drop/{names}",
		middleware(http.HandlerFunc(dropSignals))).Methods("POST")

	// notifier enpoints
	router.Handle("/notifier/add_chat_ids/{chat_ids}",
		middleware(http.HandlerFunc(addChatIDs))).Methods("POST")
	router.Handle("/notifier/get_notifications",
		middleware(http.HandlerFunc(getNotifications))).Methods("GET")

	// tester endpoints
	router.Handle("/tester/test/{id}",
		middleware(http.HandlerFunc(test))).Methods("GET")

	// trader endpoints
	router.Handle("/trader/balances",
		middleware(http.HandlerFunc(balances))).Methods("GET")
	router.Handle("/trader/get_configs",
		middleware(http.HandlerFunc(getConfigs))).Methods("GET")
	router.Handle("/trader/update_configs",
		middleware(http.HandlerFunc(updateConfigs))).Methods("POST")

	return router
}

func pong(w http.ResponseWriter, req *http.Request) {
	bts := []byte("pong")
	header := w.Header()
	header.Set("Content-Length", strconv.Itoa(len(bts)))
	w.WriteHeader(http.StatusOK)
	w.Write(bts)
}
