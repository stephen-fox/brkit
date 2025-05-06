package bstruct

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
)

// FieldOrder specifies how a struct's fields should be ordered.
type FieldOrder int

const (
	// GoFieldOrder orders fields in the order that they
	// were defined.
	GoFieldOrder FieldOrder = iota

	// ReverseFieldOrder orders the fields in the opposite
	// order that they were defined in.
	ReverseFieldOrder
)

// Byter is the interface that specifies how an object should
// be converted to bytes.
//
// Refer to the ToBytes function for more information.
type Byter interface {
	// ToBytes specifies how the object should be converted
	// to bytes for the given binary.ByteOrder.
	ToBytes(binary.ByteOrder) []byte
}

// FieldInfo provides information about a field.
type FieldInfo struct {
	// Index is the zero-based index number of the field.
	Index int

	// Name is the name of the field.
	Name string

	// Type is the string representation of the Go datatype.
	Type string

	// Value is the value of the field after it has been
	// converted to bytes.
	Value []byte
}

// FieldWriterFn returns a function that writes a field's value
// to the specified io.Writer.
//
// An optional log.Logger can be provided as well. If non-nil,
// information about the field including its hexdump-style
// value will be written to the log.Logger.
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

			logger.Printf("bstruct.fieldwriterfn - field: %d | name: %q | type: %s | value:\n%s",
				f.Index, f.Name, f.Type, hexDump)
		}

		_, err := w.Write(f.Value)
		return err
	}
}

// ToBytesX86OrExit calls ToBytesX86. It calls DefaultExitFn if
// an error occurs.
func ToBytesX86OrExit(fieldFn func(FieldInfo) error, s interface{}) {
	err := ToBytesX86(fieldFn, s)
	if err != nil {
		DefaultExitFn(fmt.Errorf("bstruct: failed to convert struct to bytes for x86 - %w", err))
	}
}

// ToBytesX86 converts struct s to bytes using fieldFn for a x86 CPU.
//
// Refer to ToBytes for more information.
func ToBytesX86(fieldFn func(FieldInfo) error, s interface{}) error {
	err := ToBytes(binary.LittleEndian, GoFieldOrder, fieldFn, s)
	if err != nil {
		return err
	}

	return nil
}

// ToBytesOrExit calls ToBytes. It calls DefaultExitFn if an error occurs.
func ToBytesOrExit(bo binary.ByteOrder, fo FieldOrder, fieldFn func(FieldInfo) error, s interface{}) {
	err := ToBytes(bo, fo, fieldFn, s)
	if err != nil {
		DefaultExitFn(fmt.Errorf("bstruct: failed to convert struct to bytes - %w", err))
	}
}

// ToBytes converts struct s to bytes. Each field's value is converted
// according to the specified binary.ByteOrder. Fields are passed to
// fieldFn according to the specified FieldOrder.
//
// If a field in struct s implements the Byter interface, its ToBytes
// method is used to convert the field value to bytes.
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
		// TODO: Should we error in this case? Maybe we can
		// make the behavior configurable using a global
		// variable?
		return nil
	}

	buf := bytes.NewBuffer(nil)
	var err error

	switch t := args.fieldValue.Interface().(type) {
	case Byter:
		_, err = buf.Write(t.ToBytes(args.bo))
	default:
		err = binary.Write(buf, args.bo, t)
	}

	if err != nil {
		return err
	}

	if args.fieldFn == nil {
		return errors.New("field function is nil")
	}

	err = args.fieldFn(FieldInfo{
		Index: args.index,
		Name:  args.field.Name,
		Type:  args.field.Type.String(),
		Value: buf.Bytes(),
	})
	if err != nil {
		return err
	}

	return nil
}
