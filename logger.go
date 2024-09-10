package main

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

func NewLogger(config *Config) *log.Logger {
	// Create local log file
	logFile, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	logger := log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Send logs to CloudWatch if configured
	if config.CloudWatchLogGroup != "" && config.CloudWatchLogStream != "" && config.AWSRegion != "" {
		go sendToCloudWatch(config, logger)
	}

	return logger
}

func sendToCloudWatch(config *Config, logger *log.Logger) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(config.AWSRegion),
	}))

	svc := cloudwatchlogs.New(sess)
	input := &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(config.CloudWatchLogGroup),
		LogStreamName: aws.String(config.CloudWatchLogStream),
	}

	_, err := svc.CreateLogStream(input)
	if err != nil {
		logger.Printf("Failed to create CloudWatch log stream: %v", err)
		return
	}

	// CloudWatch log event logic here (for brevity, not fully implemented)
	// You would batch log events and send them to CloudWatch.
}
