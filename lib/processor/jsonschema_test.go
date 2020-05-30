package processor

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/windhooked/benthos/v3/lib/log"
	"github.com/windhooked/benthos/v3/lib/message"
	"github.com/windhooked/benthos/v3/lib/metrics"
	"github.com/windhooked/benthos/v3/lib/types"
)

func TestJSONSchemaExternalSchemaCheck(t *testing.T) {
	schema := `{
		"$id": "https://example.com/person.schema.json",
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Person",
		"type": "object",
		"properties": {
		  "firstName": {
			"type": "string",
			"description": "The person's first name."
		  },
		  "lastName": {
			"type": "string",
			"description": "The person's last name."
		  },
		  "age": {
			"description": "Age in years which must be equal to or greater than zero.",
			"type": "integer",
			"minimum": 0
		  }
		}
	}`

	tmpSchemaFile, err := ioutil.TempFile("", "benthos_jsonschema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpSchemaFile.Name())

	// write schema definition to tmpfile
	if _, err := tmpSchemaFile.Write([]byte(schema)); err != nil {
		t.Fatal(err)
	}

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	testMet := metrics.DudType{}

	type fields struct {
		schemaPath string
	}
	tests := []struct {
		name   string
		fields fields
		arg    [][]byte
		output string
		err    string
	}{
		{
			name: "schema match",
			fields: fields{
				schemaPath: fmt.Sprintf("file://%s", tmpSchemaFile.Name()),
			},
			arg: [][]byte{
				[]byte(`{"firstName":"John","lastName":"Doe","age":21}`),
			},
			output: `{"firstName":"John","lastName":"Doe","age":21}`,
		},
		{
			name: "schema no match",
			fields: fields{
				schemaPath: fmt.Sprintf("file://%s", tmpSchemaFile.Name()),
			},
			arg: [][]byte{
				[]byte(`{"firstName":"John","lastName":"Doe","age":-20}`),
			},
			output: `{"firstName":"John","lastName":"Doe","age":-20}`,
			err:    `age must be greater than or equal to 0`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := NewConfig()
			conf.Type = "jsonschema"
			conf.JSONSchema.SchemaPath = tt.fields.schemaPath

			c, err := NewJSONSchema(conf, nil, testLog, testMet)
			if err != nil {
				t.Error(err)
				return
			}
			msgs, _ := c.ProcessMessage(message.New(tt.arg))

			if len(msgs) != 1 {
				t.Fatalf("Test '%v' did not succeed", tt.name)
			}

			if exp, act := tt.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
				t.Errorf("Wrong result '%v': %v != %v", tt.name, act, exp)
			}
			msgs[0].Iter(func(i int, part types.Part) error {
				act := part.Metadata().Get(FailFlagKey)
				if len(act) > 0 && act != tt.err {
					t.Errorf("Wrong error message '%v': %v != %v", tt.name, act, tt.err)
				}
				return nil
			})
		})
	}
}

func TestJSONSchemaInlineSchemaCheck(t *testing.T) {
	schemaDef := `{
		"$id": "https://example.com/person.schema.json",
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "Person",
		"type": "object",
		"properties": {
		  "firstName": {
			"type": "string",
			"description": "The person's first name."
		  },
		  "lastName": {
			"type": "string",
			"description": "The person's last name."
		  },
		  "age": {
			"description": "Age in years which must be equal to or greater than zero.",
			"type": "integer",
			"minimum": 0
		  }
		}
	}`

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	testMet := metrics.DudType{}

	type fields struct {
		schema string
		part   int
	}
	tests := []struct {
		name   string
		fields fields
		arg    [][]byte
		output string
		err    string
	}{
		{
			name: "schema match",
			fields: fields{
				schema: schemaDef,
				part:   0,
			},
			arg: [][]byte{
				[]byte(`{"firstName":"John","lastName":"Doe","age":21}`),
			},
			output: `{"firstName":"John","lastName":"Doe","age":21}`,
		},
		{
			name: "schema no match",
			fields: fields{
				schema: schemaDef,
				part:   0,
			},
			arg: [][]byte{
				[]byte(`{"firstName":"John","lastName":"Doe","age":-20}`),
			},
			output: `{"firstName":"John","lastName":"Doe","age":-20}`,
			err:    `age must be greater than or equal to 0`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := NewConfig()
			conf.Type = "jsonschema"
			conf.JSONSchema.Schema = tt.fields.schema
			conf.JSONSchema.Parts = []int{0}

			c, err := NewJSONSchema(conf, nil, testLog, testMet)
			if err != nil {
				t.Error(err)
				return
			}
			msgs, _ := c.ProcessMessage(message.New(tt.arg))

			if len(msgs) != 1 {
				t.Fatalf("Test '%v' did not succeed", tt.name)
			}

			if exp, act := tt.output, string(message.GetAllBytes(msgs[0])[0]); exp != act {
				t.Errorf("Wrong result '%v': %v != %v", tt.name, act, exp)
			}
			msgs[0].Iter(func(i int, part types.Part) error {
				act := part.Metadata().Get(FailFlagKey)
				if len(act) > 0 && act != tt.err {
					t.Errorf("Wrong error message '%v': %v != %v", tt.name, act, tt.err)
				}
				return nil
			})
		})
	}
}

func TestJSONSchemaPathNotExist(t *testing.T) {
	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	testMet := metrics.DudType{}

	conf := NewConfig()
	conf.Type = "jsonschema"
	conf.JSONSchema.SchemaPath = fmt.Sprintf("file://path_does_not_exist")

	_, err := NewJSONSchema(conf, nil, testLog, testMet)
	if err == nil {
		t.Error("expected error from loading non existant schema file")
	}
}

func TestJSONSchemaInvalidSchema(t *testing.T) {
	schema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "any"
	}`

	tmpSchemaFile, err := ioutil.TempFile("", "benthos_jsonschema_invalid_schema_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpSchemaFile.Name())

	// write schema definition to tmpfile
	if _, err := tmpSchemaFile.Write([]byte(schema)); err != nil {
		t.Fatal(err)
	}

	testLog := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	testMet := metrics.DudType{}

	conf := NewConfig()
	conf.Type = "jsonschema"
	conf.JSONSchema.SchemaPath = fmt.Sprintf("file://%s", tmpSchemaFile.Name())

	_, err = NewJSONSchema(conf, nil, testLog, testMet)
	if err == nil {
		t.Error("expected error from loading bad schema")
	}
}
