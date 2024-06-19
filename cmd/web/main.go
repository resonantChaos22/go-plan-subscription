package main

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/resonantchaos22/go-concur-final/data"
)

const (
	webPort = "3000"
)

func main() {
	//*	connect to db
	db := initDB()
	db.Ping()

	//*	create sessions
	session := initSession()

	//*	create loggers
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	//!	create channels

	//*	create waitgroup
	wg := sync.WaitGroup{}

	//*	set up application config
	app := Config{
		Session:       session,
		DB:            db,
		WaitGroup:     &wg,
		InfoLog:       infoLog,
		ErrorLog:      errorLog,
		Models:        data.New(db),
		ErrorChan:     make(chan error),
		ErrorChanDone: make(chan bool),
	}

	//*	set up mail
	app.Mailer = app.createMail()
	go app.listenForMail()

	//*	listen for signals
	go app.listenForShutdown()

	//*	listen for errors
	go app.listenForErrors()

	//*	listen for web connections
	app.serve()
}

func (app *Config) listenForErrors() {
	for {
		select {
		case err := <-app.ErrorChan:
			app.ErrorLog.Println(err)
		case <-app.ErrorChanDone:
			return
		}
	}
}

func (app *Config) serve() {
	//	start http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: app.routes(),
	}

	app.InfoLog.Println("Starting web server...")
	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func initDB() *sql.DB {
	conn := connectToDB()
	if conn == nil {
		log.Panic("Cant connect to database")
	}

	return conn
}

func connectToDB() *sql.DB {
	counts := 0

	dsn := os.Getenv("DSN")

	for {
		connection, err := openDB(dsn)
		if err != nil {
			log.Println("Postgres not yet ready...")
			counts++
			if counts > 10 {
				log.Println(err)
				return nil
			}
			time.Sleep(2 * time.Second)
			continue
		}
		log.Println("Connected to database")
		return connection
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func initSession() *scs.SessionManager {
	gob.Register(data.User{})

	//*	set up session
	session := scs.New()
	session.Store = redisstore.New(initRedis())
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = true

	return session
}

func initRedis() *redis.Pool {
	return &redis.Pool{
		MaxIdle: 10,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", os.Getenv("REDIS"))
		},
	}
}

func (app *Config) createMail() Mail {
	//	create channels
	errorChan := make(chan error)
	mailerChan := make(chan Message, 100) //	blocks after 100 mails
	mailerDoneChan := make(chan bool)

	m := Mail{
		Domain:      "localhost",
		Host:        "localhost",
		Port:        1025,
		Encryption:  "none",
		FromAddress: "info@mycompany.com",
		FromName:    "Info",
		ErrorChan:   errorChan,
		MailerChan:  mailerChan,
		DoneChan:    mailerDoneChan,
		WaitGroup:   app.WaitGroup,
	}

	return m
}

func (app *Config) listenForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	app.shutdown()
	os.Exit(0)
}

func (app *Config) shutdown() {
	app.InfoLog.Println("would run cleanup tasks...")

	app.WaitGroup.Wait()
	app.Mailer.DoneChan <- true
	app.ErrorChanDone <- true

	app.InfoLog.Println("closing channels and shutting down applications")
	close(app.Mailer.MailerChan)
	close(app.Mailer.ErrorChan)
	close(app.Mailer.DoneChan)
	close(app.ErrorChan)
	close(app.ErrorChanDone)
}
