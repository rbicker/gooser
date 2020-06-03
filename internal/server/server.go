package server

//go:generate gotext -srclang=en update -out=catalog.go -lang=en,de

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/rbicker/gooser/internal/mailer"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/rbicker/gooser/internal/auth"

	"google.golang.org/grpc/metadata"

	gooserv1 "github.com/rbicker/gooser/api/proto/v1"
	"github.com/rbicker/gooser/internal/store"
	"github.com/rbicker/gooser/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server implements the gooser server.
type Server struct {
	secret              string
	port                string
	store               store.Store
	mailer              mailer.Messenger
	grpcServer          *grpc.Server
	useReflection       bool
	listener            net.Listener
	authClient          auth.UserLookup
	errorLogger         *log.Logger
	infoLogger          *log.Logger
	contextUserReceiver func(ctx context.Context, db store.Store) (*store.User, error)
}

// PageToken represents a pagination token.
type PageToken struct {
	Filter string
	Skip   int32
}

// ensure server implements gooserv1.GooserServer
var _ gooserv1.GooserServer = &Server{}

// NewServer returns a new gooser server.
func NewServer(secret string, db store.Store, authClient auth.UserLookup, mailer mailer.Messenger, opts ...func(*Server) error) (*Server, error) {
	printer := message.NewPrinter(language.English)
	// create server
	var srv = Server{
		infoLogger:  log.New(os.Stdout, "INFO: ", log.Lmsgprefix+log.LstdFlags),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.Lmsgprefix+log.LstdFlags),
		port:        "50051", // default port
		authClient:  authClient,
		store:       db,
		mailer:      mailer,
	}
	// run functional options
	for _, op := range opts {
		err := op(&srv)
		if err != nil {
			return nil, fmt.Errorf("setting option failed: %w", err)
		}
	}
	// hash secret key
	h := md5.New()
	if _, err := io.WriteString(h, secret); err != nil {
		return nil, fmt.Errorf("unable to hash secret key: %w", err)
	}
	srv.secret = fmt.Sprintf("%x", h.Sum(nil))
	// user from context receiver
	if srv.contextUserReceiver == nil {
		srv.contextUserReceiver = func(ctx context.Context, db store.Store) (*store.User, error) {
			accessToken, ok := ctx.Value("access_token").(string)
			if !ok || accessToken == "" {
				return nil, nil
			}
			id, err := srv.authClient.GetUserIDbyToken(accessToken)
			if err != nil {
				return nil, err
			}
			user, err := srv.store.GetUser(ctx, printer, id)
			// if user was not found, turn the error into an unauthorized one
			if code, _ := status.FromError(err); code.Code() == codes.NotFound {
				return nil, status.Errorf(codes.Unauthenticated, "user not found")
			}
			return user, err
		}
	}
	// unary server interceptor
	unaryInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Internal, "retrieving metadata failed")
		}
		if header, ok := md["access_token"]; ok {
			token := header[0]
			ctx = context.WithValue(ctx, "access_token", token)
		}
		return handler(ctx, req)
	}
	// register grpc server
	srv.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
	)
	// enable reflection
	if srv.useReflection {
		reflection.Register(srv.grpcServer)
	}
	gooserv1.RegisterGooserServer(srv.grpcServer, &srv)
	return &srv, nil
}

// Serve starts serving the gooser server.
func (srv *Server) Serve() error {
	var err error
	if srv.listener == nil {
		// use tcp listener by default
		srv.listener, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", srv.port))
	}
	if err != nil {
		return fmt.Errorf("gooser server is unable to server: %w", err)
	}
	return srv.grpcServer.Serve(srv.listener)
}

// Stop stops the gooser server.
func (srv *Server) Stop() error {
	stopped := make(chan struct{})
	go func() {
		srv.grpcServer.GracefulStop()
		close(stopped)
	}()
	t := time.NewTimer(10 * time.Second)
	select {
	case <-t.C:
		srv.grpcServer.Stop()
	case <-stopped:
		t.Stop()
	}
	return nil
}

