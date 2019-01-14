package main

import (
	"net/http"
	"os"
	"time"

	"github.com/leecalcote/pwk-twitter-auth/dao"
	"github.com/leecalcote/pwk-twitter-auth/handlers"
	"github.com/leecalcote/pwk-twitter-auth/queue"

	"github.com/sirupsen/logrus"
)

// main creates and starts a Server listening.
func main() {
	address := os.Getenv("HOST")
	port := os.Getenv("PORT")
	proto := os.Getenv("PROTO")

	dyno, err := dao.NewDynoDao()
	if err != nil {
		logrus.Fatalf("unable to create a DynamoDB dao: %v", err)
	}

	mmq := queue.NewMemQ(dyno)

	// read credentials from environment variables if available
	config := &handlers.Config{
		TwitterConsumerKey:    os.Getenv("TWITTER_CONSUMER_KEY"),
		TwitterConsumerSecret: os.Getenv("TWITTER_CONSUMER_SECRET"),
		CallbackURL:           proto + address + ":" + port + "/twitter/callback",

		Mmq: mmq,
	}
	if config.TwitterConsumerKey == "" {
		logrus.Fatal("Missing Twitter Consumer Key")
	}
	if config.TwitterConsumerSecret == "" {
		logrus.Fatal("Missing Twitter Consumer Secret")
	}

	config.Loc, err = time.LoadLocation(os.Getenv("LOCAL_TZ_NAME"))
	if err != nil {
		logrus.Fatalf("Time zone provided is not valid: %v", err)
	}
	logrus.Infof("Starting Server listening on %s:%s", address, port)
	err = http.ListenAndServe(":"+port, handlers.New(config))
	if err != nil {
		logrus.Fatalf("ListenAndServe error: %v", err)
	}
}
