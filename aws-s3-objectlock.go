package alfredo

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (s3c *S3ClientSession) IsObjectLockEnabled() error {
	if len(s3c.Bucket) == 0 {
		panic("aws-s3.go::IsObjectLockEnabled():: bucket is not set")
	}
	if len(s3c.Endpoint) == 0 {
		panic("aws-s3.go::IsObjectLockEnabled():: endpoint is not set")
	}
	if len(os.Getenv("SIMULATE_S3_ERROR")) > 0 {
		return fmt.Errorf("simulated S3 error")
	}

	if !s3c.Versioning {
		VerbosePrintln("can't have object lock without versioning enabled")
		s3c.EnableObjectLock = false
		return nil
	}

	if err := s3c.EstablishSession(); err != nil {
		panic(err.Error())
	}

	ct := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !s3c.enforceCertificates},
	}

	awsConfig := aws.NewConfig().
		WithEndpoint(s3c.Endpoint).
		WithCredentials(credentials.NewStaticCredentials(
			s3c.Credentials.AccessKey,
			s3c.Credentials.SecretKey,
			"",
		)).
		WithS3ForcePathStyle(true).
		WithRegion(s3c.Region).
		WithHTTPClient(&http.Client{Transport: ct})

	sess := session.Must(session.NewSession(awsConfig))
	svc := s3.New(sess)

	input := &s3.GetObjectLockConfigurationInput{
		Bucket: aws.String(s3c.Bucket),
	}

	result, err := svc.GetObjectLockConfiguration(input)
	if err != nil {
		// Object Lock not enabled
		if strings.Contains(err.Error(), "NoSuchObjectLockConfiguration") || strings.Contains(err.Error(), "ObjectLockConfigurationNotFound") {
			VerbosePrintln("no such object lock configuration")
			s3c.EnableObjectLock = false
			return nil
		}

		// Access denied → treat as disabled (mirrors your versioning logic)
		if strings.Contains(err.Error(), "Access Denied") {
			VerbosePrintln("access denied")
			s3c.EnableObjectLock = false
			return nil
		}

		panic(err.Error())
	}

	if result.ObjectLockConfiguration == nil ||
		result.ObjectLockConfiguration.ObjectLockEnabled == nil {
		VerbosePrintln("ObjectLockConfiguration structure was nil")
		s3c.EnableObjectLock = false
		return nil
	}
	VerbosePrintln("last chance to determine if object lock is enabled...")
	s3c.EnableObjectLock = strings.EqualFold(
		*result.ObjectLockConfiguration.ObjectLockEnabled,
		s3.ObjectLockEnabledEnabled,
	)
	VerbosePrintf("result=%s\n", *result.ObjectLockConfiguration.ObjectLockEnabled)
	VerbosePrintf("RHS=%s\n", s3.ObjectLockEnabledEnabled)

	return nil
}

func (s3c *S3ClientSession) putObjectLockConfiguration(mode string, retentionDays int64) error {
	if mode != s3.ObjectLockRetentionModeGovernance &&
		mode != s3.ObjectLockRetentionModeCompliance {
		return fmt.Errorf("invalid retention mode: %s", mode)
	}

	_, err := s3c.Client.PutObjectLockConfiguration(&s3.PutObjectLockConfigurationInput{
		Bucket: aws.String(s3c.Bucket),
		ObjectLockConfiguration: &s3.ObjectLockConfiguration{
			ObjectLockEnabled: aws.String(s3.ObjectLockEnabledEnabled),
			Rule: &s3.ObjectLockRule{
				DefaultRetention: &s3.DefaultRetention{
					Mode: aws.String(mode),
					Days: aws.Int64(retentionDays),
				},
			},
		},
	})
	return err
}

func (s3c *S3ClientSession) PutObjectLockConfigurationCompliance(retentionDays int64) error {
	return s3c.putObjectLockConfiguration("COMPLIANCE", retentionDays)
}

func (s3c *S3ClientSession) PutObjectLockConfigurationGovernance(retentionDays int64) error {
	return s3c.putObjectLockConfiguration("GOVERNANCE", retentionDays)
}

func (s3c *S3ClientSession) putObjectRetentionDays(
	key, versionID string,
	mode string, // "GOVERNANCE" or "COMPLIANCE"
	days int,
) error {

	if mode != s3.ObjectLockRetentionModeGovernance &&
		mode != s3.ObjectLockRetentionModeCompliance {
		return fmt.Errorf("invalid retention mode: %s", mode)
	}

	retainUntil := time.Now().UTC().AddDate(0, 0, days)

	input := &s3.PutObjectRetentionInput{
		Bucket:    aws.String(s3c.Bucket),
		Key:       aws.String(key),
		VersionId: aws.String(versionID),
		Retention: &s3.ObjectLockRetention{
			Mode:            aws.String(mode),
			RetainUntilDate: aws.Time(retainUntil),
		},
	}

	_, err := s3c.Client.PutObjectRetention(input)
	return err
}

func (s3c *S3ClientSession) PutObjectRetentionGovernanceDays(key, versionID string, days int) error {
	return s3c.putObjectRetentionDays(key, versionID, s3.ObjectLockRetentionModeGovernance, days)
}

func (s3c *S3ClientSession) PutObjectRetentionComplianceDays(key, versionID string, days int) error {
	return s3c.putObjectRetentionDays(key, versionID, s3.ObjectLockRetentionModeCompliance, days)
}

func (s3c *S3ClientSession) GetBucketDefaultRetention() (*s3.DefaultRetention, error) {
	out, err := s3c.Client.GetObjectLockConfiguration(&s3.GetObjectLockConfigurationInput{
		Bucket: aws.String(s3c.Bucket),
	})
	if err != nil {
		return nil, err
	}

	// Object Lock not enabled or no default rule
	if out.ObjectLockConfiguration == nil ||
		out.ObjectLockConfiguration.Rule == nil ||
		out.ObjectLockConfiguration.Rule.DefaultRetention == nil {
		return nil, nil
	}

	return out.ObjectLockConfiguration.Rule.DefaultRetention, nil
}

func (s3c *S3ClientSession) GetObjectRetention(key, versionID string,
) (*s3.GetObjectRetentionOutput, error) {

	return s3c.Client.GetObjectRetention(&s3.GetObjectRetentionInput{
		Bucket:    aws.String(s3c.Bucket),
		Key:       aws.String(key),
		VersionId: aws.String(versionID),
	})
}
