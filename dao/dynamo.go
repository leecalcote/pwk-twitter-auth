package dao

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/sirupsen/logrus"
)

type TwitterUser struct {
	ID        string    `json:"twitter_id,omitempty"`
	Name      string    `json:"name,omitempty"`
	Email     string    `json:"email,omitempty"`
	Event     string    `json:"event,omitempty"`
	LoginTime time.Time `json:"log_in_at,omitempty"`
}

type DynoDao struct {
	dbSess *dynamodb.DynamoDB
}

func NewDynoDao() (*DynoDao, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION"))},
	)
	if err != nil {
		logrus.Errorf("error creating a new session: %v", err)
		return nil, err
	}
	return &DynoDao{
		dbSess: dynamodb.New(sess),
	}, nil
}

func (d *DynoDao) AddUser(user *TwitterUser) error {
	av, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		logrus.Errorf("error marshalling user: %#v - %v", user, err)
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(os.Getenv("DYNAMO_USER_TABLE")),
	}

	_, err = d.dbSess.PutItem(input)

	if err != nil {
		logrus.Errorf("error adding user: %#v - %v", user, err)
		return err
	}

	logrus.Infof("Successfully added user: %#v", user)
	return nil
}

func (d *DynoDao) CheckIfUserEventExists(twitterID, event string) (bool, error) {
	consistentRead := true
	gi := &dynamodb.GetItemInput{
		ConsistentRead: &consistentRead,
		TableName:      aws.String(os.Getenv("DYNAMO_USER_TABLE")),
		Key: map[string]*dynamodb.AttributeValue{
			"twitter_id": &dynamodb.AttributeValue{
				S: &twitterID,
			},
			"event": &dynamodb.AttributeValue{
				S: &event,
			},
		},
	}
	gio, err := d.dbSess.GetItem(gi)
	if err != nil {
		logrus.Errorf("error getting twitter id: %s, event: %s - %v", twitterID, event, err)
		return false, err
	}
	logrus.Infof("retrieved user: %+#v", gio)
	return len(gio.Item) > 0, nil
}
