package main

import (
	"log"
	"os"
	"time"
)

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
)

var deferError = false

func checkDeferError(err error) bool {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "Throttling" {
				log.Printf("Warning (%s): %s\n", aerr.Code(), aerr.Message())
				return false
			}

			switch aerr.Code() {
			case "NoSuchEntity":
			case "ResourceNotFoundException":
			case "ValidationError":
			default:
				log.Printf("Error (%s): %s\n", aerr.Code(), aerr.Message())
				deferError = true
				return true
			}
			log.Printf("Warning (%s): %s\n", aerr.Code(), aerr.Message())
		} else {
			log.Printf("Error: %s\n", err.Error())
			deferError = true
		}
		return true
	}
	return false
}

func DeleteEksRole(iamClient *iam.IAM, eksRoleName *string) {
	log.Printf("Deleting EKS service role \"%s\"...\n", *eksRoleName)

	policyArn := "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
	_, err := iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	checkDeferError(err)

	policyArn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
	_, err = iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
		RoleName:  eksRoleName,
		PolicyArn: &policyArn,
	})
	checkDeferError(err)

	_, err = iamClient.DeleteRole(&iam.DeleteRoleInput{
		RoleName: eksRoleName,
	})
	if checkDeferError(err) {
		return
	}

	log.Printf("Role \"%s\" successfully deleted!\n", *eksRoleName)
}

func DeleteEksClusterVpc(cfClient *cloudformation.CloudFormation, clusterStackName *string) {
	log.Printf("Deleting Amazon EKS Cluster VPC \"%s\"...\n", *clusterStackName)

	resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: clusterStackName,
	})
	if checkDeferError(err) {
		return
	}

	stackId := resp.Stacks[0].StackId

	_, err = cfClient.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: clusterStackName,
	})
	if checkDeferError(err) {
		return
	}

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: stackId,
		})
		if checkDeferError(err) {
			if err.(awserr.Error).Code() == "ValidationError" {
				log.Printf("Cluster VPC \"%s\" successfully deleted!\n", *clusterStackName)
			}
			return
		}

		switch *resp.Stacks[0].StackStatus {
		case "DELETE_COMPLETE":
			log.Printf("Cluster VPC \"%s\" successfully deleted!\n", *clusterStackName)
			return
		case "DELETE_IN_PROGRESS":
			time.Sleep(requestInterval)
		default:
			log.Printf("Error: Unexpected stack status: %s\n", *resp.Stacks[0].StackStatus)
			deferError = true
			return
		}
	}
}

func DeleteEksCluster(eksClient *eks.EKS, clusterName *string) {
	log.Printf("Deleting Amazon EKS Cluster \"%s\"...\n", *clusterName)

	_, err := eksClient.DeleteCluster(&eks.DeleteClusterInput{
		Name: clusterName,
	})
	if checkDeferError(err) {
		return
	}

	for {
		resp, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
			Name: clusterName,
		})

		if err != nil && err.(awserr.Error).Code() == "ResourceNotFoundException" {
			log.Printf("EKS Cluster \"%s\" successfully Deleted!\n", *clusterName)
			return
		}

		if checkDeferError(err) {
			return
		}

		switch *resp.Cluster.Status {
		case "DELETING":
			time.Sleep(requestInterval)
		default:
			log.Printf("Error: Unexpected cluster status: %s\n", *resp.Cluster.Status)
			deferError = true
			return
		}
	}
}

func DeleteEksEc2KeyPair(ec2Client *ec2.EC2, keyPairName *string) {
	log.Printf("Deleting Amazon EC2 key pair \"%s\"...\n", *keyPairName)
	_, err := ec2Client.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: keyPairName,
	})
	if checkDeferError(err) {
		return
	}

	log.Printf("Amazon EC2 key pair \"%s\" successfully Deleted!\n", *keyPairName)
}

func DeleteEksWorkerNodes(cfClient *cloudformation.CloudFormation, nodesStackName *string) {
	log.Printf("Deleting Amazon EKS Worker Nodes...\n")

	resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: nodesStackName,
	})
	if checkDeferError(err) {
		return
	}

	stackId := resp.Stacks[0].StackId

	_, err = cfClient.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: nodesStackName,
	})
	if checkDeferError(err) {
		return
	}

	for {
		resp, err := cfClient.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: stackId,
		})
		if checkDeferError(err) {
			return
		}

		switch *resp.Stacks[0].StackStatus {
		case "DELETE_COMPLETE":
			log.Printf("EKS Worker Nodes \"%s\" successfully deleted!\n", *nodesStackName)
			return
		case "DELETE_IN_PROGRESS":
			time.Sleep(requestInterval)
		default:
			log.Printf("Error: Unexpected stack status: %s\n", *resp.Stacks[0].StackStatus)
			deferError = true
			return
		}
	}
}

func deleteAWSKubernetesCluster() {
	sess := session.Must(session.NewSession())
	iamClient := iam.New(sess)
	eksClient := eks.New(sess)
	cfClient := cloudformation.New(sess)
	ec2Client := ec2.New(sess)

	serviceSuffix := os.Getenv("NSM_AWS_SERVICE_SUFFIX")

	// Deleting Amazon EKS Worker Nodes
	nodesStackName := "nsm-nodes" + serviceSuffix
	DeleteEksWorkerNodes(cfClient, &nodesStackName)

	// Deleting Amazon EKS Cluster
	clusterName := "nsm" + serviceSuffix
	DeleteEksCluster(eksClient, &clusterName)

	// Deleting Amazon EKS Cluster VPC
	clusterStackName := "nsm-srv" + serviceSuffix
	DeleteEksClusterVpc(cfClient, &clusterStackName)

	// Deleting Amazon Roles and Keys
	eksRoleName := "nsm-role" + serviceSuffix
	DeleteEksRole(iamClient, &eksRoleName)
	keyPairName := "nsm-key-pair" + serviceSuffix
	DeleteEksEc2KeyPair(ec2Client, &keyPairName)

	if deferError {
		os.Exit(1)
	}
}
