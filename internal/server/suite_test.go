package server

import (
	"context"
	"net"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/rbicker/gooser/internal/mocks"
	"github.com/rbicker/gooser/internal/store"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

// Suite
type Suite struct {
	suite.Suite
	srv      *Server
	listener *bufconn.Listener
	// client     *gooserv1.GooserClient
	// clientConn *grpc.ClientConn
	mockStore *mocks.Store
}

// SetupSuite runs once before all tests
func (suite *Suite) SetupSuite() {
	t := suite.T()
	var srvOpts []func(*Server) error
	// store
	db := new(mocks.Store)
	// mock function srv.InitCollections()
	db.On("GetUserByUsername", mock.Anything, "admin").Return(
		&store.User{
			Id:        "admin",
			Username:  "admin",
			Roles:     []string{"admin"},
			Confirmed: true,
		},
		nil,
	)
	// stub the WithContextUserReceiver
	srvOpts = append(srvOpts, WithContextUserReceiver(func(ctx context.Context, db store.Store) (*store.User, error) {
		accessToken, ok := ctx.Value("access_token").(string)
		if !ok || accessToken == "" {
			return nil, nil
		}
		plainPassword := "password"
		hashed, _ := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
		ss := strings.Split(accessToken, ",")
		return &store.User{
			Id:        ss[0],
			Username:  ss[0],
			Password:  string(hashed),
			Mail:      ss[0],
			Roles:     ss,
			Confirmed: true,
		}, nil
	}))
	// create in-memory listener
	suite.listener = bufconn.Listen(1024 * 1024)
	srvOpts = append(srvOpts, WithListener(suite.listener))
	// oauthClient
	oauth := new(mocks.UserLookup)
	// mailer
	mailer := new(mocks.MessageDeliverer)
	// create test grpc server
	srv, err := NewServer("secret", db, oauth, mailer, srvOpts...)
	if err != nil {
		t.Fatalf("unable to create server with buffer connection: %s", err)
	}
	suite.srv = srv
	// run server
	go func() {
		if err := suite.srv.Serve(); err != nil {
			t.Fatalf("grpc server failed: %s", err)
		}
	}()
}

// NewClientConnection creates a new grpc client connection.
func (suite *Suite) NewClientConnection(meta map[string]string) (*grpc.ClientConn, error) {
	// create dialer for client
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return suite.listener.Dial()
	}
	unaryInterceptor := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		meta := make(map[string]string)
		accessToken, ok := ctx.Value("access_token").(string)
		if ok && accessToken != "" {
			meta["access_token"] = accessToken
		}
		md := metadata.New(meta)
		ctx = metadata.NewOutgoingContext(context.TODO(), md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
	return grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithUnaryInterceptor(unaryInterceptor),
		grpc.WithInsecure(),
	)
}

// TearDownSuite runs once after all tests.
func (suite *Suite) TearDownSuite() {
	suite.srv.Stop()
	// suite.clientConn.Close()
}

// TestSuite is needed in order for 'go test' to run this suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}
