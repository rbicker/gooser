package store

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/message"

	"github.com/rbicker/go-rsql"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MGO implements the store interface using a mongodb.
type MGO struct {
	rsqlParser           *rsql.Parser
	errorLogger          *log.Logger
	infoLogger           *log.Logger
	secret               string
	url                  string
	databaseName         string
	usersCollectionName  string
	groupsCollectionName string
	mongoClient          *mongo.Client
	usersCollection      *mongo.Collection
	groupsCollection     *mongo.Collection
}

// ensure MGO implements the store interface.
var _ Store = &MGO{}

// NewMongoConnection creates a new mongo database connection.
// It takes functional parameters to change default options
// such as the mongo url
// It returns the newly created server or an error if
// something went wrong.
func NewMongoConnection(secret string, opts ...func(*MGO) error) (*MGO, error) {
	// create server with default options
	var m = MGO{
		secret:               secret,
		url:                  "mongodb://localhost:27017",
		databaseName:         "db",
		usersCollectionName:  "users",
		groupsCollectionName: "groups",
	}
	// run functional options
	for _, op := range opts {
		err := op(&m)
		if err != nil {
			return nil, fmt.Errorf("setting option: %w", err)
		}
	}
	// default loggers
	if m.infoLogger == nil {
		m.infoLogger = log.New(os.Stdout, "INFO: ", log.Lmsgprefix+log.LstdFlags)
	}
	if m.errorLogger == nil {
		m.errorLogger = log.New(os.Stdout, "ERROR: ", log.Lmsgprefix+log.LstdFlags)
	}
	// handle secret
	h := md5.New()
	if _, err := io.WriteString(h, m.secret); err != nil {
		return nil, fmt.Errorf("unable to hash secret: %w", err)
	}
	m.secret = fmt.Sprintf("%x", h.Sum(nil))
	// prepare rsql parser
	var parserOpts []func(*rsql.Parser) error
	formatter := func(key, value string, not bool) string {
		var ids, values []string
		re := regexp.MustCompile(`\(||\)`)
		value = re.ReplaceAllString(value, "")
		values = strings.Split(value, ",")
		if len(values) == 1 {
			if not {
				return fmt.Sprintf(`{ "%s": { "$ne": { "$oid": %s } } }`, key, value)
			}
			return fmt.Sprintf(`{ "%s": { "$oid": %s } }`, key, value)
		}
		for _, v := range values {
			ids = append(ids, fmt.Sprintf(`{ "$oid": %s }`, v))
		}
		op := "$in"
		if not {
			op = "$nin"
		}
		return fmt.Sprintf(`{ "%s": { "%s": [%s] } }`, key, op, strings.Join(ids, ", "))
	}
	customOperators := []rsql.Operator{
		{
			// turn "_id=oid=xxx" into { "_id": { "$oid": "xxx" } } or
			// turn "userIds=oid=(xxx,yyy) into { "userIds": { "$in": [{ "$oid": "xxx" }, { "$oid": "yyy" } }] } }
			Operator: "=oid=",
			Formatter: func(key, value string) string {
				return formatter(key, value, false)
			},
		},
		{
			Operator: "!oid=",
			Formatter: func(key, value string) string {
				return formatter(key, value, true)
			},
		},
	}
	parserOpts = append(parserOpts, rsql.Mongo())
	parserOpts = append(parserOpts, rsql.WithOperators(customOperators...))
	parser, err := rsql.NewParser(parserOpts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create rsql parser: %s", err)
	}
	m.rsqlParser = parser
	return &m, nil
}

