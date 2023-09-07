package decimal

import (
	"fmt"
	"math"
	"math/big"
	"reflect"

	"github.com/ericlagergren/decimal"
	"github.com/ericlagergren/decimal/sql/postgres"
	"github.com/jackc/pgx/v5/pgtype"
)

// Decimal has to be used to prevent pgx from using the database/sql scanner function
type Decimal postgres.Decimal

func (d *Decimal) ScanNumeric(v pgtype.Numeric) error {
	if !v.Valid {
		*d = Decimal{}
		return nil
	}

	if v.NaN {
		return fmt.Errorf("cannot scan NaN into *postgres.Decimal")
	}

	if v.InfinityModifier != pgtype.Finite {
		return fmt.Errorf("cannot scan %v into *postgres.Decimal", v.InfinityModifier)
	}

	var bd *decimal.Big
	if v.Int.IsInt64() {
		// fast path avoids one allocation
		bd = decimal.New(v.Int.Int64(), -int(v.Exp))
	} else {
		bd = new(decimal.Big).SetBigMantScale(v.Int, -int(v.Exp))
	}

	*d = Decimal{
		V:    bd,
		Zero: len(v.Int.Bits()) == 0,
	}

	return nil
}

func (d Decimal) NumericValue() (pgtype.Numeric, error) {
	if d.V == nil {
		return pgtype.Numeric{Valid: false}, nil
	}

	rawAsUint, rawAsBigInt := decimal.Raw(d.V)
	if rawAsUint != nil && len(rawAsBigInt.Bits()) == 0 {
		i := int64(*rawAsUint)
		if d.V.Signbit() {
			i *= -1
		}
		return pgtype.Numeric{Int: big.NewInt(i), Exp: -int32(d.V.Scale()), Valid: true}, nil
	}
	if d.V.Sign() != rawAsBigInt.Sign() {
		rawAsBigInt = rawAsBigInt.Neg(rawAsBigInt)
	}
	return pgtype.Numeric{Int: rawAsBigInt, Exp: -int32(d.V.Scale()), Valid: true}, nil
}

func (d *Decimal) ScanFloat64(v pgtype.Float8) error {
	if !v.Valid {
		*d = Decimal(postgres.Decimal{})
		return nil
	}

	if math.IsNaN(v.Float64) {
		return fmt.Errorf("cannot scan NaN into *postgres.Decimal")
	}

	if math.IsInf(v.Float64, 0) {
		return fmt.Errorf("cannot scan %v into *postgres.Decimal", v.Float64)
	}

	res := &decimal.Big{}
	res = res.SetFloat64(v.Float64)
	*d = Decimal(postgres.Decimal{V: res, Zero: v.Float64 == 0})

	return nil
}

func (d Decimal) Float64Value() (pgtype.Float8, error) {
	dd := postgres.Decimal(d)
	floatVal, _ := dd.V.Float64()
	return pgtype.Float8{Float64: floatVal, Valid: true}, nil
}

func (d *Decimal) ScanInt64(v pgtype.Int8) error {
	if !v.Valid {
		return fmt.Errorf("cannot scan NULL into *postgres.Decimal")
	}

	*d = Decimal(postgres.Decimal{V: decimal.New(v.Int64, 0)})
	return nil
}

func (d Decimal) Int64Value() (pgtype.Int8, error) {
	dd := postgres.Decimal(d)

	if !dd.V.IsInt() {
		return pgtype.Int8{}, fmt.Errorf("cannot convert %v to int64", dd)
	}

	i64, _ := dd.V.Int64()
	return pgtype.Int8{Int64: i64, Valid: true}, nil
}

func TryWrapNumericEncodePlan(value interface{}) (plan pgtype.WrappedEncodePlanNextSetter, nextValue interface{}, ok bool) {
	if value, ok := value.(postgres.Decimal); ok {
		return &wrapDecimalEncodePlan{}, Decimal(value), true
	}

	return nil, nil, false
}

type wrapDecimalEncodePlan struct {
	next pgtype.EncodePlan
}

func (plan *wrapDecimalEncodePlan) SetNext(next pgtype.EncodePlan) { plan.next = next }

func (plan *wrapDecimalEncodePlan) Encode(value interface{}, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(Decimal(value.(postgres.Decimal)), buf)
}

func TryWrapNumericScanPlan(target interface{}) (plan pgtype.WrappedScanPlanNextSetter, nextDst interface{}, ok bool) {
	if target, ok := target.(*postgres.Decimal); ok {
		return &wrapDecimalScanPlan{}, (*Decimal)(target), true
	}

	return nil, nil, false
}

type wrapDecimalScanPlan struct {
	next pgtype.ScanPlan
}

func (plan *wrapDecimalScanPlan) SetNext(next pgtype.ScanPlan) { plan.next = next }

func (plan *wrapDecimalScanPlan) Scan(src []byte, dst interface{}) error {
	return plan.next.Scan(src, (*Decimal)(dst.(*postgres.Decimal)))
}

type NumericCodec struct {
	pgtype.NumericCodec
}

func (NumericCodec) DecodeValue(tm *pgtype.Map, oid uint32, format int16, src []byte) (interface{}, error) {
	if src == nil {
		return nil, nil
	}

	var target postgres.Decimal
	scanPlan := tm.PlanScan(oid, format, &target)
	if scanPlan == nil {
		return nil, fmt.Errorf("PlanScan did not find a plan")
	}

	err := scanPlan.Scan(src, &target)
	if err != nil {
		return nil, err
	}

	return target, nil
}

// Register registers the ericlagergren/decimal integration with a pgtype.ConnInfo.
func Register(m *pgtype.Map) {
	m.TryWrapEncodePlanFuncs = append([]pgtype.TryWrapEncodePlanFunc{TryWrapNumericEncodePlan}, m.TryWrapEncodePlanFuncs...)
	m.TryWrapScanPlanFuncs = append([]pgtype.TryWrapScanPlanFunc{TryWrapNumericScanPlan}, m.TryWrapScanPlanFuncs...)

	m.RegisterType(&pgtype.Type{
		Name:  "numeric",
		OID:   pgtype.NumericOID,
		Codec: NumericCodec{},
	})

	registerDefaultPgTypeVariants := func(name, arrayName string, value interface{}) {
		// T
		m.RegisterDefaultPgType(value, name)

		// *T
		valueType := reflect.TypeOf(value)
		m.RegisterDefaultPgType(reflect.New(valueType).Interface(), name)

		// []T
		sliceType := reflect.SliceOf(valueType)
		m.RegisterDefaultPgType(reflect.MakeSlice(sliceType, 0, 0).Interface(), arrayName)

		// *[]T
		m.RegisterDefaultPgType(reflect.New(sliceType).Interface(), arrayName)

		// []*T
		sliceOfPointerType := reflect.SliceOf(reflect.TypeOf(reflect.New(valueType).Interface()))
		m.RegisterDefaultPgType(reflect.MakeSlice(sliceOfPointerType, 0, 0).Interface(), arrayName)

		// *[]*T
		m.RegisterDefaultPgType(reflect.New(sliceOfPointerType).Interface(), arrayName)
	}

	registerDefaultPgTypeVariants("numeric", "_numeric", postgres.Decimal{})
	registerDefaultPgTypeVariants("numeric", "_numeric", Decimal{})
}
