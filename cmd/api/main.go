package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/RayMC17/bookclub-api/internal/data"
	"github.com/RayMC17/bookclub-api/internal/mailer"
	_ "github.com/lib/pq"
)

const appVersion = "1.0.0"

type serverConfig struct {
	port        int
	environment string
	db          struct {
		dsn string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type applicationDependencies struct {
	config           serverConfig
	logger           *slog.Logger
	bookModel        data.BookModel
	mailer           mailer.Mailer
	wg               sync.WaitGroup
	tokenModel       data.TokenModel
	readingListModel data.ReadingListModel
	reviewModel      data.ReviewModel
	userModel        data.UserModel
}

func main() {
	var settings serverConfig

	flag.IntVar(&settings.port, "port", 4000, "Server port")
	flag.StringVar(&settings.environment, "env", "development", "Environment?(development|staging|production)")
	flag.StringVar(&settings.db.dsn, "db-dsn", "postgres://bookclub:bookclub@localhost/bookclub?sslmode=disable", "PostgreSQL DSN")
	flag.Float64Var(&settings.limiter.rps, "limiter-rps", 2, "Rate Limiter maximum requests per second")
	flag.IntVar(&settings.limiter.burst, "limiter-burst", 5, "Rate Limiter maximum burst")
	flag.BoolVar(&settings.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.StringVar(&settings.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&settings.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&settings.smtp.username, "smtp-username", "3eeb8cf254893a", "SMTP username")
	flag.StringVar(&settings.smtp.password, "smtp-password", "1ddfdab02da850", "SMTP password")
	flag.StringVar(&settings.smtp.sender, "smtp-sender", "Readign Community <no-reply@book-club.rayray.net>", "SMTP sender")

	flag.Parse()

	// Initialize the logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Open database connection
	db, err := openDB(settings)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	// Initialize application dependencies
	appInstance := &applicationDependencies{
		config:           settings,
		logger:           logger,
		bookModel:        data.BookModel{DB: db},
		readingListModel: data.ReadingListModel{DB: db},
		reviewModel:      data.ReviewModel{DB: db},
		mailer:           mailer.New(settings.smtp.host, settings.smtp.port, settings.smtp.username, settings.smtp.password, settings.smtp.sender),
		tokenModel:       data.TokenModel{DB: db},
		userModel:        data.UserModel{DB: db},
	}

	// Start the server
	err = appInstance.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

// openDB sets up the database connection
func openDB(settings serverConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", settings.db.dsn)
	if err != nil {
		return nil, err
	}

	// Context with timeout for pinging the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
