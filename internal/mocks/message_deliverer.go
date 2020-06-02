// Code generated by mockery v1.1.2. DO NOT EDIT.

package mocks

import (
	store "github.com/rbicker/gooser/internal/store"
	mock "github.com/stretchr/testify/mock"
)

// MessageDeliverer is an autogenerated mock type for the MessageDeliverer type
type MessageDeliverer struct {
	mock.Mock
}

// SendConfirmToken provides a mock function with given fields: user
func (_m *MessageDeliverer) SendConfirmToken(user *store.User) error {
	ret := _m.Called(user)

	var r0 error
	if rf, ok := ret.Get(0).(func(*store.User) error); ok {
		r0 = rf(user)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendPasswordResetToken provides a mock function with given fields: user
func (_m *MessageDeliverer) SendPasswordResetToken(user *store.User) error {
	ret := _m.Called(user)

	var r0 error
	if rf, ok := ret.Get(0).(func(*store.User) error); ok {
		r0 = rf(user)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}