// Connect establishes a connection to a mongodb server.
func (m *MGO) Connect() error {
	var err error
	m.mongoClient, err = mongo.NewClient(options.Client().ApplyURI(m.url))
	if err != nil {
		return fmt.Errorf("unable to create mongo client: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = m.mongoClient.Connect(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect: %w", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = m.mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		return fmt.Errorf("unable to ping: %w", err)
		return err
	}
	m.usersCollection = m.mongoClient.Database(m.databaseName).Collection(m.usersCollectionName)
	m.groupsCollection = m.mongoClient.Database(m.databaseName).Collection(m.groupsCollectionName)
	return nil
}

// Disconnect closes the connection to the mongodb server.
func (m *MGO) Disconnect(ctx context.Context) error {
	return m.mongoClient.Disconnect(ctx)
}

// WithURL changes the url to which the connection should be established.
func WithURL(url string) func(*MGO) error {
	return func(m *MGO) error {
		m.url = url
		return nil
	}
}

// WithDBName changes the name of the mongodb database.
func WithDBName(databaseName string) func(*MGO) error {
	return func(m *MGO) error {
		m.databaseName = databaseName
		return nil
	}
}

// WithUsersCollectionName changes the name of the mongodb users collection.
func WithUsersCollectionName(collectionName string) func(*MGO) error {
	return func(m *MGO) error {
		m.usersCollectionName = collectionName
		return nil
	}
}

// WithGroupsCollectionName changes the name of the mongodb groups collection.
func WithGroupsCollectionName(collectionName string) func(*MGO) error {
	return func(m *MGO) error {
		m.groupsCollectionName = collectionName
		return nil
	}
}

// paginatedFilterBuilder builds a filter which considers not only filter and orderBy which might
// have been given by the user but also the pagination based on the given object.
func (m *MGO) paginatedFilterBuilder(printer *message.Printer, filter bson.D, orderBy string, obj interface{}) (bson.D, error) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Struct {
		m.errorLogger.Printf("unexpected type of given object in filterBuilder(), expected Struct, got %s", v.Kind().String())
		return nil, status.Errorf(codes.Internal, printer.Sprintf("internal error while building filter"))
	}
	id := v.FieldByName("Id")
	if !id.IsValid() {
		m.errorLogger.Printf("given object does not have an 'Id' - %+v", obj)
		return nil, status.Errorf(codes.Internal, printer.Sprintf("internal error while building filter"))
	}
	idFilter := bson.E{
		Key: "_id",
		Value: bson.E{
			Key: "$oid",
			Value: bson.E{
				Key:   "$gt",
				Value: id.Interface(),
			},
		},
	}
	// build pagination filter which makes sure that the
	// next page starts with the document coming after the given one
	// (depending on how the documents are / will be sorted)
	var pageFilter bson.D
	var next, exact bson.D
	if orderBy != "" {
		// if sorting is requested
		sorts := strings.Split(orderBy, ",")
		for _, name := range sorts {
			name = strings.TrimSpace(name)
			if len(name) == 0 {
				return nil, status.Errorf(codes.InvalidArgument, printer.Sprintf("orderBy field has a length of 0"))
			}
			// ensure snake case
			name = strings.ToLower(name[:1]) + name[1:]
			// operator
			op := "$gt"
			if name[0:1] == "-" {
				op = "$lt"
				name = name[1:]

			}
			if name[0:1] == "+" {
				name = name[1:]
			}
			// handle id field which is special
			if name == "id" {
				// like id filter but considering
				// the operator (which might be $lt)
				flt := bson.E{
					Key: "_id",
					Value: bson.E{
						Key: "$oid",
						Value: bson.E{
							Key:   op,
							Value: id.Interface(),
						},
					},
				}
				// if the only orderBy string is id
				if len(sorts) == 1 {
					pageFilter = bson.D{flt}
					// no need to do more
					// as id is the only field
					break
				}
				// for id, we know that it is unique
				// therefore there cannot be exact
				// matches so we only set next and continue
				next = append(next, flt)
				continue
			}
			// get field value by it's name (camel case)
			f := v.FieldByName(strings.Title(name))
			if !f.IsValid() {
				// if field does not exist or has zero value
				// it cannot be used as a filter
				continue
			}
			value := f.Interface()
			// A) either the next document needs to have a
			// greater or smaller value for the given field
			// (depending on sort direction)
			next = append(next, bson.E{
				Key: name,
				Value: bson.E{
					Key:   op,
					Value: value,
				},
			})
			// B) or it needs to be an exact match
			// but the id needs to be greater, a criteria
			// we add after the loop
			exact = append(exact, bson.E{
				Key:   name,
				Value: value,
			})
		}
		// if only id was given as orderBy string
		// exact has a length of 0
		if len(exact) > 0 {
			// ensure the id is greater for exact matches
			exact = append(exact, idFilter)
			// put together page filter
			pageFilter = bson.D{
				{
					Key: "$or",
					Value: bson.A{
						next,
						exact,
					},
				},
			}
		}
	}
	// if no (valid) sorting is given, it will be sorted by
	// id, which makes the pagination filter quite simple
	if len(pageFilter) == 0 {
		pageFilter = bson.D{idFilter}
	}
	if len(filter) > 0 {
		// merge given and pagination filters
		filter = bson.D{
			{
				Key: "$and",
				Value: bson.A{
					filter,
					pageFilter,
				},
			},
		}
	} else {
		// no filter given as input
		// resulting filter equals the pagination filter
		filter = pageFilter
	}
	return filter, nil
}

