package store

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

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
func NewMongoConnection(opts ...func(*MGO) error) (*MGO, error) {
	// create server with default options
	var m = MGO{
		infoLogger:           log.New(os.Stdout, "INFO: ", log.Lmsgprefix+log.LstdFlags),
		errorLogger:          log.New(os.Stderr, "ERROR: ", log.Lmsgprefix+log.LstdFlags),
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
	if err != nil {
		return fmt.Errorf("unable to use database %s and users collection %s: %w", m.databaseName, m.usersCollectionName, err)
	}
	m.groupsCollection = m.mongoClient.Database(m.databaseName).Collection(m.groupsCollectionName)
	if err != nil {
		return fmt.Errorf("unable to use database %s and groups collection %s: %w", m.databaseName, m.groupsCollectionName, err)
	}
	return nil
}

// Disconnect closes the connection to the mongodb server.
func (m *MGO) Disconnect(ctx context.Context) error {
	return m.mongoClient.Disconnect(ctx)
}

// SetURL changes the url to which the connection should be established.
func SetURL(url string) func(*MGO) error {
	return func(m *MGO) error {
		m.url = url
		return nil
	}
}

// SetDBName changes the name of the mongodb database.
func SetDBName(databaseName string) func(*MGO) error {
	return func(m *MGO) error {
		m.databaseName = databaseName
		return nil
	}
}

// SetUsersCollectionName changes the name of the mongodb users collection.
func SetUsersCollectionName(collectionName string) func(*MGO) error {
	return func(m *MGO) error {
		m.usersCollectionName = collectionName
		return nil
	}
}

// SetUsersCollectionName changes the name of the mongodb groups collection.
func SetGroupsCollectionName(collectionName string) func(*MGO) error {
	return func(m *MGO) error {
		m.groupsCollectionName = collectionName
		return nil
	}
}

// BsonDocFromRsqlString parses the given rsql string turns it
// into a BSON.D document.
// It returns a grpc status type error if anything goes wrong.
func (m *MGO) BsonDocFromRsqlString(filter string) (bson.D, error) {
	jsonFilter, err := m.rsqlParser.Process(filter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rsql filter string '%s': %s", filter, err)
	}
	doc := bson.D{}
	err = bson.UnmarshalExtJSON([]byte(jsonFilter), true, &doc)
	return doc, err
}
