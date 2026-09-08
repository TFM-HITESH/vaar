// Copyright © 2026 envaar
// SPDX-License-Identifier: Apache-2.0

package dotenv_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/envaar/vaar/internal/analysis"
	analysisdotenv "github.com/envaar/vaar/internal/analysis/dotenv"
	sourcedotenv "github.com/envaar/vaar/internal/source/dotenv"
)

func TestFromDocumentPreservesIdentityPathsAndMetadata(t *testing.T) {
	source := sourcedotenv.Document{
		Path:             "config/.env.example",
		BOM:              true,
		MixedLineEndings: true,
		EndsWithNewline:  true,
		Lines: []sourcedotenv.Line{{
			Number:        1,
			Key:           "PORT",
			HasKey:        true,
			HasValue:      true,
			HasAssignment: true,
		}},
		SourcePath: "/workspace/config/.env.example",
	}

	got := analysisdotenv.FromDocument(analysisdotenv.DocumentInput{
		ID:     analysis.DocumentID("dotenv:example"),
		Source: source,
	})

	if got.ID != "dotenv:example" {
		t.Errorf("ID = %q, want %q", got.ID, "dotenv:example")
	}
	if got.SourcePath != source.SourcePath {
		t.Errorf("SourcePath = %q, want %q", got.SourcePath, source.SourcePath)
	}
	if got.DisplayPath != source.Path {
		t.Errorf("DisplayPath = %q, want %q", got.DisplayPath, source.Path)
	}
	if !got.BOM || !got.MixedLineEndings || !got.EndsWithNewline {
		t.Fatalf("file metadata = %#v, want all flags true", got)
	}
	if len(got.Lines) != 1 || got.Lines[0].Key != "PORT" {
		t.Fatalf("Lines = %#v, want one PORT line", got.Lines)
	}
}

func TestFromDocumentsPreservesOrderAndIdentity(t *testing.T) {
	inputs := []analysisdotenv.DocumentInput{
		{
			ID: analysis.DocumentID("second"),
			Source: sourcedotenv.Document{
				Path:       "second.env",
				SourcePath: "/sources/second.env",
			},
		},
		{
			ID: analysis.DocumentID("first"),
			Source: sourcedotenv.Document{
				Path:       "first.env",
				SourcePath: "/sources/first.env",
			},
		},
	}

	got := analysisdotenv.FromDocuments(inputs)

	if len(got) != len(inputs) {
		t.Fatalf("len(FromDocuments()) = %d, want %d", len(got), len(inputs))
	}
	for i, want := range inputs {
		if got[i].ID != want.ID {
			t.Errorf("document %d ID = %q, want %q", i, got[i].ID, want.ID)
		}
		if got[i].SourcePath != want.Source.SourcePath || got[i].DisplayPath != want.Source.Path {
			t.Errorf("document %d paths = (%q, %q), want (%q, %q)", i, got[i].SourcePath, got[i].DisplayPath, want.Source.SourcePath, want.Source.Path)
		}
	}
}

func TestFromDocumentPreservesEmptyDocument(t *testing.T) {
	got := analysisdotenv.FromDocument(analysisdotenv.DocumentInput{
		ID: analysis.DocumentID("empty"),
		Source: sourcedotenv.Document{
			Path: "empty.env",
		},
	})

	if got.ID != "empty" || got.DisplayPath != "empty.env" {
		t.Fatalf("empty document metadata = %#v", got)
	}
	if got.Lines == nil {
		t.Fatal("empty document Lines is nil, want non-nil empty slice")
	}
	if len(got.Lines) != 0 {
		t.Fatalf("len(empty document Lines) = %d, want 0", len(got.Lines))
	}
}

func TestFromDocumentPreservesLineOrderNumbersAndFacts(t *testing.T) {
	source := sourcedotenv.Document{
		Path: "facts.env",
		Lines: []sourcedotenv.Line{
			{
				Number:               1,
				Key:                  "FIRST",
				HasKey:               true,
				HasValue:             true,
				HasAssignment:        true,
				QuoteState:           sourcedotenv.QuoteDouble,
				CommentState:         sourcedotenv.CommentInline,
				DelimiterState:       sourcedotenv.DelimiterEquals,
				LineEnding:           sourcedotenv.LineEndingCRLF,
				SpaceBeforeDelimiter: true,
				SpaceAfterDelimiter:  true,
			},
			{
				Number:         2,
				IsComment:      true,
				CommentState:   sourcedotenv.CommentFull,
				DelimiterState: sourcedotenv.DelimiterNone,
				LineEnding:     sourcedotenv.LineEndingLF,
			},
			{
				Number:         7,
				Key:            "BARE",
				HasKey:         true,
				DelimiterState: sourcedotenv.DelimiterMissing,
				QuoteState:     sourcedotenv.QuoteUnbalanced,
				LineEnding:     sourcedotenv.LineEndingNone,
			},
		},
		SourcePath: "/sources/facts.env",
	}

	got := analysisdotenv.FromDocument(analysisdotenv.DocumentInput{
		ID:     "facts",
		Source: source,
	})

	want := []analysis.Line{
		{
			Number:               1,
			Key:                  "FIRST",
			HasKey:               true,
			HasValue:             true,
			HasAssignment:        true,
			QuoteState:           analysis.QuoteDouble,
			CommentState:         analysis.CommentInline,
			DelimiterState:       analysis.DelimiterEquals,
			LineEnding:           analysis.LineEndingCRLF,
			SpaceBeforeDelimiter: true,
			SpaceAfterDelimiter:  true,
		},
		{
			Number:         2,
			IsComment:      true,
			CommentState:   analysis.CommentFull,
			DelimiterState: analysis.DelimiterNone,
			LineEnding:     analysis.LineEndingLF,
		},
		{
			Number:         7,
			Key:            "BARE",
			HasKey:         true,
			DelimiterState: analysis.DelimiterMissing,
			QuoteState:     analysis.QuoteUnbalanced,
			LineEnding:     analysis.LineEndingNone,
		},
	}

	if !reflect.DeepEqual(got.Lines, want) {
		t.Fatalf("Lines = %#v, want %#v", got.Lines, want)
	}
}

