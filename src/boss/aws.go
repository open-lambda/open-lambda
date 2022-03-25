package main

import (
	"os"

	// "github.com/aws/aws-sdk-go-v2/aws"
	// "github.com/aws/aws-sdk-go-v2/config"
	// "github.com/aws/aws-sdk-go-v2/service/s3"

	"fmt"
)

func CreateKeyPair(client *ec2.EC2, keyName string) (*ec2.CreateKeyPairOutput, error) {
	result, err := client.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: aws.String(keyName),
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func WriteKey(fileName string, fileData *string) error {
	err := os.WriteFile(fileName, []byte(*fileData), 0400)
	return err
}


func main() {
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile: "default",
		Config: aws.Config{
			Region: aws.String("us-west-2"),
		},
	})

	if err != nil {
		fmt.Printf("Failed to initialize new session: %v", err)
		return
	}

	ec2Client := ec2.New(sess)

	keyName := "ec2-go-tutorial-key-name"
	createRes, err := CreateKeyPair(ec2Client, keyName)
	if err != nil {
		fmt.Printf("Couldn't create key pair: %v", err)
		return
	}

	err = WriteKey("/tmp/aws_ec2_key.pem", createRes.KeyMaterial)
	if err != nil {
		fmt.Printf("Couldn't write key pair to file: %v", err)
		return
	}
	fmt.Println("Created key pair: ", *createRes.KeyName)
}