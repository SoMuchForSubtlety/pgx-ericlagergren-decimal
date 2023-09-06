package decimal_test

import (
	"context"
	"math"
	"testing"

	"github.com/ericlagergren/decimal/sql/postgres"

	pgxdecimal "github.com/SoMuchForSubtlety/pgx-ericlagergren-decimal"
	"github.com/ericlagergren/decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxtest"
	"github.com/stretchr/testify/require"
)

var defaultConnTestRunner pgxtest.ConnTestRunner

func init() {
	defaultConnTestRunner = pgxtest.DefaultConnTestRunner()
	defaultConnTestRunner.CreateConfig = func(ctx context.Context, t testing.TB) *pgx.ConnConfig {
		cfg, err := pgx.ParseConfig("postgres://postgres:testpostgres@localhost:5434/postgres?sslmode=disable")
		require.NoError(t, err)
		return cfg
	}
	defaultConnTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		pgxdecimal.Register(conn.TypeMap())
	}
}

func TestCodecDecodeValue(t *testing.T) {
	t.Parallel()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		d := &decimal.Big{}
		d.SetString("1.234")
		original := pgxdecimal.Decimal{V: d}

		rows, err := conn.Query(context.Background(), `select $1::numeric`, original)
		require.NoError(t, err)

		for rows.Next() {
			values, err := rows.Values()
			require.NoError(t, err)

			require.Len(t, values, 1)
			v0, ok := values[0].(postgres.Decimal)
			require.True(t, ok)
			require.True(t, original.V.Cmp(v0.V) == 0)
		}

		require.NoError(t, rows.Err())

		rows, err = conn.Query(context.Background(), `select $1::numeric`, nil)
		require.NoError(t, err)

		for rows.Next() {
			values, err := rows.Values()
			require.NoError(t, err)

			require.Len(t, values, 1)
			require.Equal(t, nil, values[0])
		}

		require.NoError(t, rows.Err())
	})
}

func TestNaN(t *testing.T) {
	t.Parallel()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		var d pgxdecimal.Decimal
		err := conn.QueryRow(context.Background(), `select 'NaN'::numeric`).Scan(&d)
		require.EqualError(t, err, `can't scan into dest[0]: cannot scan NaN into *postgres.Decimal`)
	})
}

func TestArray(t *testing.T) {
	t.Parallel()
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		inputSlice := []pgxdecimal.Decimal{}

		for i := 0; i < 10; i++ {
			d := decimal.New(int64(i), 0)
			inputSlice = append(inputSlice, pgxdecimal.Decimal{V: d})
		}

		var outputSlice []pgxdecimal.Decimal
		err := conn.QueryRow(context.Background(), `select $1::numeric[]`, inputSlice).Scan(&outputSlice)
		require.NoError(t, err)

		require.Equal(t, len(inputSlice), len(outputSlice))
		for i := 0; i < len(inputSlice); i++ {
			require.True(t, outputSlice[i].V.Cmp(inputSlice[i].V) == 0)
		}
	})
}

func isExpectedEqDecimal(a pgxdecimal.Decimal) func(interface{}) bool {
	return func(v interface{}) bool {
		return a.V.Cmp(v.(pgxdecimal.Decimal).V) == 0
	}
}

func isExpectedEqFloat(a float64) func(interface{}) bool {
	return func(v interface{}) bool {
		vv, _ := v.(pgxdecimal.Decimal).V.Float64()
		return a == vv
	}
}

func TestValueRoundTrip(t *testing.T) {
	t.Parallel()
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "numeric", []pgxtest.ValueRoundTripTest{
		{
			Param:  requireFromString("1"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("1")),
		},
		{
			Param:  requireFromString("0.000012345"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("0.000012345")),
		},
		{
			Param:  requireFromString("123456.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("123456.123456")),
		},
		{
			Param:  requireFromString("-1"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("-1")),
		},
		{
			Param:  requireFromString("-0.000012345"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("-0.000012345")),
		},
		{
			Param:  requireFromString("-123456.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("-123456.123456")),
		},
		{
			Param:  requireFromString("1"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("1")),
		},
		{
			Param:  requireFromString("0.000012345"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("0.000012345")),
		},
		{
			Param:  requireFromString("123456.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("123456.123456")),
		},
		{
			Param:  requireFromString("-1"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("-1")),
		},
		{
			Param:  requireFromString("-0.000012345"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("-0.000012345")),
		},
		{
			Param:  requireFromString("-123456.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("-123456.123456")),
		},
		{
			Param:  requireFromString("-123456000000000.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("-123456000000000.123456")),
		},
		{
			Param:  requireFromString("12345600000000000000.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("12345600000000000000.123456")),
		},
		{
			Param:  requireFromString("99.0000000000000000009"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(requireFromString("99.0000000000000000009")),
		},
	})
}

func TestValueRoundTripFloat8(t *testing.T) {
	t.Parallel()
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "float8", []pgxtest.ValueRoundTripTest{
		{
			Param:  requireFromString("1"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(1),
		},
		{
			Param:  requireFromString("0.000012345"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(0.000012345),
		},
		{
			Param:  requireFromString("123456.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(123456.123456),
		},
		{
			Param:  requireFromString("-1"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(-1),
		},
		{
			Param:  requireFromString("-0.000012345"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(-0.000012345),
		},
		{
			Param:  requireFromString("-123456.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(-123456.123456),
		},
		{
			Param:  requireFromString("1"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(1),
		},
		{
			Param:  requireFromString("0.000012345"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(0.000012345),
		},
		{
			Param:  requireFromString("123456.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(123456.123456),
		},
		{
			Param:  requireFromString("-1"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(-1),
		},
		{
			Param:  requireFromString("-0.000012345"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(-0.000012345),
		},
		{
			Param:  requireFromString("-123456.123456"),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqFloat(-123456.123456),
		},
	})
}

func TestValueRoundTripInt8(t *testing.T) {
	t.Parallel()
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "int8", []pgxtest.ValueRoundTripTest{
		{
			Param:  newFromInt(0),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(0)),
		},
		{
			Param:  newFromInt(1),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(1)),
		},
		{
			Param:  newFromInt(math.MaxInt64),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(math.MaxInt64)),
		},
		{
			Param:  newFromInt(-1),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(-1)),
		},
		{
			Param:  newFromInt(math.MinInt64),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(math.MinInt64)),
		},
		{
			Param:  newFromInt(0),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(0)),
		},
		{
			Param:  newFromInt(1),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(1)),
		},
		{
			Param:  newFromInt(math.MaxInt64),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(math.MaxInt64)),
		},
		{
			Param:  newFromInt(-1),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(-1)),
		},
		{
			Param:  newFromInt(math.MinInt64),
			Result: new(pgxdecimal.Decimal),
			Test:   isExpectedEqDecimal(newFromInt(math.MinInt64)),
		},
	})
}

func requireFromString(v string) pgxdecimal.Decimal {
	original := &decimal.Big{}
	original.SetString(v)
	return pgxdecimal.Decimal{V: original}
}

func newFromInt(v int64) pgxdecimal.Decimal {
	return pgxdecimal.Decimal{V: decimal.New(v, 0)}
}
