package bstruct

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
)

type FieldOrder int

const (
	GoFieldOrder FieldOrder = iota
	ReverseFieldOrder
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

func FieldWriterFn(w io.Writer, optLogger ...*log.Logger) func(FieldInfo) error {
	return func(f FieldInfo) error {
		if len(optLogger) > 0 {
			logger := optLogger[0]

			hexDump := hex.Dump(f.Value)
			if len(hexDump) <= 1 {
				// hex.Dump always adds a newline.
				hexDump = "<empty-value>"
			} else {
				hexDump = hexDump[0 : len(hexDump)-1]
			}

			logger.Printf("writing field: %d | name: %q | type: %s | value:\n%s",
				f.Index, f.Name, f.Type, hexDump)
		}

		_, err := w.Write(f.Value)
		return err
	}
}

func ToBytesX86OrExit(fieldFn func(FieldInfo) error, s interface{}) {
	err := ToBytesX86(fieldFn, s)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to convert struct to bytes for x86 - %w", err))
	}
}

func ToBytesX86(fieldFn func(FieldInfo) error, s interface{}) error {
	err := ToBytes(binary.LittleEndian, GoFieldOrder, fieldFn, s)
	if err != nil {
		return err
	}

	return nil
}

func ToBytesOrExit(bo binary.ByteOrder, fo FieldOrder, fieldFn func(FieldInfo) error, s interface{}) {
	err := ToBytes(bo, fo, fieldFn, s)
	if err != nil {
		DefaultExitFn(fmt.Errorf("failed to convert struct to bytes - %w", err))
	}
}

func ToBytes(bo binary.ByteOrder, fo FieldOrder, fieldFn func(FieldInfo) error, s interface{}) error {
	if bo == nil {
		return errors.New("binary order is nil")
	}

	switch fo {
	case GoFieldOrder, ReverseFieldOrder:
		// OK.
	default:
		return fmt.Errorf("unsupported field order: %v", fo)
	}

	if s == nil {
		return errors.New("struct is nil")
	}

	structValue := reflect.ValueOf(s)

	numFields := structValue.NumField()
	if numFields == 0 {
		return errors.New("struct contains no fields")
	}

	structType := structValue.Type()

	i := 0
	if fo == ReverseFieldOrder {
		i = numFields - 1
	}

	for {
		structField := structType.Field(i)

		err := parseField(parseFieldArgs{
			index:      i,
			field:      structField,
			fieldValue: structValue.Field(i),
			bo:         bo,
			fieldFn:    fieldFn,
		})
		if err != nil {
			return fmt.Errorf("failed to parse field %q - %w", structField.Name, err)
		}

		if fo == ReverseFieldOrder {
			i--
		} else {
			i++
		}

		if i < 0 || i == numFields {
			break
		}
	}

	return nil
}

type parseFieldArgs struct {
	index      int
	field      reflect.StructField
	fieldValue reflect.Value
	bo         binary.ByteOrder
	fieldFn    func(FieldInfo) error
}

func parseField(args parseFieldArgs) error {
	if !args.field.IsExported() {
		v, hasIt := args.field.Tag.Lookup("brkit")
		if hasIt && v == "-" {
			return nil
		}

		return errors.New("field is not exported - it can be explicitly ignored using the tag `brkit:\"-\"`")
	}

	var b []byte

	switch t := args.fieldValue.Interface().(type) {
	case Byter:
		b = t.ToBytes(args.bo)
	case uint8:
		b = []byte{t}
	case uint16:
		b = make([]byte, 2)
		args.bo.PutUint16(b, t)
	case uint32:
		b = make([]byte, 4)
		args.bo.PutUint32(b, t)
	case uint64:
		b = make([]byte, 8)
		args.bo.PutUint64(b, t)
	default:
		return fmt.Errorf("unsupported data type %T for field %q",
			t, args.field.Name)
	}

	if args.fieldFn == nil {
		return errors.New("field function is nil")
	}

	err := args.fieldFn(FieldInfo{
		Index: args.index,
		Name:  args.field.Name,
		Type:  args.field.Type.String(),
		Value: b,
	})
	if err != nil {
		return err
	}

	return nil
}
