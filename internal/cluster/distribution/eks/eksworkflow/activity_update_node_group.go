// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eksworkflow

import (
	"context"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"

	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

const awsNoUpdatesError = "No updates are to be performed."

const UpdateNodeGroupActivityName = "eks-update-node-group"

// TODO: move these to a better place
const nodePoolNameLabelKey = "nodepool.banzaicloud.io/name"
const nodePoolVersionLabelKey = "nodepool.banzaicloud.io/version"

// UpdateNodeGroupActivity updates an existing node group.
type UpdateNodeGroupActivity struct {
	sessionFactory AWSSessionFactory

	// body of the cloud formation template
	cloudFormationTemplate string
}

// UpdateNodeGroupActivityInput holds the parameters for the node group update.
type UpdateNodeGroupActivityInput struct {
	SecretID string
	Region   string

	ClusterName string

	StackName string

	NodePoolName    string
	NodePoolVersion string

	NodeImage string
}

type UpdateNodeGroupActivityOutput struct {
	NodePoolChanged bool
}

// NewUpdateNodeGroupActivity creates a new UpdateNodeGroupActivity instance.
func NewUpdateNodeGroupActivity(sessionFactory AWSSessionFactory, cloudFormationTemplate string) UpdateNodeGroupActivity {
	return UpdateNodeGroupActivity{
		sessionFactory:         sessionFactory,
		cloudFormationTemplate: cloudFormationTemplate,
	}
}

// Register registers the activity in the worker.
func (a UpdateNodeGroupActivity) Register() {
	activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: UpdateNodeGroupActivityName})
}

// Execute is the main body of the activity, returns true if there was any update and that was successful.
func (a UpdateNodeGroupActivity) Execute(ctx context.Context, input UpdateNodeGroupActivityInput) (UpdateNodeGroupActivityOutput, error) {
	sess, err := a.sessionFactory.NewSession(input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil { // internal error?
		return UpdateNodeGroupActivityOutput{}, err
	}

	cloudformationClient := cloudformation.New(sess)

	nodeLabels := []string{
		fmt.Sprintf("%v=%v", nodePoolNameLabelKey, input.NodePoolName),
	}

	if input.NodePoolVersion != "" {
		nodeLabels = append(nodeLabels, fmt.Sprintf("%v=%v", nodePoolVersionLabelKey, input.NodePoolVersion))
	}

	stackParams := []*cloudformation.Parameter{
		{
			ParameterKey:     aws.String("KeyName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeInstanceType"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeSpotPrice"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeAutoScalingGroupMinSize"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeAutoScalingGroupMaxSize"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeAutoScalingInitSize"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("ClusterName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeGroupName"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("ClusterControlPlaneSecurityGroup"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeSecurityGroup"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("VpcId"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("Subnets"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("NodeInstanceRoleId"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("ClusterAutoscalerEnabled"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:     aws.String("TerminationDetachEnabled"),
			UsePreviousValue: aws.Bool(true),
		},
		{
			ParameterKey:   aws.String("BootstrapArguments"),
			ParameterValue: aws.String(fmt.Sprintf("--kubelet-extra-args '--node-labels %v'", strings.Join(nodeLabels, ","))),
		},
	}

	{
		param := &cloudformation.Parameter{
			ParameterKey: aws.String("NodeImageId"),
		}

		if input.NodeImage != "" {
			param.ParameterValue = aws.String(input.NodeImage)
		} else {
			param.UsePreviousValue = aws.Bool(true)
		}

		stackParams = append(stackParams, param)
	}

	// we don't reuse the creation time template, since it may have changed
	updateStackInput := &cloudformation.UpdateStackInput{
		ClientRequestToken: aws.String(activity.GetInfo(ctx).WorkflowExecution.ID),
		StackName:          aws.String(input.StackName),
		Capabilities:       []*string{aws.String(cloudformation.CapabilityCapabilityIam)},
		Parameters:         stackParams,
		Tags:               getNodePoolStackTags(input.ClusterName),
		TemplateBody:       aws.String(a.cloudFormationTemplate),
	}

	_, err = cloudformationClient.UpdateStack(updateStackInput)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ValidationError" && strings.HasPrefix(awsErr.Message(), awsNoUpdatesError) {
			return UpdateNodeGroupActivityOutput{}, nil
		}

		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if awsErr.Code() == request.WaiterResourceNotReadyErrorCode {
				err = pkgCloudformation.NewAwsStackFailure(err, input.StackName, activity.GetInfo(ctx).WorkflowExecution.ID, cloudformationClient)
				err = errors.WrapIff(err, "waiting for %q CF stack create operation to complete failed", input.StackName)
				if pkgCloudformation.IsErrorFinal(err) {
					return UpdateNodeGroupActivityOutput{}, cadence.NewCustomError(ErrReasonStackFailed, err.Error())
				}
				return UpdateNodeGroupActivityOutput{}, errors.WrapIff(err, "waiting for %q CF stack create operation to complete failed", input.StackName)
			}
		}
	}

	return UpdateNodeGroupActivityOutput{NodePoolChanged: true}, nil
}
