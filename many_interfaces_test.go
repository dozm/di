package di

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/dozm/di/reflectx"
	"github.com/stretchr/testify/require"
)

type (
	IHandler interface {
		GetPath() string
	}

	IDepartment interface {
		GetName() string
	}
	IDepartment2 interface {
		IDepartment
		GetSecretName() string
	}
	ICompany interface {
		GetName() string
		GetDepartment() IDepartment
	}
	IEmployee interface {
		GetName() string
	}
	handler struct {
		path string
	}
	department struct {
		Name       string
		SecretName string
		Time       ITime
	}
	company struct {
		Name       string
		Department IDepartment
	}
	ITime interface {
		Now() time.Time
	}
	myTime struct {
		fixedTime time.Time
	}
	IScopedName interface {
		SetName(name string)
		GetName() string
	}
	scopedName struct {
		name string
	}
)

func (s *scopedName) SetName(name string) {
	s.name = name
}
func (s *scopedName) GetName() string {
	return s.name
}
func (s *handler) GetPath() string { return s.path }
func (s *myTime) Now() time.Time {
	if !s.fixedTime.IsZero() {
		return s.fixedTime
	}
	return time.Now()

}
func AddSingletonTime(b ContainerBuilder) {
	AddSingleton[ITime](b, func() ITime {
		return &myTime{}
	})
}
func AddScopedHandlers(b ContainerBuilder) {
	AddScopedWithLookupKeys[*handler](b,
		func() *handler {
			return &handler{path: "1"}
		}, []string{"1"},
		reflect.TypeOf((*IHandler)(nil)))
	AddScopedWithLookupKeys[*handler](b,
		func() *handler {
			return &handler{path: "2"}
		}, []string{"2"},
		reflect.TypeOf((*IHandler)(nil)))
}

func AddSingletonDepartments(b ContainerBuilder, names ...string) {
	// pointer to interface type
	typeIDepartment := reflect.TypeOf((*IDepartment)(nil))
	// elem of pointer to interface type
	typeIDepartment2 := reflectx.TypeOf[IDepartment2]()

	for idx := range names {
		name := names[idx]
		secretName := fmt.Sprintf("%s-FBI", name)
		fmt.Println("registering department:", name, " secretname:", secretName)
		AddSingleton[*department](b, func(tt ITime) *department {
			return &department{
				Name:       name,
				Time:       tt,
				SecretName: secretName,
			}
		}, typeIDepartment, typeIDepartment2)
	}
}
func AddSingletonCompany(b ContainerBuilder) {
	AddSingleton[ICompany](b, func(department IDepartment) *company {
		return &company{
			Name:       "Contoso",
			Department: department,
		}
	})
}
func (s *department) GetName() string       { return s.Name }
func (s *department) GetSecretName() string { return s.SecretName }

func (s *company) GetName() string            { return s.Name }
func (s *company) GetDepartment() IDepartment { return s.Department }

type (
	UniqueDepartment[TUnique any] struct {
		Name       string
		SecretName string
		Time       ITime
	}
	DepartmentIT struct{}
	DepartmentHR struct{}
)

func (d *UniqueDepartment[TUnique]) GetName() string {
	return d.Name
}

