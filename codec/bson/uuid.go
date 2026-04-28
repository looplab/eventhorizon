package bson

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Registry is a custom BSON registry that handles UUID types as strings.
// In mongo-driver v2, there is no global default registry. Consumers must
// pass this registry to their MongoDB client, database, or collection via
// options.Client().SetRegistry(bsoncodec.Registry).
var Registry *bson.Registry

func init() {
	Registry = newUUIDRegistry()
}

// Marshal marshals a value to BSON using the custom Registry that handles
// UUID encoding as strings. Use this instead of bson.Marshal to ensure
// consistent UUID encoding across driver versions.
func Marshal(val any) ([]byte, error) {
	var buf bytes.Buffer
	enc := bson.NewEncoder(bson.NewDocumentWriter(&buf))
	enc.SetRegistry(Registry)

	if err := enc.Encode(val); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Unmarshal unmarshals BSON data using the custom Registry that handles
// UUID decoding from both strings and binary formats. Use this instead of
// bson.Unmarshal to ensure backward-compatible UUID decoding.
func Unmarshal(data []byte, val any) error {
	dec := bson.NewDecoder(bson.NewDocumentReader(bytes.NewReader(data)))
	dec.SetRegistry(Registry)

	return dec.Decode(val)
}

func newUUIDRegistry() *bson.Registry {
	reg := bson.NewRegistry()
	id := uuid.Nil
	uuidType := reflect.TypeOf(id)

	reg.RegisterTypeEncoder(uuidType, bson.ValueEncoderFunc(
		func(ec bson.EncodeContext, vw bson.ValueWriter, val reflect.Value) error {
			if !val.IsValid() || val.Type() != uuidType || val.Len() != 16 {
				return bson.ValueEncoderError{
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

	reg.RegisterTypeDecoder(uuidType, bson.ValueDecoderFunc(
		func(dc bson.DecodeContext, vr bson.ValueReader, val reflect.Value) error {
			if !val.IsValid() || !val.CanSet() || val.Kind() != reflect.Array {
				return bson.ValueDecoderError{
					Name:     "uuid.UUID",
					Kinds:    []reflect.Kind{reflect.Array},
					Received: val,
				}
			}

			var id uuid.UUID
			switch vr.Type() {
			case bson.TypeString:
				s, err := vr.ReadString()
				if err != nil {
					return err
				}

				id, err = uuid.Parse(s)
				if err != nil {
					return fmt.Errorf("could not parse UUID string: %s", s)
				}
			case bson.TypeBinary:
				data, _, err := vr.ReadBinary()
				if err != nil {
					return err
				}

				id, err = uuid.FromBytes(data)
				if err != nil {
					return fmt.Errorf("could not parse UUID binary (%x): %w", data, err)
				}
			default:
				return fmt.Errorf("received invalid BSON type to decode into UUID: %s", vr.Type())
			}

			v := reflect.ValueOf(id)
			if !v.IsValid() || v.Kind() != reflect.Array {
				return fmt.Errorf("invalid kind of reflected UUID value: %s", v.Kind().String())
			}
			reflect.Copy(val, v)

			return nil
		},
	))

	return reg
}
