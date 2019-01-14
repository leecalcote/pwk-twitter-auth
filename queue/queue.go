package queue

import (
	"time"

	"github.com/leecalcote/pwk-twitter-auth/dao"
	"github.com/leecalcote/pwk-twitter-auth/tweeter"

	"github.com/go-msgqueue/msgqueue"
	"github.com/go-msgqueue/msgqueue/memqueue"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type MemQ struct {
	queue *memqueue.Queue
	dao   *dao.DynoDao
}

func NewMemQ(dao *dao.DynoDao) *MemQ {
	mq := &MemQ{dao: dao}
	mq.queue = memqueue.NewQueue(&msgqueue.Options{
		Handler:   mq.QueueHandler,
		RateLimit: rate.Every(time.Second),
	})
	return mq
}

func (m *MemQ) DepoInQueue(user *dao.TwitterUser, tweeter *tweeter.Tweeter, msg string) error {
	return m.queue.Call(user, tweeter, msg)
}

func (m *MemQ) QueueHandler(user *dao.TwitterUser, tweeter *tweeter.Tweeter, msg string) error {
	check, err := m.dao.CheckIfUserEventExists(user.ID, user.Event)
	if err != nil {
		return err
	}
	if !check {
		// first tweet, on tweet success save to DB
		err = m.dao.AddUser(user)
		if err != nil {
			return err
		}
		logrus.Infof("In the queue handler, preparing to post tweet. . .")
		err = tweeter.PostTweet(msg)
		if err != nil {
			logrus.Errorf("unable to post a tweet: %v", err)
		}
	} else {
		logrus.Infof("User - Event already exists for user: %s and event: %s. Skip tweeting.", user.ID, user.Event)
	}
	return nil
}
