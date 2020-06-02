package server

import (
	"context"
	"testing"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/rbicker/gooser/internal/mocks"
	"github.com/rbicker/gooser/internal/store"

	mock "github.com/stretchr/testify/mock"

	gooserv1 "github.com/rbicker/gooser/api/proto/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *Suite) TestListUsers() {
	t := suite.T()
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	// test page token
	printer := message.NewPrinter(language.English)
	pageTokenString, err := EncodePageToken(printer, &PageToken{
		Filter: "roles==tester",
		Skip:   3,
	})
	if err != nil {
		t.Fatalf("error while creating page token: %s", err)
	}
	// tests
	tests := []struct {
		name          string
		prepare       func(db *mocks.Store)
		accessToken   string
		req           *gooserv1.ListRequest
		wantCode      codes.Code
		wantLen       int
		wantPageToken *PageToken
	}{
		{
			name:        "unauthenticated",
			accessToken: "",
			req: &gooserv1.ListRequest{
				PageSize: 1,
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "list limited",
			accessToken: "user",
			req: &gooserv1.ListRequest{
				PageSize: 1,
			},
			prepare: func(db *mocks.Store) {
				db.On("ListUsers", mock.Anything, mock.Anything, "", int32(1), int32(0)).Return(
					&[]store.User{
						{
							Id:       "user1",
							Username: "user1",
						},
					},
					int32(5),
					nil,
				).Once()
			},
			wantCode: codes.OK,
			wantLen:  1,
			wantPageToken: &PageToken{
				Filter: "",
				Skip:   1,
			},
		},
		{
			name:        "page token",
			accessToken: "user",
			req: &gooserv1.ListRequest{
				PageSize:  1,
				PageToken: pageTokenString,
				Filter:    "roles==tester",
			},
			prepare: func(db *mocks.Store) {
				db.On("ListUsers", mock.Anything, mock.Anything, "roles==tester", int32(1), int32(3)).Return(
					&[]store.User{
						{
							Id:       "user1",
							Username: "user1",
							Roles:    []string{"tester"},
						},
					},
					int32(5),
					nil,
				).Once()
			},
			wantCode: codes.OK,
			wantLen:  1,
			wantPageToken: &PageToken{
				Filter: "roles==tester",
				Skip:   4,
			},
		},
		{
			name:        "page token with filter mismatch",
			accessToken: "user",
			req: &gooserv1.ListRequest{
				PageSize:  1,
				PageToken: pageTokenString,
				Filter:    "roles==admin",
			},
			wantCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// prepare mock
			db := new(mocks.Store)
			if tt.prepare != nil {
				tt.prepare(db)
			}
			suite.srv.store = db
			// prepare context with access token
			ctx := context.Background()
			if tt.accessToken != "" {
				ctx = context.WithValue(ctx, "access_token", tt.accessToken)
			}
			// run function
			res, err := client.ListUsers(ctx, tt.req)
			// check status code
			code, _ := status.FromError(err)
			assert.Equal(tt.wantCode, code.Code(), "response statuscode mismatch")
			db.AssertExpectations(t)
			if code.Code() != codes.OK {
				// if status code is not ok, response should be nil
				assert.Nil(res)
				return
			}
			// check result
			assert.Equal(tt.wantLen, len(res.Users), "length mismatch")
			token, _ := DecodePageToken(printer, res.GetNextPageToken(), tt.req.GetFilter())
			assert.Equal(tt.wantPageToken, token)
		})
	}
}

