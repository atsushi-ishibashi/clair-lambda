package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/optiopay/klar/clair"
	"github.com/optiopay/klar/docker"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, e events.CloudWatchEvent) error {
	type EventDetail struct {
		AWSRegion         string `json:"awsRegion"`
		RequestParameters struct {
			RepositoryName string `json:"repositoryName"`
			ImageTag       string `json:"imageTag"`
			RegistryID     string `json:"registryId"`
		} `json:"requestParameters"`
	}

	ed := EventDetail{}
	if err := json.Unmarshal(e.Detail, &ed); err != nil {
		return err
	}
	svc := ecr.New(session.New(), aws.NewConfig().WithRegion(ed.AWSRegion))
	resp, err := svc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return err
	}

	authData := resp.AuthorizationData[0]
	dockerConfig := &docker.Config{
		ImageName: strings.Replace(*authData.ProxyEndpoint, "https://", "", 1) + "/" + ed.RequestParameters.RepositoryName + ":" + ed.RequestParameters.ImageTag,
		Token:     *authData.AuthorizationToken,
	}
	image, err := docker.NewImage(dockerConfig)
	if err != nil {
		return err
	}

	err = image.Pull()
	if err != nil {
		return err
	}

	fmt.Printf("Image: %s, Layer: %d\n", ed.RequestParameters.RepositoryName, len(image.FsLayers))

	ver := 1
	c := clair.NewClair(os.Getenv("CLAIR_ADDR"), ver, 1*time.Minute)
	vs, err := c.Analyse(image)
	if err != nil {
		return fmt.Errorf("Failed to analyze using API v%d: %s", ver, err)
	}

	if len(vs) > 0 {
		slackText := fmt.Sprintf("Image: %s\nLayer: %d\n", ed.RequestParameters.RepositoryName, len(image.FsLayers))
		slackText += fmt.Sprintf("%#v", vs)
		if err := postSlack(slackText); err != nil {
			return err
		}
	}

	return nil
}

func postSlack(text string) error {
	type Slack struct {
		Text string `json:"text"`
	}
	slackBody := &Slack{
		Text: text,
	}
	body, _ := json.Marshal(slackBody)
	req, _ := http.NewRequest("POST", os.Getenv("SLACK_URL"), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	_, err := client.Do(req)
	return err
}
