package alfredo

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
)

func TestS3ClientSession_ListObjectsWithPrefix(t *testing.T) {
	SetVerbose(true)
	type fields struct {
		Credentials S3credStruct
		Bucket      string
		Endpoint    string
		Region      string
		Client      *s3.S3
		Versioning  bool
		established bool
		keepBucket  bool
		PolicyId    string
	}
	type args struct {
		prefix string
	}
	var s3c S3ClientSession
	s3c.Credentials.AccessKey = "00cf16ab0700cea0f1cb"
	s3c.Credentials.SecretKey = "9Z93vqGuVwtDGFKNa440K4Ptb1VDFRNHe2bFnMi+"
	s3c.Endpoint = "https://s3-support.cloudian.com"
	s3c.Region = "region"
	if err := s3c.EstablishSession(); err != nil {
		t.Error("failed to establish S3 session")
		t.Errorf("error=%s\n", err.Error())
		os.Exit(1)
	}
	var wanted []string
	wanted = append(wanted, "10May2024-tgt/nasunifilere18e74f0-5c04-498a-8f73-90ef5d659d7e-2-0.CLOUDIAN_METADATA.1.log")
	//endpoint = s3-support.cloudian.com

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{

		// 		aws_access_key_id = 00cf16ab0700cea0f1cb
		// aws_secret_access_key = 9Z93vqGuVwtDGFKNa440K4Ptb1VDFRNHe2bFnMi+
		// endpoint = s3-support.cloudian.com

		{
			name: "base test",
			fields: fields{
				Credentials: s3c.Credentials,
				Bucket:      "purity-20161205",
				Region:      "region",
				established: true,
				Client:      s3c.Client,
				Endpoint:    "s3-support.cloudian.com",
			},
			args: args{
				prefix: "10May2024-tgt/",
			},
			want: wanted,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3c := S3ClientSession{
				Credentials: tt.fields.Credentials,
				Bucket:      tt.fields.Bucket,
				Endpoint:    tt.fields.Endpoint,
				Region:      tt.fields.Region,
				Client:      tt.fields.Client,
				Versioning:  tt.fields.Versioning,
				established: tt.fields.established,
				keepBucket:  tt.fields.keepBucket,
				PolicyId:    tt.fields.PolicyId,
			}
			got, err := s3c.ListObjectsWithPrefix(tt.args.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("S3ClientSession.ListObjectsWithPrefix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for g := 0; g < len(got); g++ {
				VerbosePrintln(got[g])
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("S3ClientSession.ListObjectsWithPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestS3ClientSession_LoadUserCredentialsForProfile(t *testing.T) {
	type fields struct {
		Credentials S3credStruct
		Bucket      string
		Endpoint    string
		Region      string
		Client      *s3.S3
		Versioning  bool
		established bool
		keepBucket  bool
		PolicyId    string
		Profile     string
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "base test",
			fields: fields{
				Profile: "cit",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3c := S3ClientSession{
				Credentials: tt.fields.Credentials,
				Bucket:      tt.fields.Bucket,
				Endpoint:    tt.fields.Endpoint,
				Region:      tt.fields.Region,
				Client:      tt.fields.Client,
				Versioning:  tt.fields.Versioning,
				established: tt.fields.established,
				keepBucket:  tt.fields.keepBucket,
				PolicyId:    tt.fields.PolicyId,
			}
			s3c.Credentials.Profile = tt.fields.Profile
			if err := s3c.LoadUserCredentialsForProfile(); (err != nil) != tt.wantErr {
				t.Errorf("S3ClientSession.LoadUserCredentialsForProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !strings.EqualFold(s3c.Credentials.AccessKey, "7ded71cfd9a190f8b365") ||
				!strings.EqualFold(s3c.Credentials.SecretKey, "UB6qSYPFYMf3EY3tXqh7BAhD1K4i0TKcAu5mSzea") {
				t.Errorf("incorrect credentials loaded")
				t.Errorf("ak=%s", s3c.Credentials.AccessKey)
				t.Errorf("sk=%s", s3c.Credentials.SecretKey)
			}
		})
	}
}

func TestS3ClientSession_PresignedURL(t *testing.T) {
	type fields struct {
		Credentials S3credStruct
		Bucket      string
		Endpoint    string
		Region      string
		Client      *s3.S3
		Versioning  bool
		established bool
		keepBucket  bool
		PolicyId    string
		ObjectKey   string
	}
	type args struct {
		objectKey    string
		expiredHours int
	}
	var s3c S3ClientSession
	s3c.Credentials.AccessKey = "00cf16ab0700cea0f1cb"
	s3c.Credentials.SecretKey = "9Z93vqGuVwtDGFKNa440K4Ptb1VDFRNHe2bFnMi+"
	s3c.Endpoint = "https://s3-support.cloudian.com"
	s3c.Region = "region"
	if err := s3c.EstablishSession(); err != nil {
		t.Error("failed to establish S3 session")
		t.Errorf("error=%s\n", err.Error())
		os.Exit(1)
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "base test",
			fields: fields{
				Credentials: s3c.Credentials,
				Bucket:      "purity-20161205",
				Region:      "region",
				established: true,
				Client:      s3c.Client,
				Endpoint:    "s3-support.cloudian.com",
				ObjectKey:   "10May2024-tgt/nasunifilere18e74f0-5c04-498a-8f73-90ef5d659d7e-2-0.CLOUDIAN_METADATA.1.log",
			},

			args: args{
				objectKey:    "10May2024-tgt/10May2024-tgt/nasunifilere18e74f0-5c04-498a-8f73-90ef5d659d7e-2-0.CLOUDIAN_METADATA.1.log",
				expiredHours: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3c := &S3ClientSession{
				Credentials: tt.fields.Credentials,
				Bucket:      tt.fields.Bucket,
				ObjectKey:   tt.fields.ObjectKey,
				Endpoint:    tt.fields.Endpoint,
				Region:      tt.fields.Region,
				Client:      tt.fields.Client,
				Versioning:  tt.fields.Versioning,
				established: tt.fields.established,
				keepBucket:  tt.fields.keepBucket,
				PolicyId:    tt.fields.PolicyId,
			}
			got, err := s3c.PresignedURL(tt.args.expiredHours)
			if (err != nil) != tt.wantErr {
				t.Errorf("S3ClientSession.PresignedURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) == 0 {
				t.Errorf("S3ClientSession.PresignedURL() = %v, want %v", got, tt.want)
			} else {
				fmt.Printf("url=%s\n", got)
			}
		})
	}
}

func TestS3ClientSession_GetObjectHash(t *testing.T) {
	type fields struct {
		Credentials S3credStruct
		Bucket      string
		Endpoint    string
		Region      string
		Client      *s3.S3
		Versioning  bool
		established bool
		keepBucket  bool
		PolicyId    string
		ObjectKey   string
	}
	var s3c S3ClientSession
	s3c.Credentials.AccessKey = "00cf16ab0700cea0f1cb"
	s3c.Credentials.SecretKey = "9Z93vqGuVwtDGFKNa440K4Ptb1VDFRNHe2bFnMi+"
	s3c.Endpoint = "https://s3-support.cloudian.com"
	s3c.Region = "region"
	if err := s3c.EstablishSession(); err != nil {
		t.Error("failed to establish S3 session")
		t.Errorf("error=%s\n", err.Error())
		os.Exit(1)
	}

	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "base test",
			fields: fields{
				Credentials: s3c.Credentials,
				Bucket:      "purity-20161205",
				Region:      "region",
				established: true,
				Client:      s3c.Client,
				Endpoint:    "s3-support.cloudian.com",
				ObjectKey:   "10May2024-tgt/nasunifilere18e74f0-5c04-498a-8f73-90ef5d659d7e-2-0.CLOUDIAN_METADATA.1.log",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3c := &S3ClientSession{
				Credentials: tt.fields.Credentials,
				Bucket:      tt.fields.Bucket,
				Endpoint:    tt.fields.Endpoint,
				Region:      tt.fields.Region,
				Client:      tt.fields.Client,
				Versioning:  tt.fields.Versioning,
				established: tt.fields.established,
				keepBucket:  tt.fields.keepBucket,
				PolicyId:    tt.fields.PolicyId,
				ObjectKey:   tt.fields.ObjectKey,
			}
			fmt.Printf("key=%s\n", tt.fields.ObjectKey)
			fmt.Printf("s3=%s\n", s3c.GetURL())
			got, err := s3c.GetObjectHash()
			if (err != nil) != tt.wantErr {
				t.Errorf("S3ClientSession.GetObjectHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("got=%s\n", got)
			// if got != tt.want {
			// 	t.Errorf("S3ClientSession.GetObjectHash() = %v, want %v", got, tt.want)
			// }
		})
	}
}

func TestS3ClientSession_ParseFromURL(t *testing.T) {
	type fields struct {
		Credentials S3credStruct
		Bucket      string
		Endpoint    string
		Region      string
		ObjectKey   string
		Client      *s3.S3
		Versioning  bool
		established bool
		keepBucket  bool
		PolicyId    string
	}
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "base test",
			fields: fields{},

			args: args{
				url: "s3://bucket/long/object/key",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3c := &S3ClientSession{
				Credentials: tt.fields.Credentials,
				Bucket:      tt.fields.Bucket,
				Endpoint:    tt.fields.Endpoint,
				Region:      tt.fields.Region,
				ObjectKey:   tt.fields.ObjectKey,
				Client:      tt.fields.Client,
				Versioning:  tt.fields.Versioning,
				established: tt.fields.established,
				keepBucket:  tt.fields.keepBucket,
				PolicyId:    tt.fields.PolicyId,
			}
			if err := s3c.ParseFromURL(tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("S3ClientSession.ParseFromURL() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				fmt.Printf("bucket=%q key=%q\n", s3c.Bucket, s3c.ObjectKey)
			}

		})
	}
}
