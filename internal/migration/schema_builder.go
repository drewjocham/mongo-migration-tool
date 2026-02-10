package migration

import "go.mongodb.org/mongo-driver/v2/bson"

type SchemaBuilder struct {
	root bson.M
}

func Schema() *SchemaBuilder {
	return &SchemaBuilder{
		root: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
			},
		},
	}
}

func (s *SchemaBuilder) BsonType(t string) *SchemaBuilder {
	s.ensureSchema()["bsonType"] = t
	return s
}

func (s *SchemaBuilder) Required(fields ...string) *SchemaBuilder {
	if len(fields) == 0 {
		return s
	}
	s.ensureSchema()["required"] = fields
	return s
}

func (s *SchemaBuilder) Field(name string, props bson.M) *SchemaBuilder {
	if name == "" {
		return s
	}
	propsMap := s.ensureProperties()
	propsMap[name] = props
	return s
}

func (s *SchemaBuilder) String() bson.M {
	return bson.M{"bsonType": "string"}
}

func (s *SchemaBuilder) Int() bson.M {
	return bson.M{"bsonType": "int"}
}

func (s *SchemaBuilder) Long() bson.M {
	return bson.M{"bsonType": "long"}
}

func (s *SchemaBuilder) Bool() bson.M {
	return bson.M{"bsonType": "bool"}
}

func (s *SchemaBuilder) Date() bson.M {
	return bson.M{"bsonType": "date"}
}

func (s *SchemaBuilder) Array(items bson.M) bson.M {
	return bson.M{
		"bsonType": "array",
		"items":    items,
	}
}

func (s *SchemaBuilder) Object(props bson.M) bson.M {
	return bson.M{
		"bsonType":   "object",
		"properties": props,
	}
}

func (s *SchemaBuilder) Enum(values ...string) bson.M {
	return bson.M{
		"enum": values,
	}
}

func (s *SchemaBuilder) MinLength(n int) bson.M {
	return bson.M{
		"minLength": n,
	}
}

func (s *SchemaBuilder) MaxLength(n int) bson.M {
	return bson.M{
		"maxLength": n,
	}
}

func (s *SchemaBuilder) Build() bson.M {
	return s.root
}

func (s *SchemaBuilder) ensureSchema() bson.M {
	schema, ok := s.root["$jsonSchema"].(bson.M)
	if !ok {
		schema = bson.M{}
		s.root["$jsonSchema"] = schema
	}
	return schema
}

func (s *SchemaBuilder) ensureProperties() bson.M {
	schema := s.ensureSchema()
	props, ok := schema["properties"].(bson.M)
	if !ok {
		props = bson.M{}
		schema["properties"] = props
	}
	return props
}
