package store

//go:generate gotext -srclang=en update -out=catalog.go -lang=en,de

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rbicker/gooser/internal/utils"

	"golang.org/x/text/message"

	"github.com/golang/protobuf/ptypes"

	gooserv1 "github.com/rbicker/gooser/api/proto/v1"
	"golang.org/x/crypto/bcrypt"
)

// Store abstracts saving and receiving data.
type Store interface {
	ListUsers(ctx context.Context, printer *message.Printer, filter string, size, skip int32) (*[]User, int32, error)
	GetUser(ctx context.Context, printer *message.Printer, id string) (*User, error)
	GetUserByUsername(ctx context.Context, printer *message.Printer, username string) (*User, error)
	GetUserByMail(ctx context.Context, printer *message.Printer, mail string) (*User, error)
	GetUserByConfirmToken(ctx context.Context, printer *message.Printer, token string) (*User, error)
	GetUserByPasswordResetToken(ctx context.Context, printer *message.Printer, token string) (*User, error)
	SaveUser(ctx context.Context, printer *message.Printer, user *User) (*User, error)
	DeleteUser(ctx context.Context, printer *message.Printer, id string) error
	ListGroups(ctx context.Context, printer *message.Printer, filter string, size, skip int32) (*[]Group, int32, error)
	GetGroup(ctx context.Context, printer *message.Printer, id string) (*Group, error)
	GetGroupByName(ctx context.Context, printer *message.Printer, name string) (*Group, error)
	SaveGroup(ctx context.Context, printer *message.Printer, group *Group) (*Group, error)
	DeleteGroup(ctx context.Context, printer *message.Printer, id string) error
}

// User represents a user document.
type User struct {
	Id                 string    `bson:"_id,omitempty"`
	CreatedAt          time.Time `bson:"createdAt"`
	UpdatedAt          time.Time `bson:"updatedAt"`
	Username           string    `bson:"username"`
	Mail               string    `bson:"mail"`
	Password           string    `bson:"password,omitempty"`
	Language           string    `bson:"language"`
	Roles              []string  `bson:"roles,omitempty"`
	Confirmed          bool      `bson:"confirmed"`
	ConfirmToken       string    `bson:"confirmToken"`
	PasswordResetToken string    `bson:"passwordResetToken"`
}

// Group represents a group document.
type Group struct {
	Id        string    `bson:"_id,omitempty"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
	Name      string    `bson:"name"`
	Roles     []string  `bson:"roles,omitempty"`
	Members   []string  `bson:"members,omitempty"`
}

// ValidatePassword checks if the given plain text password
// matches with the user's password.
func (u *User) ValidatePassword(plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plain))
	if err != nil {
		return false
	}
	return true
}

// HasRole checks if the user has the given role.
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type Confirmation struct {
	Mail      string
	CreatedAt time.Time
}

// GenerateConfirmToken generates a new confirmation token for the given user.
// The token is assigned to the user, however the user has to be save to the
// database after generating the token.
func (u *User) GenerateConfirmToken(printer *message.Printer, key string) error {
	c := Confirmation{
		Mail:      u.Mail,
		CreatedAt: time.Now(),
	}
	b, err := json.Marshal(c)
	if err != nil {
		return status.Errorf(codes.Internal, printer.Sprintf("unable to json marshal confirmation: %s", err))
	}
	enc, err := utils.Encrypt(key, string(b))
	if err != nil {
		return status.Errorf(codes.Internal, printer.Sprintf("unable to encrypt confirmation: %s", err))
	}
	u.ConfirmToken = enc
	return nil
}

// ValidateConfirmToken checks if the given confirmation token is valid for the user.
func (u *User) ValidateConfirmToken(printer *message.Printer, key, token string) error {
	if token == "" {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("no token given"))
	}
	if u.ConfirmToken != token {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("token mismatch"))
	}
	msg, err := utils.Decrypt(key, token)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid token"))
	}
	c := &Confirmation{}
	err = json.Unmarshal([]byte(msg), c)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid token"))
	}
	if u.Mail != c.Mail {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid token"))
	}
	return nil
}

// ResetPassword provides the content for the reset password token.
type ResetPassword struct {
	CreatedAt time.Time
}

// GeneratePasswordResetToken generates a new token to reset the password for the given user.
// The token is assigned to the user, however the user has to be save to the
// database after generating the token.
func (u *User) GeneratePasswordResetToken(printer *message.Printer, key string) error {
	r := ResetPassword{
		CreatedAt: time.Now(),
	}
	b, err := json.Marshal(r)
	if err != nil {
		return status.Errorf(codes.Internal, printer.Sprintf("unable to json marshal reset password struct: %s", err))
	}
	enc, err := utils.Encrypt(key, string(b))
	if err != nil {
		return status.Errorf(codes.Internal, printer.Sprintf("unable to encrypt reset password struct: %s", err))
	}
	u.PasswordResetToken = enc
	return nil
}

// ToPb returns a protobuf representation of the user.
func (u *User) ToPb() *gooserv1.User {
	createdAt, _ := ptypes.TimestampProto(u.CreatedAt)
	updatedAt, _ := ptypes.TimestampProto(u.UpdatedAt)
	return &gooserv1.User{
		Id:        u.Id,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Username:  u.Username,
		Mail:      u.Mail,
		Roles:     u.Roles,
		Confirmed: u.Confirmed,
		Language:  u.Language,
		// do not return password
		// Password: u.Password,
	}
}

// PbToUser converts the given protobuf user into a
// store user.
func PbToUser(u *gooserv1.User) *User {
	createdAt, _ := ptypes.Timestamp(u.CreatedAt)
	updatedAt, _ := ptypes.Timestamp(u.UpdatedAt)
	return &User{
		Id:        u.GetId(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Username:  u.GetUsername(),
		Mail:      u.GetMail(),
		Roles:     u.Roles,
		Confirmed: u.GetConfirmed(),
		Password:  u.GetPassword(),
		Language:  u.Language,
	}
}

// ToPb returns a protobuf representation of the group.
func (g *Group) ToPb() *gooserv1.Group {
	createdAt, _ := ptypes.TimestampProto(g.CreatedAt)
	updatedAt, _ := ptypes.TimestampProto(g.UpdatedAt)
	return &gooserv1.Group{
		Id:        g.Id,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Name:      g.Name,
		Roles:     g.Roles,
		Members:   g.Members,
	}
}

// PbToGroup converts the given protobuf group into
// a store group.
func PbToGroup(g *gooserv1.Group) *Group {
	createdAt, _ := ptypes.Timestamp(g.CreatedAt)
	updatedAt, _ := ptypes.Timestamp(g.UpdatedAt)
	return &Group{
		Id:        g.GetId(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Name:      g.GetName(),
		Roles:     g.GetRoles(),
		Members:   g.GetMembers(),
	}
}