func (suite *Suite) TestGetUser() {
	t := suite.T()
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	if err != nil {
		t.Fatalf("error while creating gooser client: %s", err)
	}
	// tests
	tests := []struct {
		name          string
		prepare       func(db *mocks.Store)
		accessToken   string
		req           *gooserv1.IdRequest
		wantCode      codes.Code
		wantId        string
		wantUsername  string
		wantRoles     []string
		wantConfirmed bool
	}{
		{
			name:        "unauthenticated",
			accessToken: "",
			req: &gooserv1.IdRequest{
				Id: "admins",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "valid query",
			accessToken: "user",
			req: &gooserv1.IdRequest{
				Id: "user1",
			},
			prepare: func(db *mocks.Store) {
				db.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.User {
						return &store.User{
							Id:        "user1",
							Username:  "user1",
							Roles:     []string{"tester"},
							Confirmed: true,
						}
					},
					nil).Once()
			},
			wantCode:      codes.OK,
			wantId:        "user1",
			wantUsername:  "user1",
			wantConfirmed: true,
			wantRoles:     []string{"tester"},
		},
		{
			name:        "not existing",
			accessToken: "user",
			req: &gooserv1.IdRequest{
				Id: "user1",
			},
			prepare: func(db *mocks.Store) {
				db.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(
					nil,
					status.Errorf(codes.NotFound, "user not found"),
				).Once()
			},
			wantCode: codes.NotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// prepare mock
			db := new(mocks.Store)
			if tt.prepare != nil {
				tt.prepare(db)
			}
			suite.srv.store = db
			// prepare context with access token
			ctx := context.Background()
			if tt.accessToken != "" {
				ctx = context.WithValue(ctx, "access_token", tt.accessToken)
			}
			// run function
			res, err := client.GetUser(ctx, tt.req)
			// check status code
			code, _ := status.FromError(err)
			assert.Equal(tt.wantCode, code.Code(), "response statuscode mismatch")
			db.AssertExpectations(t)
			if code.Code() != codes.OK {
				// if status code is not ok, response should be nil
				assert.Nil(res)
				return
			}
			// check result
			assert.Equal(tt.wantId, res.Id, "id mismatch")
			assert.Equal(tt.wantUsername, res.Username, "username mismatch")
			assert.Equal(tt.wantConfirmed, res.Confirmed, "confirmed mismatch")
			assert.ElementsMatch(tt.wantRoles, res.Roles, "roles mismatch")
		})
	}
}

