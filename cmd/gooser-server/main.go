package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rbicker/gooser/internal/mailer"

	"github.com/rbicker/gooser/internal/auth"
	"github.com/rbicker/gooser/internal/server"
	"github.com/rbicker/gooser/internal/store"
	"github.com/rbicker/gooser/internal/utils"
)

func main() {
	infoLog := log.New(os.Stdout, "INFO: ", log.Lmsgprefix+log.LstdFlags)
	errLog := log.New(os.Stderr, "ERROR: ", log.Lmsgprefix+log.LstdFlags)
	var dbOpts []func(*store.MGO) error
	var srvOpts []func(*server.Server) error
	// init db connection
	mongoUrl := utils.LookupEnv("GOOSER_MONGO_URL", "mongodb://localhost:27017")
	dbOpts = append(dbOpts, store.SetURL(mongoUrl))
	dbName := utils.LookupEnv("GOOSER_MONGO_DB", "db")
	dbOpts = append(dbOpts, store.SetDBName(dbName))
	usersColName := utils.LookupEnv("GOOSER_MONGO_USERS_COLLECTION", "users")
	dbOpts = append(dbOpts, store.SetUsersCollectionName(usersColName))
	groupsColName := utils.LookupEnv("GOOSER_MONGO_GROUPS_COLLECTION", "groups")
	dbOpts = append(dbOpts, store.SetGroupsCollectionName(groupsColName))
	db, err := store.NewMongoConnection(dbOpts...)
	if err != nil {
		errLog.Fatalf("unable to create mongodb connection: %s", err)
	}
	err = db.Connect()
	if err != nil {
		errLog.Fatalf("unable to connect to mongodb: %s", err)
	}
	infoLog.Println("connected to mongodb")
	// mailer
	var mailClient mailer.MailClient
	smtpHost, ok := os.LookupEnv("GOOSER_SMTP_HOST")
	if ok {
		smtpPort := utils.LookupEnv("GOOSER_SMTP_PORT", "587")
		smtpUsername, _ := os.LookupEnv("GOOSER_SMTP_USERNAME")
		smtpPassword, _ := os.LookupEnv("GOOSER_SMTP_PASSWORD")
		mailClient, err = mailer.NewTLSMailer(smtpHost, smtpPort, smtpUsername, smtpPassword)
		if err != nil {
			log.Fatalf("error while creating mail client: %s", err)
		}
	} else {
		infoLog.Println("no SMTP settings given, sending mails by logging them to stdout")
		infoLog.Println("to send real mails, have a look at the GOOSER_SMTP_* environment variables")
		mailClient = mailer.NewLogMailer(infoLog)
	}
	mailFrom, _ := os.LookupEnv("GOOSER_MAIL_FROM")
	siteName := utils.LookupEnv("GOOSER_SITE_NAME", "gooser")
	confirmUrl := utils.LookupEnv("GOOSER_CONFIRM_URL", "http://localhost:1234/#/confirm-mail")
	resetPasswordUrl := utils.LookupEnv("GOOSER_RESET_PASSWORD_URL", "http://localhost:1234/#/reset-password")
	mailer, err := mailer.NewMailer(mailClient, mailFrom, siteName, confirmUrl, resetPasswordUrl)
	if err != nil {
		log.Fatalf("error while creating mailer: %s", err)
	}
	// init server
	srvOpts = append(srvOpts, server.EnableReflection())
	p := utils.LookupEnv("GOOSER_PORT", "50051")
	srvOpts = append(srvOpts, server.SetPort(p))
	oauthUrl := utils.LookupEnv("GOOSER_OAUTH_URL", "http://localhost:4444")
	oAuth, err := auth.NewOAuthClient(oauthUrl)
	if err != nil {
		errLog.Fatalf("unable to create oAuth client: %s", err)
	}
	secret := "secret"
	if v, ok := os.LookupEnv("GOOSER_SECRET"); ok {
		secret = v
	} else {
		infoLog.Printf("WARNING: using default secret '%s', this is very insecure, please set the GOOSER_SECRET environment variable to a random secret string", secret)
	}
	srv, err := server.NewServer(secret, db, oAuth, mailer, srvOpts...)
	if err != nil {
		errLog.Fatalf("unable to create new gooser server: %s", err)
	}
	// init collections
	err = srv.InitCollections(context.Background())
	if err != nil {
		errLog.Fatalf("unable to initialize collections: %s", err)
	}
	// channels
	errChan := make(chan error)
	stopChan := make(chan os.Signal)
	// bind OS events to the signal channel
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)
	// serve in a go routine
	go func() {
		infoLog.Println("starting gooser server")
		if err := srv.Serve(); err != nil {
			errChan <- err
		}
	}()
	// terminate gracefully before leaving the main function
	defer func() {
		infoLog.Println("stopping grpc server")
		srv.Stop()
		infoLog.Println("disconnecting from mongodb")
		err := db.Disconnect(context.TODO())
		if err != nil {
			errLog.Fatalf("error while disconnecting from mongodb: %s", err)
		}
	}()
	// block until either OS signal, or server fatal error
	select {
	case err := <-errChan:
		errLog.Printf("Fatal error: %v\n", err)
	case <-stopChan:
	}
}