// NextPageToken generates the next page token and returns it as an encrypted string.
func (m *MGO) NextPageToken(ctx context.Context, printer *message.Printer, collection *mongo.Collection, filterString, orderBy string, document interface{}) (string, error) {
	// get filter bson
	filter, err := m.bsonDocFromRsqlString(printer, filterString)
	if err != nil {
		return "", err
	}
	nextFilter, err := m.paginatedFilterBuilder(
		printer,
		filter,
		orderBy,
		document,
	)
	if err != nil {
		return "", err
	}
	var nextDoc bson.M
	err = collection.FindOne(ctx, nextFilter).Decode(&nextDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// no more documents
			return "", nil
		}
		m.errorLogger.Printf("error while creating pagination token, while searching for next document: %s", err)
		return "", status.Errorf(codes.Internal, printer.Sprintf("unable to search next document while creating pagination token"))
	}
	next := &PageToken{
		FilterString:     filterString,
		OrderBy:          orderBy,
		PaginationFilter: nextFilter,
	}
	res, err := next.EncryptedString(m.secret)
	if err != nil {
		m.errorLogger.Printf("unable to encrypt page token: %s", err)
	}
	return res, nil
}

// bsonDocFromRsqlString parses the given rsql string turns it
// into a BSON.D document.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) bsonDocFromRsqlString(p *message.Printer, filter string) (bson.D, error) {
	jsonFilter, err := m.rsqlParser.Process(filter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, p.Sprintf("invalid rsql filter string '%s': %s", filter, err))
	}
	doc := bson.D{}
	err = bson.UnmarshalExtJSON([]byte(jsonFilter), true, &doc)
	return doc, err
}

// bsonDocFromOrderByString creates a bson.D document which can be used as a sort option
// for a mongodb query.
// It returns a grpc status type error if anything goes wrong.
func bsonDocFromOrderByString(p *message.Printer, sort string) (bson.D, error) {
	if sort == "" {
		return bson.D{}, nil
	}
	var res bson.D
	for _, s := range strings.Split(sort, ",") {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, p.Sprintf("%s has a length of 0", s))
		}
		i := 1
		if s[0:1] == "-" {
			i = -1
			s = s[1:]
		}
		if s[0:1] == "+" {
			s = s[1:]
		}
		res = append(res, bson.E{Key: s, Value: i})
	}
	return res, nil
}

// queryDocuments queries the document from the given collection and returns a mongo cursor.
// The function considers the given filter & order by. The query will be corresponding to the given pagination token.
// It returns the total size for the query (not considering given skip or limit parameters),
// the mongo cursor, and a protobuf type error if anything goes wrong.
func (m *MGO) queryDocuments(ctx context.Context, printer *message.Printer, collection *mongo.Collection, filterString, orderBy, token string, size int32) (cur *mongo.Cursor, totalSize int32, err error) {
	// decrypt page token
	pageToken, err := PageTokenFromEncryptedString(m.secret, token)
	if err != nil {
		return nil, 0, status.Errorf(codes.InvalidArgument, printer.Sprintf("invalid page token given"))
	}
	// if page token is given, check if filter & orderBy match
	if pageToken != nil {
		if filterString != pageToken.FilterString {
			return nil, 0, status.Errorf(codes.InvalidArgument, printer.Sprintf("pagination filter and given filters do not match"))
		}
		if orderBy != pageToken.OrderBy {
			return nil, 0, status.Errorf(codes.InvalidArgument, printer.Sprintf("pagination orderBy and given orderBy do not match"))
		}
	}
	// get filter bson
	filter, err := m.bsonDocFromRsqlString(printer, filterString)
	if err != nil {
		return nil, 0, err
	}
	// get orderBy bson
	sortOption, err := bsonDocFromOrderByString(printer, orderBy)
	if err != nil {
		return nil, 0, err
	}
	// before running db queries, check if request was canceled
	if ctx.Err() == context.Canceled {
		return nil, 0, status.Errorf(codes.Canceled, printer.Sprintf("the request was canceled by the client"))
	}
	// count total size of documents
	count, err := collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		m.errorLogger.Printf("unable to count %s: %s", collection.Name(), err)
		return nil, 0, status.Errorf(codes.Internal, printer.Sprintf("unable to count %s", collection.Name()))
	}
	// if page token is not nil,
	// use the pagination filter from now on
	if pageToken != nil {
		filter = pageToken.PaginationFilter
	}
	totalSize = int32(count)
	findOptions := options.Find()
	if size > 0 {
		findOptions.SetLimit(int64(size))
	}
	findOptions.SetSort(sortOption)
	// query the documents, use the pagination filter
	// to get results for the current page
	cur, err = collection.Find(ctx, filter, findOptions)
	if err != nil {
		m.errorLogger.Printf("unable to query %s: %s", collection.Name(), err)
		return nil, 0, status.Errorf(codes.Internal, printer.Sprintf("error while querying %s", collection.Name()))
	}
	return cur, totalSize, nil
}
