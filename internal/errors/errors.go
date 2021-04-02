// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package errors defines the error handling used by kpt codebase.
package errors

import (
	"fmt"
	"strings"

	goerrors "errors"

	"github.com/GoogleContainerTools/kpt/internal/types"
)

// Error is the type that implements error interface used in the kpt codebase.
// It is based on the design in https://commandcenter.blogspot.com/2017/12/error-handling-in-upspin.html
// The intent is to capture error information in a structured format so
// that we can display it differently to different users for ex. kpt developers
// are interested in operational trace along with more diagnostic information while
// kpt end-users may be just interested in a concise and actionable information.
// Representing errors in structured format helps us decouple the error information
// from how it is surfaced to the end users.
type Error struct {
	// Path is the path of the object (pkg, file) involved in kpt operation.
	Path types.UniquePath

	// Op is the operation being performed, for ex. pkg.get, fn.render
	Op Op

	// Fn is the kpt function being run either as part of "fn render" or "fn eval"
	Fn Fn

	// Class refers to class of errors
	Class Class

	// Err refers to wrapped error (if any)
	Err error
}

func (e *Error) Error() string {
	b := new(strings.Builder)

	if e.Op != "" {
		pad(b, ": ")
		b.WriteString(string(e.Op))
	}

	if e.Path != "" {
		pad(b, ": ")
		b.WriteString("pkg ")
		b.WriteString(e.Path.RelativePath())
	}

	if e.Fn != "" {
		pad(b, ": ")
		b.WriteString("fn ")
		b.WriteString(string(e.Fn))
	}

	if e.Class != 0 {
		pad(b, ": ")
		b.WriteString(e.Class.String())
	}

	if e.Err != nil {
		var wrappedErr *Error
		if As(e.Err, &wrappedErr) {
			if !wrappedErr.Zero() {
				pad(b, ":\n\t")
				b.WriteString(wrappedErr.Error())
			}
		} else {
			pad(b, ": ")
			b.WriteString(e.Err.Error())
		}
	}
	if b.Len() == 0 {
		return "no error"
	}
	return b.String()
}

// pad appends given str to the string buffer.
func pad(b *strings.Builder, str string) {
	if b.Len() == 0 {
		return
	}
	b.WriteString(str)
}

func (e *Error) Zero() bool {
	return e.Op == "" && e.Path == "" && e.Fn == "" && e.Class == 0 && e.Err == nil
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Op describes the operation being performed.
type Op string

// Fn describes the kpt function associated with the error.
type Fn string

// Class describes the class of errors encountered.
type Class int

const (
	Other        Class = iota // Unclassified. Will not be printed.
	Exist                     // Item already exists.
	Internal                  // Internal error.
	InvalidParam              // Value is not valid.
	MissingParam              // Required value is missing or empty.
	Git                       // Errors from Git
)

func (c Class) String() string {
	switch c {
	case Other:
		return "other error"
	case Exist:
		return "item already exist"
	case Internal:
		return "internal error"
	case InvalidParam:
		return "invalid parameter value"
	case MissingParam:
		return "missing parameter value"
	case Git:
		return "git error"
	}
	return "unknown kind"
}

func E(args ...interface{}) error {
	if len(args) == 0 {
		panic("errors.E must have at least one argument")
	}

	e := &Error{}
	for _, arg := range args {
		switch a := arg.(type) {
		case types.UniquePath:
			e.Path = a
		case Op:
			e.Op = a
		case Fn:
			e.Fn = a
		case Class:
			e.Class = a
		case *Error:
			cp := *a
			e.Err = &cp
		case error:
			e.Err = a
		case string:
			e.Err = fmt.Errorf(a)
		default:
			panic(fmt.Errorf("unknown type %T for value %v in call to error.E", a, a))
		}
	}
	wrappedErr, ok := e.Err.(*Error)
	if !ok {
		return e
	}
	if e.Path == wrappedErr.Path {
		wrappedErr.Path = ""
	}

	if e.Op == wrappedErr.Op {
		wrappedErr.Op = ""
	}

	if e.Fn == wrappedErr.Fn {
		wrappedErr.Fn = ""
	}

	if e.Class == wrappedErr.Class {
		wrappedErr.Class = 0
	}
	return e
}

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool {
	return goerrors.Is(err, target)
}

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true. Otherwise, it returns false.
func As(err error, target interface{}) bool {
	return goerrors.As(err, target)
}