// InitCollections initializes the users and groups collections if necessary.
func (srv *Server) InitCollections(ctx context.Context) error {
	printer := message.NewPrinter(language.Make(utils.LookupEnv("GOOSER_DEFAULT_LANGUAGE", "en")))
	// handle admin user
	username := utils.LookupEnv("GOOSER_ADMIN_USERNAME", "admin")
	u, err := srv.store.GetUserByUsername(ctx, printer, username)
	if err != nil {
		// unexpected error
		if code, _ := status.FromError(err); code.Code() != codes.NotFound {
			return fmt.Errorf("unable to get admin user '%s': %w", username, err)
		}
		// user does not exist, create
		plain := utils.RandomString(15)
		hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("unable to hash password: %w", err)
		}
		u, err = srv.store.SaveUser(ctx, printer, &store.User{
			Username:  username,
			Password:  string(hashed),
			Roles:     []string{"admin"},
			Confirmed: true,
		})
		if err != nil {
			return fmt.Errorf("unable to create admin user '%s': %w", username, err)
		}
		srv.infoLogger.Printf("created admin user '%s' with password '%s' - change now", username, plain)
	} else if !u.HasRole("admin") {
		// fix user
		srv.infoLogger.Printf("adding role admin directly to user %s", username)
		u.Roles = append(u.Roles, "admin")
		u, err = srv.store.SaveUser(ctx, printer, u)
	}
	// group
	g, err := srv.store.GetGroupByName(ctx, printer, "admins")
	if err != nil {
		if code, _ := status.FromError(err); code.Code() != codes.NotFound {
			return fmt.Errorf("unable to get admins group: %w", err)
		}
		// create group if not exists
		_, err := srv.store.SaveGroup(ctx, printer, &store.Group{
			Name:    "admins",
			Roles:   []string{"admin"},
			Members: []string{u.Id},
		})
		if err != nil {
			return fmt.Errorf("unable to create admins-group: %w", err)
		}
		srv.infoLogger.Println("created admins group")
	} else {
		// group exists, check role
		var changed, foundRole, foundMember bool
		for _, r := range g.Roles {
			if r == "admin" {
				foundRole = true
				break
			}
		}
		if !foundRole {
			srv.infoLogger.Println("adding admin role to group admins")
			g.Roles = append(g.Roles, "admins")
			changed = true
		}
		for _, m := range g.Members {
			if m == u.Id {
				foundMember = true
				break
			}
		}
		if !foundMember {
			srv.infoLogger.Printf("adding user '%s' to group admins", username)
			g.Members = append(g.Members, u.Id)
			changed = true
		}
		if changed {
			_, err := srv.store.SaveGroup(ctx, printer, g)
			if err != nil {
				return fmt.Errorf("unable to update admins-group: %w", err)
			}
		}
	}
	return nil
}

// GetUserInfoFromContext returns the user corresponding
// to the access token in the given context.
func (srv *Server) GetUserFromContext(ctx context.Context) (*store.User, error) {
	return srv.contextUserReceiver(ctx, srv.store)
}

// SetPort sets the gooser server port.
func SetPort(port string) func(*Server) error {
	return func(srv *Server) error {
		i, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("unable to convert given port '%s' to number", port)
		}
		if i <= 0 {
			return fmt.Errorf("port number %s is invalid because it is less or equal 0", port)
		}
		srv.port = port
		return nil
	}
}

// WithListener instructs the server to use the given listener
// while service the grpc server.
func WithListener(listener net.Listener) func(*Server) error {
	return func(srv *Server) error {
		srv.listener = listener
		return nil
	}
}

// WithContextUserReceiver sets the function to receive the user from the context.
// Should only be used while testing.
func WithContextUserReceiver(f func(ctx context.Context, db store.Store) (*store.User, error)) func(*Server) error {
	return func(srv *Server) error {
		srv.contextUserReceiver = f
		return nil
	}
}

// EnableReflection instructs the grpc server to enable reflection.
func EnableReflection() func(*Server) error {
	return func(srv *Server) error {
		srv.useReflection = true
		return nil
	}
}

// EncodePageToken encodes the given token to a base64 string.
func EncodePageToken(printer *message.Printer, token *PageToken) (string, error) {
	b, err := json.Marshal(token)
	if err != nil {
		return "", status.Errorf(codes.InvalidArgument, printer.Sprintf("unable to marshal token to json: %s", err))
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// DecodePageToken decodes the given base64 string to a page token.
// The page token will be verified against the given filter.
func DecodePageToken(printer *message.Printer, s, filter string) (*PageToken, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid page token, unable to base64 decode: %s", err)
	}
	var t PageToken
	err = json.Unmarshal(b, &t)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid page token, unable to unmarshal to page token: %s", err))
	}
	if t.Filter != filter {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("mismatch between given filter %s and filter in page token %s", filter, t.Filter))
	}
	return &t, nil
}
