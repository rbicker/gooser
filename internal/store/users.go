package store

import (
	"context"
	"time"

	"golang.org/x/text/message"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListUsers lists users from the mongo db.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) ListUsers(ctx context.Context, printer *message.Printer, filter string, size, skip int32) (*[]User, int32, error) {
	d, err := m.BsonDocFromRsqlString(filter)
	if err != nil {
		return nil, 0, err
	}
	countOptions := options.Count()
	if size > 0 {
		countOptions.SetLimit(int64(size))
		countOptions.SetSkip(int64(skip))
	}
	if ctx.Err() == context.Canceled {
		return nil, 0, status.Errorf(codes.Canceled, printer.Sprintf("the request was canceled by the client"))
	}
	count, err := m.usersCollection.CountDocuments(ctx, d, countOptions)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, printer.Sprintf("unable to count users: %s", err))
	}
	findOptions := options.Find()
	findOptions.SetLimit(int64(size))
	findOptions.SetSkip(int64(skip))
	findOptions.SetSort(bson.D{{"username", 1}})
	cur, err := m.usersCollection.Find(ctx, d, findOptions)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, printer.Sprintf("unable to query users: %s", err))
	}
	defer cur.Close(ctx)
	var users []User
	for cur.Next(ctx) {
		var u User
		err = cur.Decode(&u)
		if err != nil {
			return nil, 0, status.Errorf(codes.Internal, printer.Sprintf("unable to decode user: %s", err))
		}
		users = append(users, u)
	}
	return &users, int32(count), nil
}

// GetUser gets the user with the given id from the mongo db.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) GetUser(ctx context.Context, printer *message.Printer, id string) (*User, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid id '%s'", id))
	}
	filter := bson.M{"_id": oid}
	u := &User{}
	if ctx.Err() == context.Canceled {
		return nil, status.Errorf(codes.Canceled, printer.Sprintf("the request was canceled by the client"))
	}
	if err := m.usersCollection.FindOne(ctx, filter).Decode(u); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Errorf(codes.NotFound, printer.Sprintf("unable to find user with id %s", id))
		}
		return nil, err
	}
	return u, nil
}

// getUser gets one user based on the given filter.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) getUser(ctx context.Context, printer *message.Printer, filter bson.M) (*User, error) {
	u := &User{}
	if ctx.Err() == context.Canceled {
		return nil, status.Errorf(codes.Canceled, printer.Sprintf("the request was canceled by the client"))
	}
	if err := m.usersCollection.FindOne(ctx, filter).Decode(u); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Errorf(codes.NotFound, printer.Sprintf("unable to find user"))
		}
		return nil, err
	}
	return u, nil
}

// GetUserByUsername gets the user with the given username from the mongo db.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) GetUserByUsername(ctx context.Context, printer *message.Printer, username string) (*User, error) {
	filter := bson.M{"username": username}
	return m.getUser(ctx, printer, filter)
}

// GetUserByMail gets the user with the given mail.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) GetUserByMail(ctx context.Context, printer *message.Printer, mail string) (*User, error) {
	filter := bson.M{"mail": mail}
	return m.getUser(ctx, printer, filter)
}

// GetUserByConfirmToken gets the user with the confirmation token.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) GetUserByConfirmToken(ctx context.Context, printer *message.Printer, token string) (*User, error) {
	filter := bson.M{"confirmToken": token}
	return m.getUser(ctx, printer, filter)
}

// GetUserByPasswordResetToken gets the user with the given token.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) GetUserByPasswordResetToken(ctx context.Context, printer *message.Printer, token string) (*User, error) {
	filter := bson.M{"passwordResetToken": token}
	return m.getUser(ctx, printer, filter)
}

// SaveUser stores the given user in the database.
// The users id will be used to determine if a new user has to be created
// or an existing one can be updated.
func (m *MGO) SaveUser(ctx context.Context, printer *message.Printer, user *User) (*User, error) {
	var err error
	var oid primitive.ObjectID
	user.UpdatedAt = time.Now()
	if user.Id != "" {
		oid, err = primitive.ObjectIDFromHex(user.Id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid user id '%s'", user.Id))
		}
		user.Id = ""
	} else {
		oid = primitive.NewObjectID()
		user.CreatedAt = user.UpdatedAt
	}
	opts := options.FindOneAndUpdate()
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.After)
	filter := bson.M{"_id": oid}
	doc := bson.M{"$set": user}
	u := &User{}
	err = m.usersCollection.FindOneAndUpdate(ctx, filter, doc, opts).Decode(u)
	if err != nil {
		m.errorLogger.Printf("error while saving user: %s", err)
		return nil, status.Errorf(codes.Internal, printer.Sprintf("error while saving user"))
	}
	return u, nil
}

// DeleteUser deletes the user with the given id in mongo db.
func (m *MGO) DeleteUser(ctx context.Context, printer *message.Printer, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid user id"))
	}
	filter := bson.M{"_id": oid}
	if ctx.Err() == context.Canceled {
		return status.Errorf(codes.Canceled, printer.Sprintf("the request was canceled by the client"))
	}
	res, err := m.usersCollection.DeleteOne(ctx, filter)
	if err != nil {
		return status.Errorf(codes.Internal, "unable to delete user")
	}
	if res.DeletedCount != 1 {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("unable to find user with given id"))
	}
	return nil
}
