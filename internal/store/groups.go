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

// ListGroups lists groups from the mongo db.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) ListGroups(ctx context.Context, printer *message.Printer, filter string, size, skip int32) (*[]Group, int32, error) {
	d, err := m.BsonDocFromRsqlString(filter)
	if err != nil {
		return nil, 0, err
	}
	countOptions := options.Count()
	countOptions.SetLimit(int64(size))
	countOptions.SetSkip(int64(skip))
	if ctx.Err() == context.Canceled {
		return nil, 0, status.Errorf(codes.Canceled, printer.Sprintf("the request was canceled by the client"))
	}
	count, err := m.groupsCollection.CountDocuments(ctx, d, countOptions)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, printer.Sprintf("unable to count groups: %s", err))
	}
	findOptions := options.Find()
	findOptions.SetLimit(int64(size))
	findOptions.SetSkip(int64(skip))
	findOptions.SetSort(bson.D{{"name", 1}})
	cur, err := m.groupsCollection.Find(ctx, d, findOptions)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, printer.Sprintf("unable to query groups: %s", err))
	}
	defer cur.Close(ctx)
	var groups []Group
	for cur.Next(ctx) {
		var g Group
		err = cur.Decode(&g)
		if err != nil {
			return nil, 0, status.Errorf(codes.Internal, printer.Sprintf("unable to decode group: %s", err))
		}
		groups = append(groups, g)
	}
	return &groups, int32(count), nil
}

// GetGroup gets the group with the given id from the mongo db.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) GetGroup(ctx context.Context, printer *message.Printer, id string) (*Group, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid id '%s'", id))
	}
	filter := bson.M{"_id": oid}
	g := &Group{}
	if ctx.Err() == context.Canceled {
		return nil, status.Errorf(codes.Canceled, printer.Sprintf("the request was canceled by the client"))
	}
	if err := m.groupsCollection.FindOne(ctx, filter).Decode(g); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Errorf(codes.NotFound, printer.Sprintf("unable to find group with id %s", id))
		}
		return nil, err
	}
	return g, nil
}

// GetGroupByName gets the group with the given name from the mongo db.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) GetGroupByName(ctx context.Context, printer *message.Printer, name string) (*Group, error) {
	filter := bson.M{"name": name}
	g := &Group{}
	if ctx.Err() == context.Canceled {
		return nil, status.Errorf(codes.Canceled, printer.Sprintf("the request was canceled by the client"))
	}
	if err := m.groupsCollection.FindOne(ctx, filter).Decode(g); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Errorf(codes.NotFound, printer.Sprintf("unable to find group named %s", name))
		}
		return nil, err
	}
	return g, nil
}

// SaveGroup stores the given group in the database.
// The group id will be used to determine if a new group has to be created
// or an existing one can be updated.
func (m *MGO) SaveGroup(ctx context.Context, printer *message.Printer, group *Group) (*Group, error) {
	var err error
	var oid primitive.ObjectID
	group.UpdatedAt = time.Now()
	if group.Id != "" {
		oid, err = primitive.ObjectIDFromHex(group.Id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid group id '%s'", group.Id))
		}
		group.Id = ""
	} else {
		oid = primitive.NewObjectID()
		group.CreatedAt = group.UpdatedAt
	}
	opts := options.FindOneAndUpdate()
	opts.SetUpsert(true)
	opts.SetReturnDocument(options.After)
	filter := bson.M{"_id": oid}
	doc := bson.M{"$set": group}
	g := &Group{}
	err = m.groupsCollection.FindOneAndUpdate(ctx, filter, doc, opts).Decode(g)
	if err != nil {
		m.errorLogger.Printf("error while saving user: %s", err)
		return nil, status.Errorf(codes.Internal, printer.Sprintf("error while saving group"))
	}
	return g, nil
}

// DeleteGroup deletes the group with the given id.
func (m *MGO) DeleteGroup(ctx context.Context, printer *message.Printer, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid group id")
	}
	filter := bson.M{"_id": oid}
	if ctx.Err() == context.Canceled {
		return status.Errorf(codes.Canceled, printer.Sprintf("the request was canceled by the client"))
	}
	res, err := m.usersCollection.DeleteOne(ctx, filter)
	if err != nil {
		return status.Errorf(codes.Internal, "unable to delete group")
	}
	if res.DeletedCount != 1 {
		return status.Errorf(codes.InvalidArgument, printer.Sprintf("unable to find group with id '%s'", id))
	}
	return nil
}
