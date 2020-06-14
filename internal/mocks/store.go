// Code generated by mockery v1.1.2. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	message "golang.org/x/text/message"

	store "github.com/rbicker/gooser/internal/store"
)

// Store is an autogenerated mock type for the Store type
type Store struct {
	mock.Mock
}

// CountGroups provides a mock function with given fields: ctx, printer, filterString
func (_m *Store) CountGroups(ctx context.Context, printer *message.Printer, filterString string) (int32, error) {
	ret := _m.Called(ctx, printer, filterString)

	var r0 int32
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) int32); ok {
		r0 = rf(ctx, printer, filterString)
	} else {
		r0 = ret.Get(0).(int32)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string) error); ok {
		r1 = rf(ctx, printer, filterString)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CountUsers provides a mock function with given fields: ctx, printer, filterString
func (_m *Store) CountUsers(ctx context.Context, printer *message.Printer, filterString string) (int32, error) {
	ret := _m.Called(ctx, printer, filterString)

	var r0 int32
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) int32); ok {
		r0 = rf(ctx, printer, filterString)
	} else {
		r0 = ret.Get(0).(int32)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string) error); ok {
		r1 = rf(ctx, printer, filterString)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteGroup provides a mock function with given fields: ctx, printer, id
func (_m *Store) DeleteGroup(ctx context.Context, printer *message.Printer, id string) error {
	ret := _m.Called(ctx, printer, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) error); ok {
		r0 = rf(ctx, printer, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteUser provides a mock function with given fields: ctx, printer, id
func (_m *Store) DeleteUser(ctx context.Context, printer *message.Printer, id string) error {
	ret := _m.Called(ctx, printer, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) error); ok {
		r0 = rf(ctx, printer, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetGroup provides a mock function with given fields: ctx, printer, id
func (_m *Store) GetGroup(ctx context.Context, printer *message.Printer, id string) (*store.Group, error) {
	ret := _m.Called(ctx, printer, id)

	var r0 *store.Group
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) *store.Group); ok {
		r0 = rf(ctx, printer, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.Group)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string) error); ok {
		r1 = rf(ctx, printer, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetGroupByName provides a mock function with given fields: ctx, printer, name
func (_m *Store) GetGroupByName(ctx context.Context, printer *message.Printer, name string) (*store.Group, error) {
	ret := _m.Called(ctx, printer, name)

	var r0 *store.Group
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) *store.Group); ok {
		r0 = rf(ctx, printer, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.Group)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string) error); ok {
		r1 = rf(ctx, printer, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUser provides a mock function with given fields: ctx, printer, id
func (_m *Store) GetUser(ctx context.Context, printer *message.Printer, id string) (*store.User, error) {
	ret := _m.Called(ctx, printer, id)

	var r0 *store.User
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) *store.User); ok {
		r0 = rf(ctx, printer, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string) error); ok {
		r1 = rf(ctx, printer, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserByConfirmToken provides a mock function with given fields: ctx, printer, token
func (_m *Store) GetUserByConfirmToken(ctx context.Context, printer *message.Printer, token string) (*store.User, error) {
	ret := _m.Called(ctx, printer, token)

	var r0 *store.User
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) *store.User); ok {
		r0 = rf(ctx, printer, token)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string) error); ok {
		r1 = rf(ctx, printer, token)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserByMail provides a mock function with given fields: ctx, printer, mail
func (_m *Store) GetUserByMail(ctx context.Context, printer *message.Printer, mail string) (*store.User, error) {
	ret := _m.Called(ctx, printer, mail)

	var r0 *store.User
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) *store.User); ok {
		r0 = rf(ctx, printer, mail)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string) error); ok {
		r1 = rf(ctx, printer, mail)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserByPasswordResetToken provides a mock function with given fields: ctx, printer, token
func (_m *Store) GetUserByPasswordResetToken(ctx context.Context, printer *message.Printer, token string) (*store.User, error) {
	ret := _m.Called(ctx, printer, token)

	var r0 *store.User
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) *store.User); ok {
		r0 = rf(ctx, printer, token)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string) error); ok {
		r1 = rf(ctx, printer, token)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUserByUsername provides a mock function with given fields: ctx, printer, username
func (_m *Store) GetUserByUsername(ctx context.Context, printer *message.Printer, username string) (*store.User, error) {
	ret := _m.Called(ctx, printer, username)

	var r0 *store.User
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string) *store.User); ok {
		r0 = rf(ctx, printer, username)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string) error); ok {
		r1 = rf(ctx, printer, username)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListGroups provides a mock function with given fields: ctx, printer, filterString, orderBy, token, size
func (_m *Store) ListGroups(ctx context.Context, printer *message.Printer, filterString string, orderBy string, token string, size int32) (*[]store.Group, int32, string, error) {
	ret := _m.Called(ctx, printer, filterString, orderBy, token, size)

	var r0 *[]store.Group
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string, string, string, int32) *[]store.Group); ok {
		r0 = rf(ctx, printer, filterString, orderBy, token, size)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*[]store.Group)
		}
	}

	var r1 int32
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string, string, string, int32) int32); ok {
		r1 = rf(ctx, printer, filterString, orderBy, token, size)
	} else {
		r1 = ret.Get(1).(int32)
	}

	var r2 string
	if rf, ok := ret.Get(2).(func(context.Context, *message.Printer, string, string, string, int32) string); ok {
		r2 = rf(ctx, printer, filterString, orderBy, token, size)
	} else {
		r2 = ret.Get(2).(string)
	}

	var r3 error
	if rf, ok := ret.Get(3).(func(context.Context, *message.Printer, string, string, string, int32) error); ok {
		r3 = rf(ctx, printer, filterString, orderBy, token, size)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// ListUsers provides a mock function with given fields: ctx, printer, filterString, orderBy, token, size
func (_m *Store) ListUsers(ctx context.Context, printer *message.Printer, filterString string, orderBy string, token string, size int32) (*[]store.User, int32, string, error) {
	ret := _m.Called(ctx, printer, filterString, orderBy, token, size)

	var r0 *[]store.User
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, string, string, string, int32) *[]store.User); ok {
		r0 = rf(ctx, printer, filterString, orderBy, token, size)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*[]store.User)
		}
	}

	var r1 int32
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, string, string, string, int32) int32); ok {
		r1 = rf(ctx, printer, filterString, orderBy, token, size)
	} else {
		r1 = ret.Get(1).(int32)
	}

	var r2 string
	if rf, ok := ret.Get(2).(func(context.Context, *message.Printer, string, string, string, int32) string); ok {
		r2 = rf(ctx, printer, filterString, orderBy, token, size)
	} else {
		r2 = ret.Get(2).(string)
	}

	var r3 error
	if rf, ok := ret.Get(3).(func(context.Context, *message.Printer, string, string, string, int32) error); ok {
		r3 = rf(ctx, printer, filterString, orderBy, token, size)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// SaveGroup provides a mock function with given fields: ctx, printer, group
func (_m *Store) SaveGroup(ctx context.Context, printer *message.Printer, group *store.Group) (*store.Group, error) {
	ret := _m.Called(ctx, printer, group)

	var r0 *store.Group
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, *store.Group) *store.Group); ok {
		r0 = rf(ctx, printer, group)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.Group)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, *store.Group) error); ok {
		r1 = rf(ctx, printer, group)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveUser provides a mock function with given fields: ctx, printer, user
func (_m *Store) SaveUser(ctx context.Context, printer *message.Printer, user *store.User) (*store.User, error) {
	ret := _m.Called(ctx, printer, user)

	var r0 *store.User
	if rf, ok := ret.Get(0).(func(context.Context, *message.Printer, *store.User) *store.User); ok {
		r0 = rf(ctx, printer, user)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.User)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *message.Printer, *store.User) error); ok {
		r1 = rf(ctx, printer, user)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