func (suite *Suite) TestCreateUser() {
	t := suite.T()
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	if err != nil {
		t.Fatalf("error while creating gooser client: %s", err)
	}
	// tests
	tests := []struct {
		name          string
		prepare       func(db *mocks.Store, mailer *mocks.MessageDeliverer)
		accessToken   string
		req           *gooserv1.User
		wantCode      codes.Code
		wantId        string
		wantUsername  string
		wantConfirmed bool
	}{
		{
			name: "no username",
			req: &gooserv1.User{
				Mail:     "new@testing.com",
				Password: "password1234",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "no mail",
			req: &gooserv1.User{
				Username: "new",
				Password: "password1234",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "trying to set confirmed",
			req: &gooserv1.User{
				Username:  "new",
				Mail:      "new@testing.com",
				Password:  "password1234",
				Confirmed: true,
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name: "trying to set roles",
			req: &gooserv1.User{
				Username: "new",
				Mail:     "new@testing.com",
				Password: "password1234",
				Roles:    []string{"admins"},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "username regex",
			req: &gooserv1.User{
				Username: "new@testing.com",
				Mail:     "new@testing.com",
				Password: "password1234",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "short username",
			req: &gooserv1.User{
				Username: "ne",
				Mail:     "new@testing.com",
				Password: "password1234",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "short password",
			req: &gooserv1.User{
				Username: "new",
				Mail:     "new@testing.com",
				Password: "pass",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "duplicate username or mail",
			req: &gooserv1.User{
				Username: "user1",
				Mail:     "new@testing.com",
				Password: "password1234",
			},
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("ListUsers", mock.Anything, mock.Anything, `username=="user1",mail=="new@testing.com"`, int32(1), int32(0)).Return(
					&[]store.User{
						{
							Id:       "user1",
							Username: "user1",
							Mail:     "user1@testing.com",
						},
					},
					int32(1),
					nil,
				).Once()
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "valid user",
			req: &gooserv1.User{
				Username: "new",
				Mail:     "new@testing.com",
				Password: "password1234",
			},
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("ListUsers", mock.Anything, mock.Anything, `username=="new",mail=="new@testing.com"`, int32(1), int32(0)).Return(
					nil,
					int32(0),
					nil,
				).Once()
				db.On("SaveUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
				mailer.On("SendConfirmToken", mock.Anything).Return(nil).Once()
			},
			wantCode:      codes.OK,
			wantId:        "", // should not be set by CreateUser function
			wantUsername:  "new",
			wantConfirmed: false,
		},
		{
			name:        "setting confirmed as admin",
			accessToken: "admin",
			req: &gooserv1.User{
				Username:  "new",
				Mail:      "new@testing.com",
				Password:  "password1234",
				Confirmed: true,
			},
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("ListUsers", mock.Anything, mock.Anything, `username=="new",mail=="new@testing.com"`, int32(1), int32(0)).Return(
					nil,
					int32(0),
					nil,
				).Once()
				db.On("SaveUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
			},
			wantCode:      codes.OK,
			wantUsername:  "new",
			wantConfirmed: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// prepare mock
			db := new(mocks.Store)
			mailer := new(mocks.MessageDeliverer)
			if tt.prepare != nil {
				tt.prepare(db, mailer)
			}
			suite.srv.store = db
			suite.srv.mailer = mailer
			// prepare context with access token
			ctx := context.Background()
			if tt.accessToken != "" {
				ctx = context.WithValue(ctx, "access_token", tt.accessToken)
			}
			// run function
			res, err := client.CreateUser(ctx, tt.req)
			// check status code
			code, _ := status.FromError(err)
			assert.Equal(tt.wantCode, code.Code(), "response statuscode mismatch")
			db.AssertExpectations(t)
			if code.Code() != codes.OK {
				// if status code is not ok, response should be nil
				assert.Nil(res)
				return
			}
			// check result
			assert.Equal(tt.wantId, res.Id, "id mismatch")
			assert.Equal(tt.wantUsername, res.Username, "username mismatch")
			assert.Equal(tt.wantConfirmed, res.Confirmed, "confirmed mismatch")
		})
	}
}

func (suite *Suite) TestUpdateUser() {
	t := suite.T()
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	if err != nil {
		t.Fatalf("error while creating gooser client: %s", err)
	}
	// tests
	tests := []struct {
		name          string
		prepare       func(db *mocks.Store, mailer *mocks.MessageDeliverer)
		accessToken   string
		req           *gooserv1.UpdateUserRequest
		wantCode      codes.Code
		wantId        string
		wantUsername  string
		wantMail      string
		wantConfirmed bool
	}{
		{
			name: "unauthenticated",
			req: &gooserv1.UpdateUserRequest{
				User: &gooserv1.User{
					Id:   "user1",
					Mail: "new@testing.com",
				},
				FieldMask: &field_mask.FieldMask{
					Paths: []string{"mail"},
				},
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "changing password",
			accessToken: "user1",
			req: &gooserv1.UpdateUserRequest{
				User: &gooserv1.User{
					Id:       "user1",
					Password: "New1234",
				},
				FieldMask: &field_mask.FieldMask{
					Paths: []string{"password"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:        "setting roles",
			accessToken: "user1",
			req: &gooserv1.UpdateUserRequest{
				User: &gooserv1.User{
					Id:    "user1",
					Roles: []string{"workers"},
				},
				FieldMask: &field_mask.FieldMask{
					Paths: []string{"roles"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:        "changing other user",
			accessToken: "user1",
			req: &gooserv1.UpdateUserRequest{
				User: &gooserv1.User{
					Id:   "admin",
					Mail: "user1@testing.com",
				},
				FieldMask: &field_mask.FieldMask{
					Paths: []string{"mail"},
				},
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name:        "valid update",
			accessToken: "user1",
			req: &gooserv1.UpdateUserRequest{
				User: &gooserv1.User{
					Id:       "user1",
					Username: "new",
					Mail:     "new@testing.com",
				},
				FieldMask: &field_mask.FieldMask{
					Paths: []string{"username", "mail"},
				},
			},
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.User {
						return &store.User{
							Id:        "user1",
							Username:  "user1",
							Mail:      "user1@testing.com",
							Language:  "en",
							Roles:     []string{"tester"},
							Confirmed: true,
						}
					},
					nil).Once()
				db.On("ListUsers", mock.Anything, mock.Anything, `(_id!oid="user1");(username=="new",mail=="new@testing.com")`, int32(1), int32(0)).Return(
					nil,
					int32(0),
					nil,
				).Once()
				db.On("SaveUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
				mailer.On("SendConfirmToken", mock.Anything).Return(nil).Once()
			},
			wantCode:      codes.OK,
			wantUsername:  "new",
			wantId:        "user1",
			wantConfirmed: false, // changing mail address should reset confirmed
			wantMail:      "new@testing.com",
		},
		{
			name:        "changing other user as admin",
			accessToken: "admin",
			req: &gooserv1.UpdateUserRequest{
				User: &gooserv1.User{
					Id:       "user1",
					Username: "new",
					Mail:     "new@testing.com",
				},
				FieldMask: &field_mask.FieldMask{
					Paths: []string{"username", "mail"},
				},
			},
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.User {
						return &store.User{
							Id:        "user1",
							Username:  "user1",
							Mail:      "user1@testing.com",
							Language:  "en",
							Roles:     []string{"tester"},
							Confirmed: true,
						}
					},
					nil).Once()
				db.On("ListUsers", mock.Anything, mock.Anything, `(_id!oid="user1");(username=="new",mail=="new@testing.com")`, int32(1), int32(0)).Return(
					nil,
					int32(0),
					nil,
				).Once()
				db.On("SaveUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
			},
			wantCode:      codes.OK,
			wantUsername:  "new",
			wantId:        "user1",
			wantConfirmed: true,
			wantMail:      "new@testing.com",
		},
		{
			name:        "remove mail address",
			accessToken: "user1",
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.User {
						return &store.User{
							Id:        "user1",
							Username:  "user1",
							Mail:      "user1@testing.com",
							Roles:     []string{"tester"},
							Confirmed: true,
						}
					},
					nil).Once()
			},
			req: &gooserv1.UpdateUserRequest{
				User: &gooserv1.User{
					Id:   "user1",
					Mail: "",
				},
				FieldMask: &field_mask.FieldMask{
					Paths: []string{"mail"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:        "remove mail address as admin",
			accessToken: "admin",
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.User {
						return &store.User{
							Id:        "user1",
							Username:  "user1",
							Mail:      "user1@testing.com",
							Language:  "en",
							Roles:     []string{"tester"},
							Confirmed: true,
						}
					},
					nil).Once()
				db.On("ListUsers", mock.Anything, mock.Anything, `(_id!oid="user1");(username=="user1")`, int32(1), int32(0)).Return(
					nil,
					int32(0),
					nil,
				).Once()
				db.On("SaveUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
			},
			req: &gooserv1.UpdateUserRequest{
				User: &gooserv1.User{
					Id:   "user1",
					Mail: "",
				},
				FieldMask: &field_mask.FieldMask{
					Paths: []string{"mail"},
				},
			},
			wantCode:      codes.OK,
			wantUsername:  "user1",
			wantId:        "user1",
			wantConfirmed: true,
			wantMail:      "",
		},
		{
			name:        "mail or username duplicate",
			accessToken: "user1",
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.User {
						return &store.User{
							Id:        "user1",
							Username:  "user1",
							Mail:      "user1@testing.com",
							Language:  "en",
							Roles:     []string{"tester"},
							Confirmed: true,
						}
					},
					nil).Once()
				db.On("ListUsers", mock.Anything, mock.Anything, `(_id!oid="user1");(username=="user1",mail=="user2@testing.com")`, int32(1), int32(0)).Return(
					&[]store.User{
						{
							Id:       "user2",
							Username: "user2",
							Mail:     "user2@testing.com",
						},
					},
					int32(1),
					nil,
				).Once()
			},
			req: &gooserv1.UpdateUserRequest{
				User: &gooserv1.User{
					Id:   "user1",
					Mail: "user2@testing.com",
				},
				FieldMask: &field_mask.FieldMask{
					Paths: []string{"mail"},
				},
			},
			wantCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// prepare mock
			db := new(mocks.Store)
			mailer := new(mocks.MessageDeliverer)
			if tt.prepare != nil {
				tt.prepare(db, mailer)
			}
			suite.srv.store = db
			suite.srv.mailer = mailer
			// prepare context with access token
			ctx := context.Background()
			if tt.accessToken != "" {
				ctx = context.WithValue(ctx, "access_token", tt.accessToken)
			}
			// run function
			res, err := client.UpdateUser(ctx, tt.req)
			// check status code
			code, _ := status.FromError(err)
			assert.Equal(tt.wantCode, code.Code(), "response statuscode mismatch")
			db.AssertExpectations(t)
			if code.Code() != codes.OK {
				// if status code is not ok, response should be nil
				assert.Nil(res)
				return
			}
			// check result
			assert.Equal(tt.wantId, res.Id, "id mismatch")
			assert.Equal(tt.wantUsername, res.Username, "username mismatch")
			assert.Equal(tt.wantMail, res.Mail, "mail mismatch")
			assert.Equal(tt.wantConfirmed, res.Confirmed, "confirmed mismatch")
		})
	}
}

func (suite *Suite) TestDeleteUser() {
	t := suite.T()
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	if err != nil {
		t.Fatalf("error while creating gooser client: %s", err)
	}
	// tests
	tests := []struct {
		name        string
		prepare     func(db *mocks.Store)
		accessToken string
		req         *gooserv1.IdRequest
		wantCode    codes.Code
		wantId      string
		wantName    string
		wantRoles   []string
		wantMembers []string
	}{
		{
			name:        "unauthenticated",
			accessToken: "",
			req: &gooserv1.IdRequest{
				Id: "user1",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "permission denied",
			accessToken: "user1",
			req: &gooserv1.IdRequest{
				Id: "user1",
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name:        "valid request",
			accessToken: "admin",
			req: &gooserv1.IdRequest{
				Id: "user1",
			},
			prepare: func(db *mocks.Store) {
				db.On("ListGroups", mock.Anything, mock.Anything, `members=="user1"`, mock.Anything, mock.Anything).Return(
					&[]store.Group{
						{
							Id:      "testers",
							Name:    "testers",
							Members: []string{"user1"},
						},
					},
					int32(-1),
					nil,
				).Once()
				db.On("SaveGroup", mock.Anything, mock.Anything, mock.MatchedBy(
					func(group *store.Group) bool {
						return group.Name == "testers" && len(group.Members) == 0
					})).Return(
					func(ctx context.Context, printer *message.Printer, group *store.Group) *store.Group {
						return group
					},
					nil,
				).Once()
				db.On("DeleteUser", mock.Anything, mock.Anything, "user1").Return(
					nil,
				).Once()
			},
			wantCode: codes.OK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// prepare mock
			db := new(mocks.Store)
			if tt.prepare != nil {
				tt.prepare(db)
			}
			suite.srv.store = db
			// prepare context with access token
			ctx := context.Background()
			if tt.accessToken != "" {
				ctx = context.WithValue(ctx, "access_token", tt.accessToken)
			}
			// run function
			res, err := client.DeleteUser(ctx, tt.req)
			// check status code
			code, _ := status.FromError(err)
			assert.Equal(tt.wantCode, code.Code(), "response statuscode mismatch")
			db.AssertExpectations(t)
			if code.Code() != codes.OK {
				// if status code is not ok, response should be nil
				assert.Nil(res)
				return
			}
		})
	}
}

func (suite *Suite) TestChangePassword() {
	t := suite.T()
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	if err != nil {
		t.Fatalf("error while creating gooser client: %s", err)
	}
	// tests
	tests := []struct {
		name        string
		prepare     func(db *mocks.Store)
		accessToken string
		req         *gooserv1.ChangePasswordRequest
		wantCode    codes.Code
	}{
		{
			name:        "unauthenticated",
			accessToken: "",
			req: &gooserv1.ChangePasswordRequest{
				Id:          "user1",
				OldPassword: "password",
				NewPassword: "newPassword",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "wrong password request",
			accessToken: "user1",
			req: &gooserv1.ChangePasswordRequest{
				Id:          "user1",
				OldPassword: "wrong",
				NewPassword: "newPassword",
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name:        "valid request",
			accessToken: "user1",
			prepare: func(db *mocks.Store) {
				db.On("SaveUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
			},
			req: &gooserv1.ChangePasswordRequest{
				Id:          "user1",
				OldPassword: "password",
				NewPassword: "newPassword",
			},
			wantCode: codes.OK,
		},
		{
			name:        "reset password as admin",
			accessToken: "admin",
			prepare: func(db *mocks.Store) {
				db.On("GetUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.User {
						return &store.User{
							Id:        "user1",
							Username:  "user1",
							Mail:      "user1@testing.com",
							Roles:     []string{"tester"},
							Confirmed: true,
						}
					},
					nil).Once()
				db.On("SaveUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
			},
			req: &gooserv1.ChangePasswordRequest{
				Id:          "user1",
				OldPassword: "",
				NewPassword: "newPassword",
			},
			wantCode: codes.OK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// prepare mock
			db := new(mocks.Store)
			if tt.prepare != nil {
				tt.prepare(db)
			}
			suite.srv.store = db
			// prepare context with access token
			ctx := context.Background()
			if tt.accessToken != "" {
				ctx = context.WithValue(ctx, "access_token", tt.accessToken)
			}
			// run function
			res, err := client.ChangePassword(ctx, tt.req)
			// check status code
			code, _ := status.FromError(err)
			assert.Equal(tt.wantCode, code.Code(), "response statuscode mismatch")
			db.AssertExpectations(t)
			if code.Code() != codes.OK {
				// if status code is not ok, response should be nil
				assert.Nil(res)
				return
			}
		})
	}
}

func (suite *Suite) TestConfirmMail() {
	t := suite.T()
	printer := message.NewPrinter(language.English)
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	if err != nil {
		t.Fatalf("error while creating gooser client: %s", err)
	}
	user := store.User{
		Mail: "user1@testing.com",
	}
	if err := user.GenerateConfirmToken(printer, suite.srv.secret); err != nil {
		t.Fatalf("unable to generate confirm token: %s", err)
	}
	// tests
	tests := []struct {
		name     string
		prepare  func(db *mocks.Store)
		req      *gooserv1.ConfirmMailRequest
		wantCode codes.Code
	}{
		{
			name: "not found",
			prepare: func(db *mocks.Store) {
				db.On("GetUserByConfirmToken", mock.Anything, mock.Anything, mock.Anything).Return(
					nil,
					status.Errorf(codes.NotFound, "user not found"),
				)
			},
			req: &gooserv1.ConfirmMailRequest{
				Token: user.ConfirmToken,
			},
			wantCode: codes.NotFound,
		},
		{
			name: "valid request",
			prepare: func(db *mocks.Store) {
				db.On("GetUserByConfirmToken", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, token string) *store.User {
						u := &store.User{
							Id:           "user1",
							Username:     "user1",
							Mail:         "user1@testing.com",
							Confirmed:    false,
							ConfirmToken: user.ConfirmToken,
						}
						return u

					},
					nil,
				)
				// user should be confirmed
				db.On("SaveUser", mock.Anything, mock.Anything, mock.MatchedBy(func(user *store.User) bool {
					return user.Confirmed == true
				})).Return(func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
					return user
				},
					nil,
				)
			},
			req: &gooserv1.ConfirmMailRequest{
				Token: user.ConfirmToken,
			},
			wantCode: codes.OK,
		},
		{
			name: "mail address changed",
			prepare: func(db *mocks.Store) {
				db.On("GetUserByConfirmToken", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, token string) *store.User {
						u := &store.User{
							Id:           "user1",
							Username:     "user1",
							Mail:         "othermail@testing.com",
							Confirmed:    false,
							ConfirmToken: user.ConfirmToken,
						}
						return u

					},
					nil,
				)
			},
			req: &gooserv1.ConfirmMailRequest{
				Token: user.ConfirmToken,
			},
			wantCode: codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// prepare mock
			db := new(mocks.Store)
			if tt.prepare != nil {
				tt.prepare(db)
			}
			suite.srv.store = db
			// run function
			res, err := client.ConfirmMail(context.Background(), tt.req)
			// check status code
			code, _ := status.FromError(err)
			assert.Equal(tt.wantCode, code.Code(), "response statuscode mismatch")
			db.AssertExpectations(t)
			if code.Code() != codes.OK {
				// if status code is not ok, response should be nil
				assert.Nil(res)
				return
			}
		})
	}
}

func (suite *Suite) TestForgotPassword() {
	t := suite.T()
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	if err != nil {
		t.Fatalf("error while creating gooser client: %s", err)
	}
	// tests
	tests := []struct {
		name     string
		prepare  func(db *mocks.Store, mailer *mocks.MessageDeliverer)
		req      *gooserv1.ForgotPasswordRequest
		wantCode codes.Code
	}{
		{
			name: "mail not found",
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("GetUserByMail", mock.Anything, mock.Anything, "user1@testing.com").Return(
					nil,
					status.Errorf(codes.NotFound, "user not found"),
				).Once()
			},
			req: &gooserv1.ForgotPasswordRequest{
				Mail: "user1@testing.com",
			},
			wantCode: codes.NotFound,
		},
		{
			name: "by username",
			prepare: func(db *mocks.Store, mailer *mocks.MessageDeliverer) {
				db.On("GetUserByUsername", mock.Anything, mock.Anything, "user1").Return(
					&store.User{
						Username: "user1",
						Mail:     "user1@testing.com",
					},
					nil,
				).Once()
				db.On("SaveUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
				mailer.On("SendPasswordResetToken", mock.Anything).Return(nil).Once()
			},
			req: &gooserv1.ForgotPasswordRequest{
				Username: "user1",
			},
			wantCode: codes.OK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// prepare mock
			db := new(mocks.Store)
			mailer := new(mocks.MessageDeliverer)
			if tt.prepare != nil {
				tt.prepare(db, mailer)
			}
			suite.srv.store = db
			suite.srv.mailer = mailer
			// run function
			res, err := client.ForgotPassword(context.Background(), tt.req)
			// check status code
			code, _ := status.FromError(err)
			assert.Equal(tt.wantCode, code.Code(), "response statuscode mismatch")
			db.AssertExpectations(t)
			if code.Code() != codes.OK {
				// if status code is not ok, response should be nil
				assert.Nil(res)
				return
			}
		})
	}
}

func (suite *Suite) TestResetPassword() {
	t := suite.T()
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	user := &store.User{
		Id:                 "user1",
		Username:           "user1",
		Mail:               "user1@testing.com",
		Language:           "en",
		PasswordResetToken: "",
	}
	printer := message.NewPrinter(language.English)
	user.GeneratePasswordResetToken(printer, suite.srv.secret)
	// tests
	tests := []struct {
		name     string
		prepare  func(db *mocks.Store)
		req      *gooserv1.ResetPasswordRequest
		wantCode codes.Code
	}{
		{
			name: "empty token",
			req: &gooserv1.ResetPasswordRequest{
				Token: "",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "user not found",
			prepare: func(db *mocks.Store) {
				db.On("GetUserByPasswordResetToken", mock.Anything, mock.Anything, "xxx").Return(
					nil,
					status.Errorf(codes.NotFound, "user not found"),
				)
			},
			req: &gooserv1.ResetPasswordRequest{
				Token: "xxx",
			},
			wantCode: codes.NotFound,
		},
		{
			name: "short password",
			prepare: func(db *mocks.Store) {
				db.On("GetUserByPasswordResetToken", mock.Anything, mock.Anything, "xxx").Return(
					&store.User{
						Username:           "user1",
						Mail:               "user1@testing.com",
						PasswordResetToken: "xxx",
					},
					nil,
				)
			},
			req: &gooserv1.ResetPasswordRequest{
				Token:    "xxx",
				Password: "xxx",
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "valid",
			prepare: func(db *mocks.Store) {
				db.On("GetUserByPasswordResetToken", mock.Anything, mock.Anything, user.PasswordResetToken).Return(
					user,
					nil,
				)
				db.On("SaveUser", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				)
			},
			req: &gooserv1.ResetPasswordRequest{
				Token:    user.PasswordResetToken,
				Password: "1234567",
			},
			wantCode: codes.OK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			// prepare mock
			db := new(mocks.Store)
			if tt.prepare != nil {
				tt.prepare(db)
			}
			suite.srv.store = db
			// run function
			res, err := client.ResetPassword(context.Background(), tt.req)
			// check status code
			code, _ := status.FromError(err)
			assert.Equal(tt.wantCode, code.Code(), "response statuscode mismatch")
			db.AssertExpectations(t)
			if code.Code() != codes.OK {
				// if status code is not ok, response should be nil
				assert.Nil(res)
				return
			}
		})
	}
}
