package server

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes/empty"
	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	gooserv1 "github.com/rbicker/gooser/api/proto/v1"
	"github.com/rbicker/gooser/internal/store"
	"github.com/rbicker/gooser/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListUsers lists the users.
func (srv *Server) ListUsers(ctx context.Context, req *gooserv1.ListRequest) (*gooserv1.ListUsersResponse, error) {
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	filter := req.GetFilter()
	users, totalSize, token, err := srv.store.ListUsers(ctx, printer, filter, "", req.GetPageToken(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	var pbUsers []*gooserv1.User
	var pageSize int32
	if users != nil {
		pageSize = int32(len(*users))
		for _, m := range *users {
			pbUsers = append(pbUsers, m.ToPb())
		}
	}
	return &gooserv1.ListUsersResponse{
		Users:         pbUsers,
		NextPageToken: token,
		PageSize:      pageSize,
		TotalSize:     totalSize,
	}, nil
}

// GetUser returns the user with the given id.
func (srv *Server) GetUser(ctx context.Context, req *gooserv1.IdRequest) (*gooserv1.User, error) {
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	id := req.GetId()
	if id == "" {
		// return own user by default
		return u.ToPb(), nil
	}
	user, err := srv.store.GetUser(ctx, printer, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, printer.Sprintf("could not find user with id %s", req.GetId()))
	}
	return user.ToPb(), nil
}

// ValidateUser validates a number of requirements, should be used before saving
// a user to the store.
func (srv *Server) ValidateUser(ctx context.Context, printer *message.Printer, user *gooserv1.User) error {
	id := user.GetId()
	username := user.GetUsername()
	mail := user.GetMail()
	if len(username) < 3 {
		return status.Errorf(codes.InvalidArgument, "username must have a length of 3 or higher")
	}
	reUsername := regexp.MustCompile(`^[a-z0-9]+$`)
	if !reUsername.MatchString(username) {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid username, only lowercase letters and numbers are allowed"))
	}
	if _, err := language.Parse(user.GetLanguage()); err != nil {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("could not parse given language"))
	}
	// rsql filter string for existing users
	filterString := fmt.Sprintf(`username=="%s"`, username)
	if mail != "" {
		// validate mail address
		if !utils.IsMailAddress(mail) {
			return status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid mail address"))
		}
		filterString = fmt.Sprintf(`%s,mail=="%s"`, filterString, mail)
	}
	if id != "" {
		filterString = fmt.Sprintf(`(_id!oid="%s");(%s)`, id, filterString)
	}
	size, err := srv.store.CountUsers(ctx, printer, filterString)
	if err != nil {
		return err
	}
	if size > 0 {
		return status.Errorf(codes.InvalidArgument, "username or mail address is already taken")
	}
	return nil
}

// CreateUser creates the given user.
func (srv *Server) CreateUser(ctx context.Context, user *gooserv1.User) (*gooserv1.User, error) {
	var isAdmin bool
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	// set default language
	if user.GetLanguage() == "" {
		user.Language = utils.LookupEnv("GOOSER_DEFAULT_LANGUAGE", "en")
	}
	lang := user.Language
	if u != nil {
		if u.HasRole("admin") {
			isAdmin = true
		}
		lang = u.Language
	}
	printer := message.NewPrinter(language.Make(lang))
	user.Id = ""
	if len(user.GetPassword()) < 7 {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("password must have a length of at least 7"))
	}
	if len(user.GetRoles()) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("roles cannot be assigned to users directly, use groups instead"))
	}
	if user.GetMail() == "" && !isAdmin {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("mail address not set"))
	}
	if user.GetConfirmed() && !isAdmin {
		return nil, status.Errorf(codes.PermissionDenied, printer.Sprintf("not allowed to set confirmed"))
	}
	// validate
	if err := srv.ValidateUser(ctx, printer, user); err != nil {
		return nil, err
	}
	// hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		srv.errorLogger.Printf("error while creating password hash: %s", err)
		return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to hash given password"))
	}
	user.Password = string(hashed)
	storeUser := store.PbToUser(user)
	// generate confirmation token if necessary
	if !user.GetConfirmed() && user.GetMail() != "" {
		if err := storeUser.GenerateConfirmToken(printer, srv.secret); err != nil {
			return nil, err
		}
		if err := srv.mailer.SendConfirmToken(storeUser); err != nil {
			return nil, err
		}
	}
	newUser, err := srv.store.SaveUser(ctx, printer, storeUser)
	if err != nil {
		return nil, err
	}
	return newUser.ToPb(), nil
}

