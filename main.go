package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"os"
)

func contains(s []*string, e string) bool {
	for _, a := range s {
		if *a == e {
			return true
		}
	}
	return false
}

func getClusterContainerInstanceArn(clusterName string) ([]*string, error) {
	svc, _ := getSession()
	input := &ecs.ListContainerInstancesInput{
		Cluster: aws.String(clusterName),
	}

	result, err := svc.ListContainerInstances(input)
	if err != nil {
		return nil, err
	}
	return result.ContainerInstanceArns, err
}

func getServiceContainerInstancesArn(clusterName string, familyName string) ([]*string, error) {
	svc, _ := getSession()
	listTaskInput := &ecs.ListTasksInput{
		Cluster: aws.String(clusterName),
		Family:  aws.String(familyName),
	}

	listTasksOutput, err := svc.ListTasks(listTaskInput)

	if err != nil {
		return nil, err
	}

	if len(listTasksOutput.TaskArns) == 0 {
		return nil, nil
	}
	describeTaskInput := &ecs.DescribeTasksInput{
		Cluster: aws.String(clusterName),
		Tasks:   listTasksOutput.TaskArns,
	}

	describeTasks, err := svc.DescribeTasks(describeTaskInput)

	if err != nil {
		return nil, err
	}

	var containerInstances []*string
	for _, instance := range describeTasks.Tasks {
		containerInstances = append(containerInstances, instance.ContainerInstanceArn)
	}
	return containerInstances, nil
}

func getSession() (*ecs.ECS, error) {
	awsSession, err := session.NewSession()

	if err != nil {
		return nil, err
	}
	svc := ecs.New(awsSession)
	return svc, nil
}

func main() {
	var clusterName string
	flag.StringVar(&clusterName, "cluster", "default", "Cluster name")

	var familyName string
	flag.StringVar(&familyName, "family", "", "Cluster name")

	flag.Parse()

	if familyName == "" {
		panic(errors.New("family name required"))
	}
	clusterContainerInstances, err := getClusterContainerInstanceArn(clusterName)

	if err != nil {
		panic(err)
	}

	serviceContainerInstances, err := getServiceContainerInstancesArn(clusterName, familyName)

	if err != nil {
		panic(err)
	}

	var instancesNotInstalled []*string

	if len(serviceContainerInstances) != len(clusterContainerInstances) {
		for _, instance := range clusterContainerInstances {
			if !contains(serviceContainerInstances, *instance) {
				instancesNotInstalled = append(instancesNotInstalled, instance)
			}
		}

		describeContainerInstancesInput := &ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(clusterName),
			ContainerInstances: instancesNotInstalled,
		}

		svc, _ := getSession()
		result, err := svc.DescribeContainerInstances(describeContainerInstancesInput)
		if err != nil {
			panic(err)
		}
		fmt.Println("Some task have not been installed in this instances")
		for _, instance := range result.ContainerInstances {
			fmt.Printf("id:%s  ContainerInstanceArn: %s, runningTaskCount: %d, remaining%s:%d",
				*instance.Ec2InstanceId, *instance.ContainerInstanceArn, *instance.RunningTasksCount,
				*instance.RemainingResources[1].Name, *instance.RemainingResources[1].IntegerValue)
			fmt.Println()
			os.Exit(1)
		}
	} else {
		fmt.Println("All good")
	}
}
