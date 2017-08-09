package svc

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type SsmClient struct {
	*ssm.SSM
}

func (sc *SsmClient) RunShellScript(commands []string, workDir string, instanceIds []string) (*ssm.SendCommandOutput, error) {
	param := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"),
		InstanceIds:  PStrings(instanceIds),
		Parameters: map[string][]*string{
			"commands":         PStrings(commands),
			"workingDirectory": []*string{aws.String(workDir)},
		},
	}
	return sc.SendCommand(param)
}

func (sc *SsmClient) ListAvailableInstanceIds() ([]string, error) {
	aiids := []string{}
	param := &ssm.DescribeInstanceInformationInput{}
	result, err := sc.DescribeInstanceInformation(param)
	if err != nil {
		return aiids, err
	}
	for _, v := range result.InstanceInformationList {
		aiids = append(aiids, *v.InstanceId)
	}
	return aiids, nil
}

func (sc *SsmClient) GetCommandStatus(commandID, instanceID string) (string, error) {
	param := &ssm.GetCommandInvocationInput{
		CommandId:  aws.String(commandID),
		InstanceId: aws.String(instanceID),
	}
	result, err := sc.GetCommandInvocation(param)
	if err != nil {
		return "", err
	}
	return *result.StatusDetails, nil
}

func PStrings(ss []string) []*string {
	pss := []*string{}
	for _, v := range ss {
		pss = append(pss, aws.String(v))
	}
	return pss
}