func TestFromDocumentDerivesOnlySafeWhitespaceFacts(t *testing.T) {
	source := sourcedotenv.Document{
		Path: "whitespace.env",
		Lines: []sourcedotenv.Line{
			{
				Number:             1,
				LeadingWhitespace:  " \t",
				TrailingWhitespace: "\t",
				Value:              "semantic\tvalue",
			},
			{
				Number: 2,
				Value:  "semantic\nvalue",
			},
		},
	}

	got := analysisdotenv.FromDocument(analysisdotenv.DocumentInput{Source: source})

	if !got.Lines[0].HasLeadingWhitespace || !got.Lines[0].HasTrailingWhitespace {
		t.Errorf("line 1 whitespace flags = %#v, want both true", got.Lines[0])
	}
	if !got.Lines[0].ValueContainsWhitespace {
		t.Errorf("line 1 ValueContainsWhitespace = false, want true")
	}
	if got.Lines[1].ValueContainsWhitespace {
		t.Errorf("line 2 ValueContainsWhitespace = true, want false for newline-only whitespace")
	}
}

func TestFromDocumentPreservesEmptyAssignmentColonAndBlankFacts(t *testing.T) {
	source := sourcedotenv.Document{
		Path: "syntax.env",
		Lines: []sourcedotenv.Line{
			{
				Number:               3,
				Key:                  "EMPTY",
				HasKey:               true,
				HasValue:             false,
				HasAssignment:        true,
				DelimiterState:       sourcedotenv.DelimiterEquals,
				LineEnding:           sourcedotenv.LineEndingLF,
				LeadingWhitespace:    " ",
				TrailingWhitespace:   " ",
				SpaceBeforeDelimiter: true,
				SpaceAfterDelimiter:  true,
			},
			{
				Number:         4,
				IsBlank:        true,
				DelimiterState: sourcedotenv.DelimiterNone,
				LineEnding:     sourcedotenv.LineEndingCRLF,
			},
			{
				Number:         5,
				Key:            "COLON",
				HasKey:         true,
				HasValue:       true,
				QuoteState:     sourcedotenv.QuoteSingle,
				DelimiterState: sourcedotenv.DelimiterColon,
				LineEnding:     sourcedotenv.LineEndingNone,
			},
		},
	}

	got := analysisdotenv.FromDocument(analysisdotenv.DocumentInput{Source: source})

	if got.Lines[0].HasValue || !got.Lines[0].HasAssignment {
		t.Errorf("empty assignment facts = %#v, want HasValue false and HasAssignment true", got.Lines[0])
	}
	if !got.Lines[0].HasLeadingWhitespace || !got.Lines[0].HasTrailingWhitespace {
		t.Errorf("empty assignment whitespace facts = %#v, want both true", got.Lines[0])
	}
	if !got.Lines[1].IsBlank || got.Lines[1].LineEnding != analysis.LineEndingCRLF {
		t.Errorf("blank line facts = %#v, want blank CRLF line", got.Lines[1])
	}
	if got.Lines[2].DelimiterState != analysis.DelimiterColon || got.Lines[2].QuoteState != analysis.QuoteSingle {
		t.Errorf("colon line facts = %#v, want colon single-quoted line", got.Lines[2])
	}
}

func TestFromDocumentCopiesLineStorage(t *testing.T) {
	source := sourcedotenv.Document{
		Path:  "mutable.env",
		Lines: []sourcedotenv.Line{{Number: 1, Key: "ORIGINAL"}},
	}

	got := analysisdotenv.FromDocument(analysisdotenv.DocumentInput{Source: source})
	source.Lines[0].Key = "MUTATED"
	source.Lines = append(source.Lines, sourcedotenv.Line{Number: 2, Key: "APPENDED"})

	if len(got.Lines) != 1 || got.Lines[0].Key != "ORIGINAL" {
		t.Fatalf("converted Lines = %#v, want independent original line slice", got.Lines)
	}
}

func TestFromDocumentDoesNotExposeSourceValues(t *testing.T) {
	secret := "sentinel-value-from-source-85"
	source := sourcedotenv.Document{
		Path:     "safe.env",
		Original: []byte(secret),
		Lines: []sourcedotenv.Line{{
			Number:   1,
			Key:      "SECRET_KEY",
			Raw:      secret,
			Content:  secret,
			Value:    secret,
			ValueRaw: secret,
		}},
	}

	got := analysisdotenv.FromDocument(analysisdotenv.DocumentInput{Source: source})
	if strings.Contains(fmt.Sprintf("%#v", got), secret) {
		t.Fatalf("analysis document contains source sentinel: %#v", got)
	}
}

func TestFromDocumentsNilReturnsNonNilEmptySlice(t *testing.T) {
	got := analysisdotenv.FromDocuments(nil)

	if got == nil {
		t.Fatal("FromDocuments(nil) returned nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("len(FromDocuments(nil)) = %d, want 0", len(got))
	}
}
