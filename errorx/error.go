package errorx

import (
	"fmt"
	"reflect"
	"strings"
)

type ArgumentNilError struct {
	Name string
}

func (e *ArgumentNilError) Error() string {
	return fmt.Sprintf("ArgumentNilError: %v", e.Name)
}

func NewArgumentNilError(name string) *ArgumentNilError {
	return &ArgumentNilError{name}
}

type ArgumentError struct {
	Message string
}

func (e *ArgumentError) Error() string {
	return fmt.Sprintf("ArgumentError: %v", e.Message)
}

func NewArgumentError(message string) *ArgumentError {
	return &ArgumentError{message}
}

type CircularDependencyError struct {
	Message string
}

func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("CircularDependencyError: %v", e.Message)
}

type FuncSignatureError struct {
	Message string
}

func (e *FuncSignatureError) Error() string {
	return fmt.Sprintf("FuncSignatureError: %v", e.Message)
}

type ServiceNotFound struct {
	ServiceType reflect.Type
}

func (e *ServiceNotFound) Error() string {
	return fmt.Sprintf("ServiceNotFound '%v'", e.ServiceType)
}

type InvalidDescriptor struct {
	ServiceType reflect.Type
}

func (e *InvalidDescriptor) Error() string {
	return fmt.Sprintf("InvalidDescriptor '%v'", e.ServiceType)
}

type TypeIncompatibilityError struct {
	To   reflect.Type
	From reflect.Type
}

func (e *TypeIncompatibilityError) Error() string {
	return fmt.Sprintf("the value of type '%v' can not assignable to type '%v'", e.From, e.To)
}

type ObjectDisposedError struct {
	Message string
}

func (e *ObjectDisposedError) Error() string {
	return fmt.Sprintf("ObjectDisposedError: %v", e.Message)
}

type ScopedServiceFromRootError struct {
	Message string
}

func (e *ScopedServiceFromRootError) Error() string {
	return fmt.Sprintf("ScopedServiceFromRootError: %v", e.Message)
}

type AggregateError struct {
	Errors []error
}

func (e *AggregateError) Add(err error) {
	e.Errors = append(e.Errors, err)
}

func (e *AggregateError) Error() string {
	var b strings.Builder
	b.WriteString("AggregateError: \n")
	for _, e := range e.Errors {
		b.WriteString(e.Error())
		b.WriteString("\n")
	}
	return b.String()
}
