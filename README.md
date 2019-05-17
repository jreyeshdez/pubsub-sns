
 # Connecting Google Cloud Pub/Sub to Amazon SNS Topic through Cloud Function
 
 Using Google Cloud Functions to integrate AWS SNS and Google Cloud Pub/Sub using Go.
 
 The SNS service makes a POST request to the Cloud Function URL when a new message is published to the
 SNS Topic. This function will validate the sender.
 
 This function then publishes the message to Cloud Pub/Sub.
 
 For this to work is expected that you are already familiar with setting up an AWS SNS topic.
 
 # Creating Cloud Pub/Sub topic and subscription
 
 Creating the topic that will receive SNS messages: 
 - `gcloud pubsub topics create sns-events`
 
 Creating a subscription to test the integration
 - `gcloud pubsub subscriptions create sns-watcher --topic sns-events`
 
 Deploying the function. Replace [YOUR_STAGE_BUCKET] with your Cloud Functions staging bucket as well as SNS_ARN:
 - `gcloud functions deploy PublishSNSMessage --set-env-vars TOPIC_NAME=sns-events,SNS_ARN=arn:aws:sns:AWS_REGION:AWS_ACCOUNT:my-sns-topic --runtime go111 --trigger-http --stage-bucket [YOUR_STAGE_BUCKET]`
 
 Once the function is deployed copy the `httpsTrigger` URL in the output.
 
 Do not forget to add proper permissions to Service account in PubSub Topic.
 
 # Creating the AWS SNS subscription
 
 In the AWS console go to your SNS topic and create a subscription.
 Choose HTTPS as the protocol and enter the Cloud Function URL you copied above and Create the subscription.
 
 The subscription will be created in pending state. SNS then sends a confirmation request to the Cloud Function.
 The function recognises the request as a confirmation request and confirm it by fetching the URL provided by SNS.
 
 # Testing the integration
 
 Using the Publish feature in the SNS section of the AWS console to generate a test message in raw format.
 Wait a few seconds and then run the following command to confirm that Cloud Function relayed the message to Pub/Sub:
 - `gcloud pubsub subscriptions pull sns-watcher --auto-ack`