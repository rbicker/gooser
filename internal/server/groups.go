package server

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	fieldmaskutils "github.com/mennanov/fieldmask-utils"

	"github.com/rbicker/gooser/internal/store"

	"github.com/rbicker/gooser/internal/utils"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes/empty"
	gooserv1 "github.com/rbicker/gooser/api/proto/v1"
)

// ListGroups lists the groups from the store.
func (srv *Server) ListGroups(ctx context.Context, req *gooserv1.ListRequest) (*gooserv1.ListGroupsResponse, error) {
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	reqToken := req.GetPageToken()
	filter := req.GetFilter()
	size := req.GetPageSize()
	var skip int32
	if reqToken != "" {
		token, err := DecodePageToken(printer, reqToken, filter)
		if err != nil {
			return nil, err
		}
		skip += token.Skip
	}
	groups, totalSize, err := srv.store.ListGroups(ctx, printer, filter, size, skip)
	if err != nil {
		return nil, err
	}
	var pbGroup []*gooserv1.Group
	var pageSize int32
	if groups != nil {
		pageSize = int32(len(*groups))
		for _, g := range *groups {
			pbGroup = append(pbGroup, g.ToPb())
		}
	}
	var token string
	if totalSize > pageSize+skip {
		token, err = EncodePageToken(printer, &PageToken{
			Filter: filter,
			Skip:   pageSize + skip,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to create page token: %s", err))
		}
	}
	return &gooserv1.ListGroupsResponse{
		Groups:        pbGroup,
		NextPageToken: token,
		PageSize:      pageSize,
		TotalSize:     totalSize,
	}, nil
}

// GetGroup returns the group with the given id.
func (srv *Server) GetGroup(ctx context.Context, req *gooserv1.IdRequest) (*gooserv1.Group, error) {
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	g, err := srv.store.GetGroup(ctx, printer, req.GetId())
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, status.Errorf(codes.NotFound, printer.Sprintf("could not find group with id %s", req.GetId()))
	}
	return g.ToPb(), nil
}

// ValidateGroup validates the given group. This function should be run
// before storing the group.
func (srv *Server) ValidateGroup(ctx context.Context, printer *message.Printer, group *gooserv1.Group) error {
	id := group.GetId()
	name := group.GetName()
	if len(name) < 3 {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("group name needs to have a length of at least 3"))
	}
	// rsql filter string for existing groups
	filter := fmt.Sprintf(`(name=="%s")`, name)
	if id != "" {
		filter = fmt.Sprintf(`(_id!oid="%s");%s`, id, filter)
	}
	_, size, err := srv.store.ListGroups(ctx, printer, filter, -1, 0)
	if code, _ := status.FromError(err); err != nil && code.Code() != codes.NotFound {
		srv.errorLogger.Printf("error while counting groups: %s", err)
		return status.Errorf(codes.Internal, printer.Sprintf("error while counting groups"))
	}
	if size > 0 {
		return status.Errorf(codes.InvalidArgument, "name is already taken")
	}
	return nil
}

