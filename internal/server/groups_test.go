package server

import (
	"context"
	"testing"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/rbicker/gooser/internal/mocks"
	"github.com/rbicker/gooser/internal/store"

	"google.golang.org/genproto/protobuf/field_mask"

	mock "github.com/stretchr/testify/mock"

	gooserv1 "github.com/rbicker/gooser/api/proto/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (suite *Suite) TestListGroups() {
	t := suite.T()
	printer := message.NewPrinter(language.English)
	// client connection
	conn, err := suite.NewClientConnection(nil)
	if err != nil {
		t.Fatalf("unable to create client connection: %s", err)
	}
	defer conn.Close()
	client := gooserv1.NewGooserClient(conn)
	// test page token
	pageTokenString, err := EncodePageToken(printer, &PageToken{
		Filter: "name==admins",
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
				db.On("ListGroups", mock.Anything, mock.Anything, "", int32(1), int32(0)).Return(
					&[]store.Group{
						{
							Id:      "testers",
							Name:    "testers",
							Roles:   []string{"tester"},
							Members: []string{"user"},
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
				Filter:    "name==admins",
			},
			prepare: func(db *mocks.Store) {
				db.On("ListGroups", mock.Anything, mock.Anything, "name==admins", int32(1), int32(3)).Return(
					&[]store.Group{
						{
							Id:      "testers",
							Name:    "testers",
							Roles:   []string{"tester"},
							Members: []string{"user"},
						},
					},
					int32(5),
					nil,
				).Once()
			},
			wantCode: codes.OK,
			wantLen:  1,
			wantPageToken: &PageToken{
				Filter: "name==admins",
				Skip:   4,
			},
		},
		{
			name:        "page token with filter mismatch",
			accessToken: "user",
			req: &gooserv1.ListRequest{
				PageSize:  1,
				PageToken: pageTokenString,
				Filter:    "name==changed",
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
			res, err := client.ListGroups(ctx, tt.req)
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
			assert.Equal(tt.wantLen, len(res.Groups), "length mismatch")
			token, _ := DecodePageToken(printer, res.GetNextPageToken(), tt.req.GetFilter())
			assert.Equal(tt.wantPageToken, token)
		})
	}
}

func (suite *Suite) TestGetGroup() {
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
				Id: "admins",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "valid query",
			accessToken: "user",
			req: &gooserv1.IdRequest{
				Id: "testers",
			},
			prepare: func(db *mocks.Store) {
				db.On("GetGroup", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.Group {
						return &store.Group{
							Id:      "testers",
							Name:    "testers",
							Members: []string{"user1", "user2"},
							Roles:   []string{"tester"},
						}
					},
					nil).Once()
			},
			wantCode:    codes.OK,
			wantId:      "testers",
			wantName:    "testers",
			wantMembers: []string{"user1", "user2"},
			wantRoles:   []string{"tester"},
		},
		{
			name:        "not existing",
			accessToken: "user",
			req: &gooserv1.IdRequest{
				Id: "testers",
			},
			prepare: func(db *mocks.Store) {
				db.On("GetGroup", mock.Anything, mock.Anything, mock.Anything).Return(
					nil,
					status.Errorf(codes.NotFound, "group not found"),
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
			res, err := client.GetGroup(ctx, tt.req)
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
			assert.Equal(tt.wantName, res.Name, "name mismatch")
			assert.ElementsMatch(tt.wantRoles, res.Roles, "roles mismatch")
			assert.ElementsMatch(tt.wantMembers, res.Members, "members mismatch")
		})
	}
}

func (suite *Suite) TestCreateGroup() {
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
		req         *gooserv1.Group
		wantCode    codes.Code
		wantId      string
		wantName    string
		wantRoles   []string
		wantMembers []string
	}{
		{
			name:        "unauthenticated",
			accessToken: "",
			req: &gooserv1.Group{
				Id: "user",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "permission denied",
			accessToken: "tester",
			req: &gooserv1.Group{
				Id: "user",
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name:        "valid request",
			accessToken: "admin",
			req: &gooserv1.Group{
				Id:      "xxx",
				Name:    "testers",
				Members: []string{"user1", "user2"},
				Roles:   []string{"tester"},
			},
			prepare: func(db *mocks.Store) {
				// checking for existing group
				db.On("ListGroups", mock.Anything, mock.Anything, `(name=="testers")`, mock.Anything, mock.Anything).Return(
					nil,
					int32(0),
					nil,
				).Once()
				db.On("ListUsers", mock.Anything, mock.Anything, `_id=oid=("user1","user2")`, mock.Anything, mock.Anything).Return(
					&[]store.User{
						{
							Id:       "user1",
							Username: "user1",
							Roles:    []string{"tester"},
						},
						{
							Id:       "user2",
							Username: "user2",
							Roles:    nil,
						},
					},
					int32(2), // size
					nil,      // error
				).Once()
				db.On("SaveGroup", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, group *store.Group) *store.Group {
						return group
					},
					nil,
				).Once()
				// user2 should be updated with the tester role
				db.On("SaveUser", mock.Anything, mock.Anything, mock.MatchedBy(func(user *store.User) bool {
					return user.Id == "user2" && user.Roles[0] == "tester"
				}), mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
			},
			wantCode:    codes.OK,
			wantId:      "", // id should be reset
			wantName:    "testers",
			wantMembers: []string{"user1", "user2"},
			wantRoles:   []string{"tester"},
		},
		{
			name:        "name not unique",
			accessToken: "admin",
			req: &gooserv1.Group{
				Name:    "testers",
				Members: []string{"user1", "user2"},
				Roles:   []string{"tester"},
			},
			prepare: func(db *mocks.Store) {
				// checking for existing group
				db.On("ListGroups", mock.Anything, mock.Anything, `(name=="testers")`, mock.Anything, mock.Anything).Return(
					&[]store.Group{
						{
							Id:   "testers",
							Name: "testers",
						},
					},
					int32(1),
					nil,
				).Once()
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name:        "inexistent member",
			accessToken: "admin",
			req: &gooserv1.Group{
				Id:      "xxx",
				Name:    "testers",
				Members: []string{"user1", "user2"},
				Roles:   []string{"tester"},
			},
			prepare: func(db *mocks.Store) {
				// checking for existing group
				db.On("ListGroups", mock.Anything, mock.Anything, `(name=="testers")`, mock.Anything, mock.Anything).Return(
					nil,
					int32(0),
					nil,
				).Once()
				db.On("ListUsers", mock.Anything, mock.Anything, `_id=oid=("user1","user2")`, mock.Anything, mock.Anything).Return(
					&[]store.User{
						{
							Id:       "user1",
							Username: "user1",
							Roles:    []string{"tester"},
						},
					},
					int32(1), // size
					nil,      // error
				).Once()
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
			res, err := client.CreateGroup(ctx, tt.req)
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
			assert.Equal(tt.wantName, res.Name, "name mismatch")
			assert.ElementsMatch(tt.wantRoles, res.Roles, "roles mismatch")
			assert.ElementsMatch(tt.wantMembers, res.Members, "members mismatch")
		})
	}
}

func (suite *Suite) TestUpdateGroup() {
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
		req         *gooserv1.UpdateGroupRequest
		wantCode    codes.Code
		wantId      string
		wantName    string
		wantRoles   []string
		wantMembers []string
	}{
		{
			name:        "unauthenticated",
			accessToken: "",
			req: &gooserv1.UpdateGroupRequest{
				Group: &gooserv1.Group{
					Id:   "testers",
					Name: "validators",
				},
				FieldMask: &field_mask.FieldMask{Paths: []string{"name"}},
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "permission denied",
			accessToken: "tester",
			req: &gooserv1.UpdateGroupRequest{
				Group: &gooserv1.Group{
					Id:   "testers",
					Name: "validators",
				},
				FieldMask: &field_mask.FieldMask{Paths: []string{"name"}},
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name:        "valid request",
			accessToken: "admin",
			req: &gooserv1.UpdateGroupRequest{
				Group: &gooserv1.Group{
					Id:      "testers",
					Name:    "validators",
					Roles:   []string{"validator"},
					Members: []string{"user2"},
				},
				FieldMask: &field_mask.FieldMask{Paths: []string{"name", "roles", "members"}},
			},
			prepare: func(db *mocks.Store) {
				db.On("GetGroup", mock.Anything, mock.Anything, "testers").Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.Group {
						return &store.Group{
							Id:      "testers",
							Name:    "testers",
							Members: []string{"user1"},
							Roles:   []string{"tester"},
						}
					},
					nil).Once()
				db.On("GetUser", mock.Anything, mock.Anything, "user1").Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.User {
						return &store.User{
							Id:       "user1",
							Username: "user1",
							Roles:    []string{"tester", "admin"},
						}
					},
					nil).Twice()
				// checking for existing group
				db.On("ListGroups", mock.Anything, mock.Anything, `(_id!oid="testers");(name=="validators")`, mock.Anything, mock.Anything).Return(
					nil,
					int32(0),
					nil,
				).Once()
				// while querying roles for removed members + while querying removed members
				db.On("ListGroups", mock.Anything, mock.Anything, `_id!oid="testers";members=="user1";roles=="tester"`, mock.Anything, mock.Anything).Return(
					nil,
					int32(0), // size
					nil,      // error
				).Twice()
				// while querying added roles + while querying added members
				db.On("ListUsers", mock.Anything, mock.Anything, `_id=oid=("user2")`, mock.Anything, mock.Anything).Return(
					&[]store.User{
						{
							Id:       "user2",
							Username: "user2",
							Roles:    nil,
						},
					},
					int32(1), // size
					nil,      // error
				).Twice()
				db.On("SaveGroup", mock.Anything, mock.Anything, mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, group *store.Group) *store.Group {
						return group
					},
					nil,
				).Once()
				// user1 should have tester role removed
				db.On("SaveUser", mock.Anything, mock.Anything, mock.MatchedBy(func(user *store.User) bool {
					return user.Id == "user1" && user.Roles[0] == "admin"
				}), mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Twice()
				// user2 should be updated with the validator role
				db.On("SaveUser", mock.Anything, mock.Anything, mock.MatchedBy(func(user *store.User) bool {
					return user.Id == "user2" && user.Roles[0] == "validator"
				}), mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Twice()
			},
			wantCode:    codes.OK,
			wantId:      "testers",
			wantName:    "validators",
			wantMembers: []string{"user2"},
			wantRoles:   []string{"validator"},
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
			res, err := client.UpdateGroup(ctx, tt.req)
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
			assert.Equal(tt.wantName, res.Name, "name mismatch")
			assert.ElementsMatch(tt.wantRoles, res.Roles, "roles mismatch")
			assert.ElementsMatch(tt.wantMembers, res.Members, "members mismatch")
		})
	}
}

func (suite *Suite) TestDeleteGroup() {
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
				Id: "admins",
			},
			wantCode: codes.Unauthenticated,
		},
		{
			name:        "permission denied",
			accessToken: "user",
			req: &gooserv1.IdRequest{
				Id: "admins",
			},
			wantCode: codes.PermissionDenied,
		},
		{
			name:        "valid request",
			accessToken: "admin",
			req: &gooserv1.IdRequest{
				Id: "testers",
			},
			prepare: func(db *mocks.Store) {
				db.On("GetGroup", mock.Anything, mock.Anything, "testers").Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.Group {
						return &store.Group{
							Id:      "testers",
							Name:    "testers",
							Members: []string{"user1"},
							Roles:   []string{"tester", "worker"},
						}
					},
					nil).Once()
				// checking if there is another group providing the tester role
				db.On("ListGroups", mock.Anything, mock.Anything, `_id!oid="testers";members=="user1";roles=="tester"`, mock.Anything, mock.Anything).Return(
					nil,
					int32(0), // size
					nil,      // error
				).Once()
				// checking if there is another group providing the worker role
				db.On("ListGroups", mock.Anything, mock.Anything, `_id!oid="testers";members=="user1";roles=="worker"`, mock.Anything, mock.Anything).Return(
					&[]store.Group{
						{
							Id:    "workers",
							Name:  "workers",
							Roles: []string{"worker"},
						},
					},
					int32(1), // size
					nil,      // error
				).Once()
				db.On("GetUser", mock.Anything, mock.Anything, "user1").Return(
					func(ctx context.Context, printer *message.Printer, id string) *store.User {
						return &store.User{
							Id:       "user1",
							Username: "user1",
							Roles:    []string{"tester", "worker"},
						}
					},
					nil).Once()
				// user1 should have tester role removed
				db.On("SaveUser", mock.Anything, mock.Anything, mock.MatchedBy(func(user *store.User) bool {
					return user.Id == "user1" && user.Roles[0] == "worker" && len(user.Roles) == 1
				}), mock.Anything).Return(
					func(ctx context.Context, printer *message.Printer, user *store.User) *store.User {
						return user
					},
					nil,
				).Once()
				db.On("DeleteGroup", mock.Anything, mock.Anything, "testers").Return(
					nil,
				)
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
			res, err := client.DeleteGroup(ctx, tt.req)
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
