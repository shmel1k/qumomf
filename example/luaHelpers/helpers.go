package luaHelpers

import (
	"fmt"
)

func ParseInt(f interface{}) (int, error) {
	switch t := f.(type) {
	case uint:
		return int(t), nil
	case int:
		return t, nil
	case uint8:
		return int(t), nil
	case int8:
		return int(t), nil
	case uint16:
		return int(t), nil
	case int16:
		return int(t), nil
	case uint32:
		return int(t), nil
	case int32:
		return int(t), nil
	case uint64:
		return int(t), nil
	case int64:
		return int(t), nil
	}
	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseUint(f interface{}) (uint, error) {
	switch t := f.(type) {
	case uint:
		return t, nil
	case int:
		return uint(t), nil
	case uint8:
		return uint(t), nil
	case int8:
		return uint(t), nil
	case uint16:
		return uint(t), nil
	case int16:
		return uint(t), nil
	case uint32:
		return uint(t), nil
	case int32:
		return uint(t), nil
	case uint64:
		return uint(t), nil
	case int64:
		return uint(t), nil
	}
	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseUint8(f interface{}) (uint8, error) {
	switch t := f.(type) {
	case uint:
		return uint8(t), nil
	case int:
		return uint8(t), nil
	case uint8:
		return t, nil
	case int8:
		return uint8(t), nil
	case uint16:
		return uint8(t), nil
	case int16:
		return uint8(t), nil
	case uint32:
		return uint8(t), nil
	case int32:
		return uint8(t), nil
	case uint64:
		return uint8(t), nil
	case int64:
		return uint8(t), nil
	}
	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseUint16(f interface{}) (uint16, error) {
	switch t := f.(type) {
	case uint:
		return uint16(t), nil
	case int:
		return uint16(t), nil
	case uint8:
		return uint16(t), nil
	case int8:
		return uint16(t), nil
	case uint16:
		return t, nil
	case int16:
		return uint16(t), nil
	case uint32:
		return uint16(t), nil
	case int32:
		return uint16(t), nil
	case uint64:
		return uint16(t), nil
	case int64:
		return uint16(t), nil
	}
	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseUint32(f interface{}) (uint32, error) {
	switch t := f.(type) {
	case uint:
		return uint32(t), nil
	case int:
		return uint32(t), nil
	case uint8:
		return uint32(t), nil
	case int8:
		return uint32(t), nil
	case uint16:
		return uint32(t), nil
	case int16:
		return uint32(t), nil
	case uint32:
		return t, nil
	case int32:
		return uint32(t), nil
	case uint64:
		return uint32(t), nil
	case int64:
		return uint32(t), nil
	}
	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseUint64(f interface{}) (uint64, error) {
	switch t := f.(type) {
	case uint:
		return uint64(t), nil
	case int:
		return uint64(t), nil
	case uint8:
		return uint64(t), nil
	case int8:
		return uint64(t), nil
	case int16:
		return uint64(t), nil
	case uint16:
		return uint64(t), nil
	case int32:
		return uint64(t), nil
	case uint32:
		return uint64(t), nil
	case uint64:
		return t, nil
	case int64:
		return uint64(t), nil
	default:
		return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
	}
}

func ParseInt8(f interface{}) (int8, error) {
	switch t := f.(type) {
	case uint:
		return int8(t), nil
	case int:
		return int8(t), nil
	case uint8:
		return int8(t), nil
	case int8:
		return t, nil
	case uint16:
		return int8(t), nil
	case int16:
		return int8(t), nil
	case uint32:
		return int8(t), nil
	case int32:
		return int8(t), nil
	case uint64:
		return int8(t), nil
	case int64:
		return int8(t), nil
	}
	return 0, nil
}

func ParseInt16(f interface{}) (int16, error) {
	switch t := f.(type) {
	case uint:
		return int16(t), nil
	case int:
		return int16(t), nil
	case uint8:
		return int16(t), nil
	case int8:
		return int16(t), nil
	case uint16:
		return int16(t), nil
	case int16:
		return t, nil
	case uint32:
		return int16(t), nil
	case int32:
		return int16(t), nil
	case uint64:
		return int16(t), nil
	case int64:
		return int16(t), nil
	}
	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseInt32(f interface{}) (int32, error) {
	switch t := f.(type) {
	case uint:
		return int32(t), nil
	case int:
		return int32(t), nil
	case uint8:
		return int32(t), nil
	case int8:
		return int32(t), nil
	case uint16:
		return int32(t), nil
	case int16:
		return int32(t), nil
	case uint32:
		return int32(t), nil
	case int32:
		return t, nil
	case uint64:
		return int32(t), nil
	case int64:
		return int32(t), nil
	}
	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseInt64(f interface{}) (int64, error) {
	switch t := f.(type) {
	case int:
		return int64(t), nil
	case uint:
		return int64(t), nil
	case uint8:
		return int64(t), nil
	case int8:
		return int64(t), nil
	case uint16:
		return int64(t), nil
	case int16:
		return int64(t), nil
	case uint32:
		return int64(t), nil
	case int32:
		return int64(t), nil
	case uint64:
		return int64(t), nil
	case int64:
		return t, nil
	}
	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseFloat32(f interface{}) (float32, error) {
	switch t := f.(type) {
	case uint:
		return float32(t), nil
	case int:
		return float32(t), nil
	case uint8:
		return float32(t), nil
	case int8:
		return float32(t), nil
	case uint16:
		return float32(t), nil
	case int16:
		return float32(t), nil
	case uint32:
		return float32(t), nil
	case int32:
		return float32(t), nil
	case uint64:
		return float32(t), nil
	case int64:
		return float32(t), nil
	case float32:
		return t, nil
	case float64:
		return float32(t), nil
	}

	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseFloat64(f interface{}) (float64, error) {
	// NOTE(a.petrukhin): tarantool can serialize small integers to uint64, to int64 and to float.
	switch t := f.(type) {
	case uint:
		return float64(t), nil
	case int:
		return float64(t), nil
	case uint8:
		return float64(t), nil
	case int8:
		return float64(t), nil
	case uint16:
		return float64(t), nil
	case int16:
		return float64(t), nil
	case uint32:
		return float64(t), nil
	case int32:
		return float64(t), nil
	case uint64:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case float32:
		return float64(t), nil
	case float64:
		return t, nil
	}

	return 0, fmt.Errorf("got invalid type %T for value %v", f, f)
}

func ParseString(f interface{}) (string, error) {
	switch t := f.(type) {
	case string:
		return t, nil
	}
	return "", fmt.Errorf("got invalid type %T for value %v", f, f)
}

func Scan(tuple []interface{}, dest ...interface{}) error {
	if len(dest) > len(tuple) {
		return fmt.Errorf("got invalid destination size %d, tuple len=%d", len(tuple), len(dest))
	}

	var err error
	for i := 0; i < len(dest); i++ {
		switch t := dest[i].(type) {
		case *string:
			*t, err = ParseString(tuple[i])
		case *uint:
			*t, err = ParseUint(tuple[i])
		case *uint8:
			*t, err = ParseUint8(tuple[i])
		case *uint16:
			*t, err = ParseUint16(tuple[i])
		case *uint32:
			*t, err = ParseUint32(tuple[i])
		case *uint64:
			*t, err = ParseUint64(tuple[i])
		case *int:
			*t, err = ParseInt(tuple[i])
		case *int8:
			*t, err = ParseInt8(tuple[i])
		case *int16:
			*t, err = ParseInt16(tuple[i])
		case *int32:
			*t, err = ParseInt32(tuple[i])
		case *int64:
			*t, err = ParseInt64(tuple[i])
		case *float32:
			*t, err = ParseFloat32(tuple[i])
		case *float64:
			*t, err = ParseFloat64(tuple[i])
		default:
			err = fmt.Errorf("got unknown type %T", t)
		}
		if err != nil {
			err = fmt.Errorf("failed to parse %d destination value: %s", i, err)
			break
		}
	}
	return err
}
