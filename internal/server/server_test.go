package server

import (
	"testing"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/stretchr/testify/assert"
)

func (suite *Suite) TestDecodePageToken() {
	t := suite.T()
	type args struct {
		s      string
		filter string
	}
	tests := []struct {
		name    string
		args    args
		want    *PageToken
		wantErr bool
	}{
		{
			name: "decode valid token",
			args: struct {
				s      string
				filter string
			}{
				s:      "eyJGaWx0ZXIiOiJlbmFibGVkPXRydWUiLCJTa2lwIjo1fQ==",
				filter: "enabled=true",
			},
			want: &PageToken{
				Filter: "enabled=true",
				Skip:   5,
			},
		},
		{
			name: "filter mismatch",
			args: struct {
				s      string
				filter string
			}{
				s:      "eyJGaWx0ZXIiOiJlbmFibGVkPXRydWUiLCJTa2lwIjo1fQ==",
				filter: "enabled=false",
			},
			wantErr: true,
		},
		{
			name: "invalid base64",
			args: struct {
				s      string
				filter string
			}{
				s:      "xx",
				filter: "",
			},
			wantErr: true,
		},
	}
	printer := message.NewPrinter(language.English)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got, err := DecodePageToken(printer, tt.args.s, tt.args.filter)
			assert.Equal(tt.wantErr, err != nil, "Error result was not as expected")
			assert.Equal(got, tt.want)
		})
	}
}

func (suite *Suite) TestEncodePageToken() {
	t := suite.T()
	type args struct {
		token *PageToken
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "encode valid token",
			args: struct{ token *PageToken }{token: &PageToken{
				Filter: "enabled=true",
				Skip:   5,
			}},
			want: "eyJGaWx0ZXIiOiJlbmFibGVkPXRydWUiLCJTa2lwIjo1fQ==",
		},
	}
	printer := message.NewPrinter(language.English)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got, err := EncodePageToken(printer, tt.args.token)
			assert.Equal(tt.wantErr, err != nil, "Error result was not as expected")
			assert.Equal(got, tt.want)
		})
	}
}

func (suite *Suite) TestSetPort() {
	t := suite.T()
	type args struct {
		port string
	}
	tests := []struct {
		name            string
		args            args
		wantPort        string
		wantErrorString string
	}{
		{
			name: "valid port 1234",
			args: args{
				port: "1234",
			},
			wantPort:        "1234",
			wantErrorString: "",
		},
		{
			name: "invalid port example",
			args: args{
				port: "example",
			},
			wantErrorString: "unable to convert given port 'example' to number",
		},
		{
			name: "invalid port -1",
			args: args{
				port: "-1",
			},
			wantErrorString: "port number -1 is invalid because it is less or equal 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			srv := Server{}
			err := SetPort(tt.args.port)(&srv)
			if tt.wantErrorString != "" {
				assert.EqualError(err, tt.wantErrorString)
			} else {
				assert.Nil(err)
				assert.Equal(tt.wantPort, srv.port)
			}
		})
	}
}
