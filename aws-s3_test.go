package alfredo

import (
	"fmt"
	"os"
	"reflect"
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
	s3c.Load("getobjecttest.json")
	if err := s3c.EstablishSession(); err != nil {
		t.Error("failed to establish S3 session")
		t.Errorf("error=%s\n", err.Error())
		os.Exit(1)
	}
	var wanted []string
	//	wanted = append(wanted, "")

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "base test",
			fields: fields{
				Credentials: s3c.Credentials,
				established: true,
				Client:      s3c.Client,
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
	s3c.Load("getobjecttest.json")
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
			// s3c := &S3ClientSession{
			// 	Credentials: tt.fields.Credentials,
			// 	Bucket:      tt.fields.Bucket,
			// 	ObjectKey:   tt.fields.ObjectKey,
			// 	Endpoint:    tt.fields.Endpoint,
			// 	Region:      tt.fields.Region,
			// 	Client:      tt.fields.Client,
			// 	Versioning:  tt.fields.Versioning,
			// 	established: tt.fields.established,
			// 	keepBucket:  tt.fields.keepBucket,
			// 	PolicyId:    tt.fields.PolicyId,
			// }
			s3c.Load("getobjecttest.json")
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
	s3c.Load("getobjecttest.json")
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
				established: true,
				Client:      s3c.Client,
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

func TestS3ClientSession_RecursiveBucketDeleteAlt(t *testing.T) {
	// type fields struct {
	// 	Credentials S3credStruct
	// 	Bucket      string
	// 	Endpoint    string
	// 	Region      string
	// 	ObjectKey   string
	// 	Client      *s3.S3
	// 	Versioning  bool
	// 	established bool
	// 	keepBucket  bool
	// 	PolicyId    string
	// }
	// tests := []struct {
	// 	name    string
	// 	fields  fields
	// 	wantErr bool
	// }{
	// 	// TODO: Add test cases.
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		s3c := S3ClientSession{
	// 			Credentials: tt.fields.Credentials,
	// 			Bucket:      tt.fields.Bucket,
	// 			Endpoint:    tt.fields.Endpoint,
	// 			Region:      tt.fields.Region,
	// 			ObjectKey:   tt.fields.ObjectKey,
	// 			Client:      tt.fields.Client,
	// 			Versioning:  tt.fields.Versioning,
	// 			established: tt.fields.established,
	// 			keepBucket:  tt.fields.keepBucket,
	// 			PolicyId:    tt.fields.PolicyId,
	// 		}
	// 		if err := s3c.RecursiveBucketDeleteAlt(); (err != nil) != tt.wantErr {
	// 			t.Errorf("S3ClientSession.RecursiveBucketDeleteAlt() error = %v, wantErr %v", err, tt.wantErr)
	// 		}
	// 	})
	// }
}
