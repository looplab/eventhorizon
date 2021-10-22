package bson

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// Update the default BSON registry to be able to handle UUID types as strings.
func init() {
	rb := bson.NewRegistryBuilder()
	id := uuid.Nil
	uuidType := reflect.TypeOf(id)

	rb.RegisterTypeEncoder(uuidType, bsoncodec.ValueEncoderFunc(
		func(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
			if !val.IsValid() || val.Type() != uuidType || val.Len() != 16 {
				return bsoncodec.ValueEncoderError{
					Name:     "uuid.UUID",
					Types:    []reflect.Type{uuidType},
					Received: val,
				}
			}
			b := make([]byte, 16)
			v := reflect.ValueOf(b)
			reflect.Copy(v, val)
			id, err := uuid.FromBytes(v.Bytes())
			if err != nil {
				return fmt.Errorf("could not parse UUID bytes (%x): %w", v.Bytes(), err)
			}

			return vw.WriteString(id.String())
		},
	))

	rb.RegisterTypeDecoder(uuidType, bsoncodec.ValueDecoderFunc(
		func(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
			if !val.IsValid() || !val.CanSet() || val.Kind() != reflect.Array {
				return bsoncodec.ValueDecoderError{
					Name:     "uuid.UUID",
					Kinds:    []reflect.Kind{reflect.Bool},
					Received: val,
				}
			}

			var s string
			switch vr.Type() {
			case bsontype.String:
				var err error
				if s, err = vr.ReadString(); err != nil {
					return err
				}
			default:
				return fmt.Errorf("received invalid BSON type to decode into UUID: %s", vr.Type())
			}

			id, err := uuid.Parse(s)
			if err != nil {
				return fmt.Errorf("could not parse UUID string: %s", s)
			}
			v := reflect.ValueOf(id)
			if !v.IsValid() || v.Kind() != reflect.Array {
				return fmt.Errorf("invalid kind of reflected UUID value: %s", v.Kind().String())
			}
			reflect.Copy(val, v)

			return nil
		},
	))

	bson.DefaultRegistry = rb.Build()
}
