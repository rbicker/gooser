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
// It returns the documents, the total size of documents for the given filter and a grpc status type error if anything goes wrong.
func (m *MGO) ListUsers(ctx context.Context, printer *message.Printer, filterString, orderBy, token string, size int32) (users *[]User, totalSize int32, nextToken string, err error) {
	cur, total, err := m.queryDocuments(
		ctx,
		printer,
		m.usersCollection,
		filterString,
		orderBy,
		token,
		size,
	)
	if err != nil {
		return nil, 0, "", err
	}
	defer cur.Close(ctx)
	var u User

	for cur.Next(ctx) {
		err = cur.Decode(&u)
		if err != nil {
			return nil, 0, "", status.Errorf(codes.Internal, printer.Sprintf("unable to decode user: %s", err))
		}
		*users = append(*users, u)
	}
	// if there might be more results
	l := int32(len(*users))
	if size == l && totalSize > l {
		nextToken, err = m.NextPageToken(
			ctx,
			printer,
			m.usersCollection,
			filterString,
			orderBy,
			u,
		)
		if err != nil {
			return nil, 0, "", err
		}
	}
	return users, total, nextToken, err
}

// CountUsers returns the number of user documents corresponding to the given filter.
func (m *MGO) CountUsers(ctx context.Context, printer *message.Printer, filterString string) (int32, error) {
	filter, err := m.bsonDocFromRsqlString(printer, filterString)
	if err != nil {
		return 0, err
	}
	count, err := m.usersCollection.CountDocuments(ctx, filter, nil)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		m.errorLogger.Printf("unable to count users: %s", err)
		return 0, status.Errorf(codes.Internal, printer.Sprintf("unable to count users"))
	}
	return int32(count), nil
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
