package gPubSubToSNS

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/pubsub"
)

type publishRequest struct {
	Type             string `json:"Type,omitempty"`
	MessageID        string `json:"MessageId,omitempty"`
	Token            string `json:"Token,omitempty"`
	TopicArn         string `json:"TopicArn,omitempty"`
	Subject          string `json:"Subject,omitempty"`
	Message          string `json:"Message,omitempty"`
	SubscribeURL     string `json:"SubscribeURL,omitempty"`
	Timestamp        string `json:"Timestamp,omitempty"`
	SignatureVersion string `json:"SignatureVersion,omitempty"`
	Signature        string `json:"Signature,omitempty"`
	SigningCertURL   string `json:"SigningCertURL,omitempty"`
	UnsubscribeURL   string `json:"UnsubscribeURL,omitempty"`
}

// projectID is set from the GCP_PROJECT environment variable, which is
// automatically set by the Cloud Functions runtime.
var projectID = os.Getenv("GCP_PROJECT")

var topicName = os.Getenv("TOPIC_NAME")
var expectedTopicArn = os.Getenv("SNS_ARN")

// client is a global Pub/Sub client, initialized once per instance.
var client *pubsub.Client

func init() {
	// err is pre-declared to avoid shadowing client.
	var err error

	// client is initialized with context.Background() because it should
	// persist between function invocations.
	client, err = pubsub.NewClient(context.Background(), projectID)
	if err != nil {
		log.Fatalf("an error occurred creating pubsun client: %v", err)
	}
}

// PublishSNSMessage publishes a message to Pub/Sub.
// PublishSNSMessage only works with topics that already exist.
func PublishSNSMessage(w http.ResponseWriter, r *http.Request) {

	// Read the request body.
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("error reading request %v", err)
		http.Error(w, "error reading request", http.StatusBadRequest)
		return
	}

	// Parse the request body
	message := publishRequest{}
	if err := json.Unmarshal(data, &message); err != nil {
		log.Printf("invalid SNS message: %v", err)
		http.Error(w, "invalid SNS message", http.StatusForbidden)
		return
	}

	if strings.TrimSpace(message.TopicArn) != strings.TrimSpace(expectedTopicArn) {
		// we got a request from a topic we were not expecting
		// this function is set up to only receive from one specified SNS topic
		// one could adapt this to accept an array, but if you do not check
		// the origin of the message, anyone could end up publishing to your
		// cloud function
		log.Printf("invalid SNS topic: %v", err)
		http.Error(w, "invalid SNS topic", http.StatusForbidden)
		return
	}

	switch t := strings.ToLower(message.Type); t {
	case "subscriptionconfirmation":

		if r.Method != http.MethodPost {
			log.Printf("only POST method accepted, received %s", r.Method)
			http.Error(w, "error reading request", http.StatusMethodNotAllowed)
			return
		}

		// SNS subscriptions are confirmed by requesting the special URL sent
		// by the service as a confirmation

		client := &http.Client{}
		req, err := http.NewRequest("GET", message.SubscribeURL, nil)
		if err != nil {
			log.Printf("an error occurred: %v", err)
			http.Error(w, "error creating request to confirm subscription", http.StatusInternalServerError)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("an error occurred while confirming subscription: %v", err)
			http.Error(w, "an error occurred while confirming subscription", http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("confirming subscription failed: %v", err)
			http.Error(w, "confirming subscription failed", http.StatusInternalServerError)
			return
		}

		log.Println(fmt.Sprintf("subscription confirmed through %s", message.SubscribeURL))
	case "notification":

		if r.Method != http.MethodPost {
			log.Printf("only POST method accepted, received %s", r.Method)
			http.Error(w, "error reading request", http.StatusMethodNotAllowed)
			return
		}

		// this is a regular SNS notice, we relay to Pubsub
		log.Println(fmt.Sprintf("messageID %s : Message %s", message.MessageID, message.Message))

		attributes := make(map[string]string)
		attributes["snsMessageId"] = message.MessageID
		attributes["snsSubject"] = message.Subject

		m := &pubsub.Message{
			Data:       []byte(message.Message),
			Attributes: attributes,
		}

		// Publish and Get use r.Context() because they are only needed for this
		// function invocation. If this were a background function, they would use
		// the ctx passed as an argument.
		id, err := client.Topic(topicName).Publish(r.Context(), m).Get(r.Context())
		if err != nil {
			log.Printf("error publishing message in topic %s: %v", topicName, err)
			http.Error(w, "Error publishing message", http.StatusInternalServerError)
			return
		}

		log.Printf("Published msg: %v", id)
		return

	default:
		log.Printf("it should not get here ..: %v", err)
		http.Error(w, "an error happened", http.StatusBadRequest)
		return
	}
}
