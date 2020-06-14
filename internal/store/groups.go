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
// It returns the documents, the total size of documents for the given filter and a grpc status type error if anything goes wrong.
func (m *MGO) ListGroups(ctx context.Context, printer *message.Printer, filterString, orderBy, token string, size int32) (groups *[]Group, totalSize int32, nextToken string, err error) {
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
	var g Group
	for cur.Next(ctx) {
		err = cur.Decode(&g)
		if err != nil {
			return nil, 0, "", status.Errorf(codes.Internal, printer.Sprintf("unable to decode group: %s", err))
		}
		*groups = append(*groups, g)
	}
	// if there might be more results
	l := int32(len(*groups))
	if size == l && totalSize > l {
		nextToken, err = m.NextPageToken(
			ctx,
			printer,
			m.usersCollection,
			filterString,
			orderBy,
			g,
		)
		if err != nil {
			return nil, 0, "", err
		}
	}
	return groups, total, nextToken, nil
}

// CountGroups returns the number of user documents corresponding to the given filter.
func (m *MGO) CountGroups(ctx context.Context, printer *message.Printer, filterString string) (int32, error) {
	filter, err := m.bsonDocFromRsqlString(printer, filterString)
	if err != nil {
		return 0, err
	}
	count, err := m.groupsCollection.CountDocuments(ctx, filter, nil)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}
		m.errorLogger.Printf("unable to count groups: %s", err)
		return 0, status.Errorf(codes.Internal, printer.Sprintf("unable to count groups"))
	}
	return int32(count), nil
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