func (s *UniqueDepartment[TUnique]) GetSecretName() string { return s.SecretName }
func AddUniqueDepartment[TUnique any](b ContainerBuilder, name string) {
	AddSingleton[*UniqueDepartment[TUnique]](b, func(tt ITime) *UniqueDepartment[TUnique] {
		return &UniqueDepartment[TUnique]{
			Name:       name,
			Time:       tt,
			SecretName: fmt.Sprintf("%s-FBI", name),
		}
	})
	AddSingleton[IDepartment](b, func(d *UniqueDepartment[TUnique]) IDepartment {
		return d
	})
	AddSingleton[IDepartment2](b, func(d *UniqueDepartment[TUnique]) IDepartment2 {
		return d
	})
}
func TestUniquenesWithMany(t *testing.T) {
	b := Builder()
	AddSingletonTime(b)
	AddUniqueDepartment[DepartmentIT](b, "IT")
	AddUniqueDepartment[DepartmentHR](b, "HR")
	c := b.Build()
	department := Get[IDepartment](c)
	require.Equal(t, "HR", department.GetName())
	department2 := Get[IDepartment2](c)
	require.Equal(t, "HR", department2.GetName())
	require.Equal(t, "HR-FBI", department2.GetSecretName())

	departments := Get[[]IDepartment](c)
	require.Equal(t, 2, len(departments))
	require.Equal(t, "IT", departments[0].GetName())
	require.Equal(t, "HR", departments[1].GetName())
	department2s := Get[[]IDepartment2](c)
	require.Equal(t, 2, len(department2s))
	require.Equal(t, "IT", department2s[0].GetName())
	require.Equal(t, "HR", department2s[1].GetName())
	require.Equal(t, "IT-FBI", department2s[0].GetSecretName())
	require.Equal(t, "HR-FBI", department2s[1].GetSecretName())

}
func TestSingletonOneInterface(t *testing.T) {
	b := Builder()
	// get the reflet.Type of the interface
	it := reflect.TypeOf((*ITime)(nil))

	AddSingleton[*myTime](b, func() *myTime {
		return &myTime{
			fixedTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		}
	}, it)
	AddSingleton[ITime](b, func() *myTime {
		return &myTime{
			fixedTime: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		}
	})
	// Build the container
	c := b.Build()
	tt := Get[ITime](c)
	now := tt.Now()
	require.Equal(t, time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), now)

	tts := Get[[]ITime](c)
	require.Equal(t, 2, len(tts))
	for _, t := range tts {
		fmt.Println(t.Now())
	}
	require.Equal(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), tts[0].Now())
	require.Equal(t, time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), tts[1].Now())
}
func TestManyDepartments(t *testing.T) {
	b := Builder()
	AddSingletonTime(b)
	AddSingletonDepartments(b, "IT", "HR")
	// Build the container
	c := b.Build()

	// many departments are registered, so it should return the last one
	department := Get[IDepartment](c)
	require.Equal(t, "HR", department.GetName())
	department2 := Get[IDepartment2](c)
	require.Equal(t, "HR", department2.GetName())
	require.Equal(t, "HR-FBI", department2.GetSecretName())

	// get all the departments
	departments := Get[[]IDepartment](c)
	require.Equal(t, 2, len(departments))
	departments2 := Get[[]IDepartment2](c)
	require.Equal(t, 2, len(departments2))

	// the order must be as registered in the container [IT, HR]
	// both are actually HR
	for _, d := range departments {
		fmt.Println(d.GetName())
	}
	for _, d := range departments2 {
		fmt.Println(d.GetName())
		fmt.Println(d.GetSecretName())
	}
	require.Equal(t, "IT", departments[0].GetName())
	require.Equal(t, "HR", departments[1].GetName())
}
func TestSingleton(t *testing.T) {
	b := Builder()
	AddSingletonTime(b)
	AddSingletonDepartments(b, "IT")
	AddSingletonCompany(b)

	// Build the container
	c := b.Build()

	// IEmployee is not registered, so it should return an error
	employee, err := TryGet[IEmployee](c)
	require.Nil(t, employee)
	require.Error(t, err)

	// only one department is registered
	department := Get[IDepartment](c)
	require.Equal(t, "IT", department.GetName())

	department2 := Get[IDepartment2](c)
	require.Equal(t, "IT", department2.GetName())
	require.Equal(t, "IT-FBI", department2.GetSecretName())

	// only one company is registered
	company := Get[ICompany](c)
	require.Equal(t, "IT", company.GetDepartment().GetName())
	require.Equal(t, "Contoso", company.GetName())
}
func TestManyWithScopeWithLookupKeys(t *testing.T) {
	b := Builder()
	// Build the container
	AddScopedHandlers(b)
	c := b.Build()
	scopeFactory := Get[ScopeFactory](c)
	scope1 := scopeFactory.CreateScope()
	handlers := Get[[]IHandler](scope1.Container())
	require.Equal(t, 2, len(handlers))
	require.NotPanics(t, func() {
		h := GetByLookupKey[IHandler](c, "1")
		require.NotNil(t, h)
		require.Equal(t, "1", h.GetPath())
	})
}
func TestManyWithScope(t *testing.T) {
	b := Builder()
	AddSingletonTime(b)
	AddSingletonDepartments(b, "IT", "HR")
	AddScoped[IScopedName](b, func() IScopedName {
		return &scopedName{
			name: "ScopedNameOne",
		}
	})
	AddScoped[IScopedName](b, func() IScopedName {
		return &scopedName{
			name: "ScopedNameTwo",
		}
	})
	// Build the container
	c := b.Build()

	scopeFactory := Get[ScopeFactory](c)
	scope1 := scopeFactory.CreateScope()
	scope2 := scopeFactory.CreateScope()

	// get all the departments
	department1 := Get[IDepartment2](scope1.Container())
	department2 := Get[IDepartment2](scope2.Container())

	require.Equal(t, department1, department2)

	scopeName1 := Get[IScopedName](scope1.Container())
	scopeName2 := Get[IScopedName](scope2.Container())
	require.NotSame(t, scopeName1, scopeName2)

	// get all the IScopedName(s)
	scopeNames1 := Get[[]IScopedName](scope1.Container())
	require.Equal(t, 2, len(scopeNames1))
	require.NotSame(t, scopeNames1[0], scopeNames1[1])
	scopeNames2 := Get[[]IScopedName](scope2.Container())
	require.Equal(t, 2, len(scopeNames2))
	require.NotSame(t, scopeNames2[0], scopeNames2[1])

	for i := 0; i < len(scopeNames1); i++ {
		require.NotSame(t, scopeNames1[i], scopeNames2[i])
	}
}
