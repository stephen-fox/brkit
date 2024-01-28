package bstruct

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
)

type Byter interface {
	ToBytes(binary.ByteOrder) []byte
}

type FieldInfo struct {
	Index int
	Name  string
	Type  string
	Value []byte
}

func StructToBytesOrExit(s interface{}, bo binary.ByteOrder, optFn func(FieldInfo) error) []byte {
	b, err := StructToBytes(s, bo, optFn)
	if err != nil {
		DefaultExitFn(err)
	}

	return b
}

func StructToBytes(s interface{}, bo binary.ByteOrder, optFn func(FieldInfo) error) ([]byte, error) {
	if s == nil {
		return nil, errors.New("struct is nil")
	}

	structValue := reflect.ValueOf(s)

	numFields := structValue.NumField()

	structType := structValue.Type()

	var b []byte

	for i := 0; i < numFields; i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		//fmt.Printf("field %d: %s (%s) = %T\n", i,
		//	field.Name, field.Type, fieldValue.Interface())

		at := len(b)

		switch t := fieldValue.Interface().(type) {
		case Byter:
			b = append(b, t.ToBytes(bo)...)
		case uint8:
			b = append(b, t)
		case uint16:
			b = append(b, make([]byte, 2)...)
			bo.PutUint16(b[len(b)-2:], t)
		case uint32:
			b = append(b, make([]byte, 4)...)
			bo.PutUint32(b[len(b)-4:], t)
		case uint64:
			b = append(b, make([]byte, 8)...)
			bo.PutUint64(b[len(b)-8:], t)
		default:
			return nil, fmt.Errorf("unsupported data type %T for field %q (index %d)",
				t, field.Name, i)
		}

		if optFn != nil {
			err := optFn(FieldInfo{
				Index: i,
				Name:  field.Name,
				Type:  field.Type.String(),
				Value: b[at:],
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return b, nil
}
