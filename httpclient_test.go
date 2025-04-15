package alfredo

import "testing"

func TestHttpApiStruct_getBaseURL(t *testing.T) {
	type fields struct {
		UserName       string
		Password       string
		Fqdn           string
		responseBody   []byte
		requestPayload []byte
		statusCode     int
		ssh            SSHStruct
		Timeout        int
		QueryParams    map[string]string
		Headers        map[string]string
		forceLocal     bool
		forceRemote    bool
		Secure         bool
		Token          string
		Port           int
		IgnoreConflict bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "HTTP with default port",
			fields: fields{
				Fqdn:   "example.com",
				Secure: false,
				Port:   80,
			},
			want: "http://example.com",
		},
		{
			name: "HTTPS with default port",
			fields: fields{
				Fqdn:   "example.com",
				Secure: true,
				Port:   443,
			},
			want: "https://example.com",
		},
		{
			name: "HTTP with custom port",
			fields: fields{
				Fqdn:   "example.com",
				Secure: false,
				Port:   8080,
			},
			want: "http://example.com:8080",
		},
		{
			name: "HTTPS with custom port",
			fields: fields{
				Fqdn:   "example.com",
				Secure: true,
				Port:   8443,
			},
			want: "https://example.com:8443",
		},
		{
			name: "HTTP with missing port",
			fields: fields{
				Fqdn:   "example.com",
				Secure: false,
				Port:   0,
			},
			want: "panic",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has := HttpApiStruct{
				UserName:       tt.fields.UserName,
				Password:       tt.fields.Password,
				Fqdn:           tt.fields.Fqdn,
				responseBody:   tt.fields.responseBody,
				requestPayload: tt.fields.requestPayload,
				statusCode:     tt.fields.statusCode,
				ssh:            tt.fields.ssh,
				Timeout:        tt.fields.Timeout,
				QueryParams:    tt.fields.QueryParams,
				Headers:        tt.fields.Headers,
				forceLocal:     tt.fields.forceLocal,
				forceRemote:    tt.fields.forceRemote,
				Secure:         tt.fields.Secure,
				token:          tt.fields.Token,
				Port:           tt.fields.Port,
				IgnoreConflict: tt.fields.IgnoreConflict,
			}
			defer func() {
				if r := recover(); r != nil {
					if tt.want != "panic" {
						t.Errorf("HttpApiStruct.getBaseURL() = panic, want %v", tt.want)
					}
				}
			}()
			if got := has.getBaseURL(); got != tt.want {
				t.Errorf("HttpApiStruct.getBaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestHttpApiStruct_BuildCurlCLI(t *testing.T) {
	type fields struct {
		UserName       string
		Password       string
		Fqdn           string
		responseBody   []byte
		requestPayload []byte
		statusCode     int
		ssh            SSHStruct
		Timeout        int
		QueryParams    map[string]string
		Headers        map[string]string
		forceLocal     bool
		forceRemote    bool
		Secure         bool
		Token          string
		Port           int
		IgnoreConflict bool
	}
	type args struct {
		method string
		uri    string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "GET request with default headers",
			fields: fields{
				Fqdn:   "example.com",
				Secure: false,
				Port:   80,
			},
			args: args{
				method: "GET",
				uri:    "/api/v1/resource",
			},
			want: "curl -m 30 -ss -k -X GET -H \"Content-Type: application/json\" http://example.com/api/v1/resource",
		},
		{
			name: "POST request with payload",
			fields: fields{
				Fqdn:           "example.com",
				Secure:         true,
				Port:           443,
				requestPayload: []byte(`{"key":"value"}`),
			},
			args: args{
				method: "POST",
				uri:    "/api/v1/resource",
			},
			want: "curl -m 30 -ss -k -X POST -d @- -H \"Content-Type: application/json\" https://example.com/api/v1/resource",
		},
		{
			name: "GET request with query params",
			fields: fields{
				Fqdn:        "example.com",
				Secure:      false,
				Port:        8080,
				QueryParams: map[string]string{"param1": "value1", "param2": "value2"},
			},
			args: args{
				method: "GET",
				uri:    "/api/v1/resource",
			},
			want: "curl -m 30 -ss -k -X GET -H \"Content-Type: application/json\" -G --data-urlencode param1=value1 --data-urlencode param2=value2 http://example.com:8080/api/v1/resource",
		},
		{
			name: "GET request with custom headers",
			fields: fields{
				Fqdn:    "example.com",
				Secure:  true,
				Port:    8443,
				Headers: map[string]string{"Custom-Header": "HeaderValue"},
			},
			args: args{
				method: "GET",
				uri:    "/api/v1/resource",
			},
			want: "curl -m 30 -ss -k -X GET -H \"Custom-Header: HeaderValue\" -H \"Content-Type: application/json\" https://example.com:8443/api/v1/resource",
		},
		{
			name: "GET request with authorization header",
			fields: fields{
				Fqdn:   "example.com",
				Secure: true,
				Port:   443,
				Token:  "mytoken",
			},
			args: args{
				method: "GET",
				uri:    "/api/v1/resource",
			},
			want: "curl -m 30 -ss -k -X GET -H \"Authorization: Bearer mytoken\" -H \"Content-Type: application/json\" https://example.com/api/v1/resource",
		},
		{
			name: "GET request with authorization header using username and password",
			fields: fields{
				Fqdn:     "example.com",
				Secure:   true,
				Port:     8443,
				UserName: "myuser",
				Password: "myp&ssw!!rd",
			},
			args: args{
				method: "GET",
				uri:    "/api/v1/resource",
			},
			want: "curl -m 30 -ss -k -X GET -u myuser:\"myp&ssw!!rd\" -H \"Content-Type: application/json\" https://example.com:8443/api/v1/resource",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has := HttpApiStruct{
				UserName:       tt.fields.UserName,
				Password:       tt.fields.Password,
				Fqdn:           tt.fields.Fqdn,
				responseBody:   tt.fields.responseBody,
				requestPayload: tt.fields.requestPayload,
				statusCode:     tt.fields.statusCode,
				ssh:            tt.fields.ssh,
				Timeout:        tt.fields.Timeout,
				QueryParams:    tt.fields.QueryParams,
				Headers:        tt.fields.Headers,
				forceLocal:     tt.fields.forceLocal,
				forceRemote:    tt.fields.forceRemote,
				Secure:         tt.fields.Secure,
				token:          tt.fields.Token,
				Port:           tt.fields.Port,
				IgnoreConflict: tt.fields.IgnoreConflict,
			}
			if got := has.BuildCurlCLI(tt.args.method, tt.args.uri); got != tt.want {
				t.Errorf("HttpApiStruct.BuildCurlCLI() = \ngot =\"%v\"\nwant=\"%v\"", got, tt.want)
			}
		})
	}
}
func TestHttpApiStruct_ParseFromURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantFqdn   string
		wantPort   int
		wantSecure bool
		wantURI    string
		wantErr    bool
	}{
		{
			name:       "HTTP URL with default port",
			url:        "http://example.com/path/to/resource",
			wantFqdn:   "example.com",
			wantPort:   80,
			wantSecure: false,
			wantURI:    "/path/to/resource",
			wantErr:    false,
		},
		{
			name:       "HTTPS URL with default port",
			url:        "https://example.com/path/to/resource",
			wantFqdn:   "example.com",
			wantPort:   443,
			wantSecure: true,
			wantURI:    "/path/to/resource",
			wantErr:    false,
		},
		{
			name:       "HTTP URL with custom port",
			url:        "http://example.com:8080/path/to/resource",
			wantFqdn:   "example.com",
			wantPort:   8080,
			wantSecure: false,
			wantURI:    "/path/to/resource",
			wantErr:    false,
		},
		{
			name:       "HTTPS URL with custom port",
			url:        "https://example.com:8443/path/to/resource",
			wantFqdn:   "example.com",
			wantPort:   8443,
			wantSecure: true,
			wantURI:    "/path/to/resource",
			wantErr:    false,
		},
		{
			name:       "Invalid URL",
			url:        "://example.com/path/to/resource",
			wantFqdn:   "",
			wantPort:   0,
			wantSecure: false,
			wantURI:    "",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has := &HttpApiStruct{}
			gotURI, err := has.ParseFromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("HttpApiStruct.ParseFromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotURI != tt.wantURI {
				t.Errorf("HttpApiStruct.ParseFromURL() = %v, want %v", gotURI, tt.wantURI)
			}
			if has.Fqdn != tt.wantFqdn {
				t.Errorf("HttpApiStruct.Fqdn = %v, want %v", has.Fqdn, tt.wantFqdn)
			}
			if has.Port != tt.wantPort {
				t.Errorf("HttpApiStruct.Port = %v, want %v", has.Port, tt.wantPort)
			}
			if has.Secure != tt.wantSecure {
				t.Errorf("HttpApiStruct.Secure = %v, want %v", has.Secure, tt.wantSecure)
			}
		})
	}
}
