package targetsync

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

// NewAWSTargetGroup returns a new AWS target group destination
func NewAWSTargetGroup(cfg *AWSConfig) (*AWSTargetGroup, error) {
	// TODO: verify that this client is good at creation time (ping or something)
	return &AWSTargetGroup{
		svc: elbv2.New(session.New()),
		cfg: cfg,
	}, nil
}

// AWSTargetGroup is a TargetDestination implementation for AWS target groups
type AWSTargetGroup struct {
	svc *elbv2.ELBV2
	cfg *AWSConfig
}

// GetTargets returns the current set of targets at the destination
func (tg *AWSTargetGroup) GetTargets(ctx context.Context) ([]*Target, error) {
	input := &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(tg.cfg.TargetGroupARN),
	}

	result, err := tg.svc.DescribeTargetHealthWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeInvalidTargetException:
				fmt.Println(elbv2.ErrCodeInvalidTargetException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeHealthUnavailableException:
				fmt.Println(elbv2.ErrCodeHealthUnavailableException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil, err
	}

	targets := make([]*Target, 0)
	for _, targetHealthDecription := range result.TargetHealthDescriptions {
		if tg.cfg.AvailabilityZone == "" ||
			*targetHealthDecription.Target.AvailabilityZone == tg.cfg.AvailabilityZone {
			targets = append(targets, &Target{
				IP:   *targetHealthDecription.Target.Id,
				Port: int(*targetHealthDecription.Target.Port),
			})
		}
	}

	return targets, nil
}

// AddTargets simply adds the targets described
func (tg *AWSTargetGroup) AddTargets(ctx context.Context, targets []*Target) error {

	input := &elbv2.RegisterTargetsInput{
		TargetGroupArn: aws.String(tg.cfg.TargetGroupARN),
		Targets:        tg.TargetToTargetDescription(targets),
	}

	// TODO: check output
	_, err := tg.svc.RegisterTargetsWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeTargetGroupNotFoundException:
				fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeTooManyTargetsException:
				fmt.Println(elbv2.ErrCodeTooManyTargetsException, aerr.Error())
			case elbv2.ErrCodeInvalidTargetException:
				fmt.Println(elbv2.ErrCodeInvalidTargetException, aerr.Error())
			case elbv2.ErrCodeTooManyRegistrationsForTargetIdException:
				fmt.Println(elbv2.ErrCodeTooManyRegistrationsForTargetIdException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}
	return nil
}

// RemoveTargets simply removes the targets described
func (tg *AWSTargetGroup) RemoveTargets(ctx context.Context, targets []*Target) error {
	input := &elbv2.DeregisterTargetsInput{
		TargetGroupArn: aws.String(tg.cfg.TargetGroupARN),
		Targets:        tg.TargetToTargetDescription(targets),
	}

	// TODO: check output
	_, err := tg.svc.DeregisterTargetsWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeTargetGroupNotFoundException:
				fmt.Println(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeInvalidTargetException:
				fmt.Println(elbv2.ErrCodeInvalidTargetException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	return nil
}

// TargetToTargetDescription translates the `Target` struct into an ec2 `TargetDescription`
func (tg *AWSTargetGroup) TargetToTargetDescription(targets []*Target) []*elbv2.TargetDescription {
	descs := make([]*elbv2.TargetDescription, len(targets))
	for i, target := range targets {
		descs[i] = &elbv2.TargetDescription{
			Id:   aws.String(target.IP),
			Port: aws.Int64(int64(target.Port)),
		}
		if tg.cfg.AvailabilityZone != "" {
			descs[i].AvailabilityZone = aws.String(tg.cfg.AvailabilityZone)
		}
	}
	return descs
}
