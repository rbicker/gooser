package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
