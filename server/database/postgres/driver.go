package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/m4tth3/loggui/core"
	d "github.com/m4tth3/loggui/server/database"
)

type driver struct {
	conn *pgx.Conn
}

func (d driver) Init() error {
	// Initialize the connection or perform any setup needed.
	return nil
}

func (d driver) GetLogs(filter *d.Filter) (chan *core.Log, error) {
	return make(chan *core.Log), nil
}

func (d driver) WriteLog(log *core.Log) error {
	return nil
}

func NewQueryHandler(url string) (d.QueryHandler, error) {
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}

	return driver{
		conn: conn,
	}, nil
}