// UpdateUser changes the given user in the database.
func (srv *Server) UpdateUser(ctx context.Context, req *gooserv1.UpdateUserRequest) (*gooserv1.User, error) {
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	id := req.GetUser().GetId()
	// by default, own user will be modified
	if id == "" {
		id = u.Id
	}
	if id != u.Id && !u.HasRole("admin") {
		return nil, status.Errorf(codes.PermissionDenied, printer.Sprintf("not allowed to edit other users"))
	}
	mask, err := fieldmask_utils.MaskFromProtoFieldMask(req.GetFieldMask(), generator.CamelCase)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unable to create generate field mask: %s", err)
	}
	// if roles were set
	if _, ok := mask.Get("Roles"); ok {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("roles cannot be assigned to users directly"))
	}
	// if password was changed
	if _, ok := mask.Get("Password"); ok {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("password cannot be changed using the UpdateUser function, use ChangePassword instead"))
	}
	// if confirmed was set and user is not admin
	if _, ok := mask.Get("Confirmed"); ok && !u.HasRole("admin") && req.GetUser().GetConfirmed() {
		return nil, status.Errorf(codes.PermissionDenied, printer.Sprintf("not allowed to set confirmed"))
	}
	// query existing user
	existing, err := srv.store.GetUser(ctx, printer, id)
	if err != nil {
		return nil, err
	}
	user := existing.ToPb()
	// copy request user to existing user with field mask applied
	err = fieldmask_utils.StructToStruct(mask, req.GetUser(), user)
	if err != nil {
		srv.errorLogger.Printf("unable to merge users: %s", err)
		return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to merge users"))
	}
	// validate
	if err := srv.ValidateUser(ctx, printer, user); err != nil {
		return nil, err
	}
	// if mail address is about to be updated
	if _, ok := mask.Get("Mail"); ok && !u.HasRole("admin") {
		// if mail address was really changed
		if existing.Mail != req.GetUser().GetMail() {
			// if new mail is not set by a non-admin
			if req.GetUser().GetMail() == "" && !u.HasRole("admin") {
				return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("mail address not set"))
			}
			if !u.HasRole("admin") {
				// if mail was updated by a non-admin,
				// confirmed is reset
				user.Confirmed = false
				// generate confirmation
				if err := u.GenerateConfirmToken(printer, srv.secret); err != nil {
					return nil, err
				}
				if err := srv.mailer.SendConfirmToken(u); err != nil {
					// unable to send confirmation token
					// this should not be a terminating error
					srv.errorLogger.Printf("unable to send confirmation token for user %s: %s", user.Username, err)
				}
			}
		}
	}
	updated, err := srv.store.SaveUser(ctx, printer, store.PbToUser(user))
	if err != nil {
		return nil, err
	}
	return updated.ToPb(), nil
}

// DeleteUser deletes the user with the given id from the store.
func (srv *Server) DeleteUser(ctx context.Context, req *gooserv1.IdRequest) (*empty.Empty, error) {
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	if !u.HasRole("admin") {
		return nil, status.Errorf(codes.PermissionDenied, printer.Sprintf("not allowed to delete user"))
	}
	id := req.GetId()
	if id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "empty id given")
	}
	filter := fmt.Sprintf(`members=="%s"`, id)
	groups, _, _, err := srv.store.ListGroups(ctx, printer, filter, "", "", -1)
	for _, g := range *groups {
		g.Members = utils.RemoveFromStringSlice(g.Members, id)
		_, err := srv.store.SaveGroup(ctx, printer, &g)
		if err != nil {
			srv.errorLogger.Printf("unable to remove user with id %s from group %s with id %s: %s", id, g.Name, g.Id, err)
			return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to remove user from group %s", g.Name))
		}
	}
	err = srv.store.DeleteUser(ctx, printer, id)
	return &empty.Empty{}, err
}

