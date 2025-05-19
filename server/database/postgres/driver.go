package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	d "github.com/m4tth3/loggui/server/database"
)

type driver struct {
	conn *pgx.Conn
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