// AddRolesToMembers ensures the users with the given ids have the given roles.
func (srv *Server) AddRolesToMembers(ctx context.Context, printer *message.Printer, memberIds []string, roles []string) error {
	if len(memberIds) > 0 {
		var filterIds []string
		for _, id := range memberIds {
			filterIds = append(filterIds, fmt.Sprintf(`"%s"`, id))
		}
		filter := fmt.Sprintf("_id=oid=(%s)", strings.Join(filterIds, ","))
		members, size, err := srv.store.ListUsers(ctx, printer, filter, int32(len(memberIds)), 0)
		if err != nil {
			srv.errorLogger.Printf("unable to query members: %s", err)
			return status.Errorf(codes.Internal, printer.Sprintf("unable to query members"))
		}
		if int(size) != len(memberIds) {
			return status.Errorf(codes.InvalidArgument, printer.Sprintf("only %v of %v given memberIds were found", int(size), len(memberIds)))
		}
		// add roles to members if necessary
		for _, m := range *members {
			var changed, c bool
			for _, r := range roles {
				m.Roles, c = utils.AppendUniqueString(m.Roles, r)
				changed = changed || c
			}
			if changed {
				if _, err := srv.store.SaveUser(ctx, printer, &m); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// RemoveRolesFromMembers removes the given roles from the users with the given ids. Before it removes the roles
// it makes sure that the users is not entitled to have the role because of any other group.
func (srv *Server) RemoveRolesFromMembers(ctx context.Context, printer *message.Printer, groupId string, memberIds []string, roles []string) error {
	for _, userId := range memberIds {
		var user *store.User
		for _, role := range roles {
			// check if any other group is providing the role
			filter := fmt.Sprintf(`_id!oid="%s";members=="%s";roles=="%s"`, groupId, userId, role)
			_, size, err := srv.store.ListGroups(ctx, printer, filter, int32(1), 0)
			if code, _ := status.FromError(err); err != nil && code.Code() != codes.NotFound {
				srv.errorLogger.Printf("error while looking for groups that are providing the role %s for the user %s: %s", role, userId, err)
				return status.Errorf(codes.Internal, printer.Sprintf("error while looking up groups"))
			}
			// no other group is providing the role
			// if the size is 0
			if size == 0 {
				if user == nil {
					user, err = srv.store.GetUser(ctx, printer, userId)
					if err != nil {
						srv.errorLogger.Printf("error while getting user with id %s: %s", userId, err)
						return status.Errorf(codes.Internal, printer.Sprintf("error while querying member"))
					}
				}
				user.Roles = utils.RemoveFromStringSlice(user.Roles, role)
			}
		}
		// user was queried / updated
		if user != nil {
			_, err := srv.store.SaveUser(ctx, printer, user)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CreateGroup creates the given group and places it in the store.
func (srv *Server) CreateGroup(ctx context.Context, group *gooserv1.Group) (*gooserv1.Group, error) {
	// check user
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	if !u.HasRole("admin") {
		return nil, status.Errorf(codes.PermissionDenied, printer.Sprintf("not allowed to create groups"))
	}
	group.Id = ""
	err = srv.ValidateGroup(ctx, printer, group)
	if err != nil {
		return nil, err
	}
	// make sure roles are unique
	group.Roles, _ = utils.UniqueStringSlice(group.Roles)
	// make sure members are unique
	group.Members, _ = utils.UniqueStringSlice(group.Members)
	// update members
	if err := srv.AddRolesToMembers(ctx, printer, group.GetMembers(), group.GetRoles()); err != nil {
		return nil, err
	}
	// save group
	newGroup, err := srv.store.SaveGroup(ctx, printer, store.PbToGroup(group))
	if err != nil {
		return nil, err
	}
	return newGroup.ToPb(), nil
}

// UpdateGroup changes the given group in the database.
func (srv *Server) UpdateGroup(ctx context.Context, req *gooserv1.UpdateGroupRequest) (*gooserv1.Group, error) {
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	if !u.HasRole("admin") {
		return nil, status.Errorf(codes.PermissionDenied, printer.Sprintf("not allowed to update groups"))
	}
	group := req.GetGroup()
	id := group.GetId()
	mask, err := fieldmaskutils.MaskFromProtoFieldMask(req.GetFieldMask(), generator.CamelCase)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("unable to create generate field mask: %s", err))
	}
	// get existing group
	existing, err := srv.store.GetGroup(ctx, printer, id)
	if err != nil {
		return nil, err
	}
	// make sure roles are unique
	group.Roles, _ = utils.UniqueStringSlice(group.Roles)
	// make sure members are unique
	group.Members, _ = utils.UniqueStringSlice(group.Members)
	// store existing slices values in vars
	existingMembers := make([]string, len(existing.Members))
	existingRoles := make([]string, len(existing.Roles))
	copy(existingMembers, existing.Members)
	copy(existingRoles, existing.Roles)
	// result to return in the end
	res := existing.ToPb()
	// copy given group to existing group with field mask applied
	err = fieldmaskutils.StructToStruct(mask, group, res)
	if err != nil {
		srv.errorLogger.Printf("unable to merge groups: %s", err)
		return nil, status.Errorf(codes.Internal, printer.Sprintf("unable to merge groups"))
	}
	// validate group
	err = srv.ValidateGroup(ctx, printer, res)
	if err != nil {
		return nil, err
	}
	// save group
	updated, err := srv.store.SaveGroup(ctx, printer, store.PbToGroup(res))
	// handle changes to roles
	if _, ok := mask.Get("Roles"); ok {
		addedRoles, removedRoles := utils.StringSlicesDiff(existingRoles, group.Roles)
		if len(addedRoles) > 0 {
			// update members, add new roles to given group members
			if err := srv.AddRolesToMembers(ctx, printer, res.Members, addedRoles); err != nil {
				return nil, err
			}
		}
		if len(removedRoles) > 0 {
			// update members, remove roles from existing group members
			if err := srv.RemoveRolesFromMembers(ctx, printer, group.GetId(), existingMembers, removedRoles); err != nil {
				return nil, err
			}
		}
	}
	// handle changes to members
	if _, ok := mask.Get("Members"); ok {
		addedMembers, removedMembers := utils.StringSlicesDiff(existingMembers, group.Members)
		if len(addedMembers) > 0 {
			// update members, add new roles to given group members
			if err := srv.AddRolesToMembers(ctx, printer, addedMembers, res.Roles); err != nil {
				return nil, err
			}
		}
		if len(removedMembers) > 0 {
			// update members, remove existing roles from removed members
			if err := srv.RemoveRolesFromMembers(ctx, printer, group.GetId(), removedMembers, existingRoles); err != nil {
				return nil, err
			}
		}
	}
	return updated.ToPb(), nil
}

// DeleteGroup deletes the group with the given id from the store.
func (srv *Server) DeleteGroup(ctx context.Context, req *gooserv1.IdRequest) (*empty.Empty, error) {
	u, err := srv.GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	printer := message.NewPrinter(language.Make(u.Language))
	if !u.HasRole("admin") {
		return nil, status.Errorf(codes.PermissionDenied, printer.Sprintf("not allowed to delete groups"))
	}
	id := req.GetId()
	// get existing group
	group, err := srv.store.GetGroup(ctx, printer, id)
	if err != nil {
		return nil, err
	}
	// remove all the group's roles from all its members if necessary
	srv.RemoveRolesFromMembers(ctx, printer, id, group.Members, group.Roles)
	// delete group
	err = srv.store.DeleteGroup(ctx, printer, req.GetId())
	return &empty.Empty{}, err
}
