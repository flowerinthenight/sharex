package kettle

import (
	"os"

	awszconf "github.com/NYTimes/gizmo/config/aws"
	zpubsub "github.com/NYTimes/gizmo/pubsub"
	awszpubsub "github.com/NYTimes/gizmo/pubsub/aws"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
)

type Creds struct {
	Region string
	Key    string
	Secret string
}

// GetTopic returns the ARN of a newly created topic or an existing one. CreateTopic API
// returns the ARN of an existing topic.
func GetTopic(name string, c ...Creds) (*string, error) {
	var sess *session.Session
	var err error
	region := os.Getenv("AWS_REGION")
	if len(c) > 0 {
		if c[0].Region != "" {
			region = c[0].Region
		}

		if c[0].Key != "" && c[0].Secret != "" {
			sess, err = session.NewSession(&aws.Config{
				Region:      aws.String(region),
				Credentials: credentials.NewStaticCredentials(c[0].Key, c[0].Secret, ""),
			})
		}
	}

	if sess == nil {
		sess = session.Must(session.NewSession())
	}

	svc := sns.New(sess)
	res, err := svc.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String(name),
	})

	if err != nil {
		return nil, err
	}

	return res.TopicArn, nil
}

func NewPublisher(name string, c ...Creds) (zpubsub.Publisher, error) {
	topicArn, err := GetTopic(name, c...)
	if err != nil {
		return nil, err
	}

	region := os.Getenv("AWS_REGION")
	key := os.Getenv("AWS_ACCESS_KEY_ID")
	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if len(c) > 0 {
		if c[0].Region != "" {
			region = c[0].Region
		}

		if c[0].Key != "" {
			key = c[0].Key
		}

		if c[0].Secret != "" {
			secret = c[0].Secret
		}
	}

	cnf := awszpubsub.SNSConfig{
		Config: awszconf.Config{
			Region:    region,
			AccessKey: key,
			SecretKey: secret,
		},
		Topic: *topicArn,
	}

	return awszpubsub.NewPublisher(cnf)
}