// ChangePassword can be used to change own password. The old and the new password need to be provided.
// Admins can use this function to reset passwords for other users. In this case, the
// old password is not needed.
func (srv *Server) ChangePassword(ctx context.Context, req *gooserv1.ChangePasswordRequest) (*empty.Empty, error) {
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	isAdmin := u.HasRole("admin")
	newPassword := req.GetNewPassword()
	if len(newPassword) < 7 {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("password must have a length of at least 7"))
	}
	id := req.GetId()
	// by default, own user will be modified
	if id == "" {
		id = u.Id
	}
	if id != u.Id {
		// only admins
		if !isAdmin {
			return nil, status.Errorf(codes.PermissionDenied, printer.Sprintf("not allowed to change password for other users"))
		}
		// query user
		u, err = srv.store.GetUser(ctx, printer, id)
		if err != nil {
			return nil, err
		}
	} else {
		if !u.ValidatePassword(req.GetOldPassword()) {
			return nil, status.Errorf(codes.PermissionDenied, printer.Sprintf("password mismatch"))
		}
	}
	// hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		srv.errorLogger.Printf("error while creating password hash: %s", err)
		return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to hash given password"))
	}
	u.Password = string(hashed)
	_, err = srv.store.SaveUser(ctx, printer, u)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// ConfirmMail tries to confirm the given mail address.
func (srv *Server) ConfirmMail(ctx context.Context, req *gooserv1.ConfirmMailRequest) (*empty.Empty, error) {
	printer := message.NewPrinter(language.Make(utils.LookupEnv("GOOSER_DEFAULT_LANGUAGE", "en")))
	user, err := srv.store.GetUserByConfirmToken(ctx, printer, req.GetToken())
	if err != nil {
		return nil, err
	}
	err = user.ValidateConfirmToken(printer, srv.secret, req.GetToken())
	if err != nil {
		return nil, err
	}
	user.Confirmed = true
	user.ConfirmToken = ""
	_, err = srv.store.SaveUser(ctx, printer, user)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// ForgotPassword generates a password reset token and sends the token to the user.
func (srv *Server) ForgotPassword(ctx context.Context, req *gooserv1.ForgotPasswordRequest) (*empty.Empty, error) {
	printer := message.NewPrinter(language.Make(utils.LookupEnv("GOOSER_DEFAULT_LANGUAGE", "en")))
	var user *store.User
	username, mail := req.GetUsername(), req.GetMail()
	if username != "" {
		user, _ = srv.store.GetUserByUsername(ctx, printer, username)
	}
	if user == nil && mail != "" {
		user, _ = srv.store.GetUserByMail(ctx, printer, mail)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}
	printer = message.NewPrinter(language.Make(user.Language))
	if user.Mail == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user does not have a mail address")
	}
	user.GeneratePasswordResetToken(printer, srv.secret)
	if _, err := srv.store.SaveUser(ctx, printer, user); err != nil {
		return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to save user"))
	}
	if err := srv.mailer.SendPasswordResetToken(user); err != nil {
		return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to send reset password mail"))
	}
	return &empty.Empty{}, nil
}

// ResetPassword tries to reset the password for the user matching with the given token.
func (srv *Server) ResetPassword(ctx context.Context, req *gooserv1.ResetPasswordRequest) (*empty.Empty, error) {
	printer := message.NewPrinter(language.Make(utils.LookupEnv("GOOSER_DEFAULT_LANGUAGE", "en")))
	token, password := req.GetToken(), req.GetPassword()
	if token == "" {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("no token given"))
	}
	user, err := srv.store.GetUserByPasswordResetToken(ctx, printer, token)
	if err != nil {
		return nil, err
	}
	printer = message.NewPrinter(language.Make(user.Language))
	if len(password) < 7 {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("password must have a length of at least 7"))
	}
	msg, err := utils.Decrypt(srv.secret, token)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid token"))
	}
	r := &store.ResetPassword{}
	err = json.Unmarshal([]byte(msg), r)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid token"))
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		srv.errorLogger.Printf("error while creating password hash: %s", err)
		return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to hash given password"))
	}
	user.PasswordResetToken = ""
	user.Password = string(hashed)
	if _, err := srv.store.SaveUser(ctx, printer, user); err != nil {
		return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to save user"))
	}
	return &empty.Empty{}, nil
}
