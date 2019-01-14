# pwk-twitter-auth
This is a twitter authenticator app which has been solely used in our [Istio workshops](https://github.com/leecalcote/istio-service-mesh-workshop) for logging into the customized BookInfo app. It is written in Go. The app authenticates users with twitter, posts a tweet on their behalf and also persists minimal info in AWS DynamoDB.

__Please note:__ This app is __NOT__ meant for use in production.

## Requirements to run this app:
1. Create a twitter app using the [developer console](https://developer.twitter.com/apps) and register a callback url like `proto://host:port/twitter/callback` with your values for proto (http or https), host and port
1. Get the consumer key and secret, and store them in environment variables: TWITTER_CONSUMER_KEY & TWITTER_CONSUMER_SECRET.
1. Create a AWS DynamoDB table with the following fields:
    1. twitter_id - type String, Partition Key
    1. event - type String, Sort Key
1. Get the aws access and secret key for a user with read and write privileges to the DynamoDB table.
1. Store the aws region, DynamoDB table name, access and secret key in environment variables: AWS_REGION, DYNAMO_USER_TABLE, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY

## Setup and run
The project uses [dep](https://github.com/golang/dep) for dependency management. Please setup `dep` if you have not already done so. Then run `dep ensure` to download the dependencies.

A Makefile is created for ease of use.
1. To run the app locally: `make run`
1. To create a docker image run: `make docker`
1. To run the app locally in a docker container: `make docker-run`
1. `make docker-push` to push the built image
