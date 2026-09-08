/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package dotenv_test

import (
	"reflect"
	"testing"

	"github.com/envaar/vaar/internal/source/dotenv"
)

func TestDocumentUsesExplicitSourceOwnedFields(t *testing.T) {
	typeOfDocument := reflect.TypeOf(dotenv.Document{})
	for i := 0; i < typeOfDocument.NumField(); i++ {
		field := typeOfDocument.Field(i)
		if field.Anonymous {
			t.Fatalf("Document anonymously embeds %s; source fields must be explicit", field.Type)
		}
	}
}

func TestParseIsOwnedByDotenvSource(t *testing.T) {
	document, err := dotenv.Parse(".env", []byte("KEY=value\n"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if document.Path != ".env" {
		t.Fatalf("Path = %q, want %q", document.Path, ".env")
	}
	if len(document.Lines) != 1 || document.Lines[0].Key != "KEY" {
		t.Fatalf("Lines = %#v, want one KEY line", document.Lines)
	}
}
