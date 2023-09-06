### Usage:

You have to use the type alias `Decimal` declared in this package instead of `postgres.Decimal` from `ericlagergren/decimal` in your structs. Both will work, but the latter will be much slower because it will use the default `database/sql` interface.

```golang
import (
	"context"
	"fmt"
    "log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
    pgxdecimal "github.com/SoMuchForSubtlety/pgx-ericlagergren-decimal"
)

func main() {
	pgCfg, err := pgxpool.ParseConfig("postgres://user:paass@localhost:5432")
	if err != nil {
		log.Fatal(err)
	}
	pgCfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		pgxdecimal.Register(c.TypeMap())
		return nil
	}
    pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		log.Fatal(err)
	}
}

```

This package borrows heavily from the implementation of https://github.com/jackc/pgx-shopspring-decimal
