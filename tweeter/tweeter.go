package tweeter

import (
	"errors"
	"strings"

	gotwitter "github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/sirupsen/logrus"
)

// One Tweeter per user, not to be shared
type Tweeter struct {
	ConsumerKey, ConsumerSecret string

	AccessToken, AccessSecret string

	TweetSucceeded bool
}

func (tw *Tweeter) PostTweet(msg string) error {
	if strings.TrimSpace(msg) == "" {
		logrus.Error("Error: message is empty")
		return errors.New("message is empty")
	}
	if !tw.TweetSucceeded { // to prevent retweeting
		conf := oauth1.NewConfig(tw.ConsumerKey, tw.ConsumerSecret)
		token := oauth1.NewToken(tw.AccessToken, tw.AccessSecret)
		httpClient := conf.Client(oauth1.NoContext, token)

		client := gotwitter.NewClient(httpClient)

		logrus.Infof("Msg to be tweeted: %s", msg)
		tweet, resp, err := client.Statuses.Update(msg, nil)
		if err != nil {
			logrus.Errorf("Error: posting a tweet: %v", err)
			return errors.New("unable to tweet")
		}
		logrus.Infof("Tweet resp status code: %d", resp.StatusCode)
		logrus.Infof("Tweet: %+#v", tweet)
		tw.TweetSucceeded = true
	}
	return nil
}
