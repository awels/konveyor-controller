package model

import (
	"errors"
	"fmt"
	"github.com/konveyor/controller/pkg/ref"
	"github.com/onsi/gomega"
	"math"
	"testing"
	"time"
)

type TestEncoded struct {
	Name string
}

type TestBase struct {
	Parent int    `sql:""`
	Phone  string `sql:""`
}

type PlainObject struct {
	ID   int    `sql:"pk"`
	Name string `sql:""`
	Age  int    `sql:""`
}

func (m *PlainObject) Pk() string {
	return fmt.Sprintf("%d", m.ID)
}

func (m *PlainObject) String() string {
	return fmt.Sprintf(
		"PlainObject: id: %d, name:%s",
		m.ID,
		m.Name)
}

func (m *PlainObject) Equals(other Model) bool {
	return false
}

func (m *PlainObject) Labels() Labels {
	return nil
}

type TestObject struct {
	TestBase
	RowID  int64          `sql:"virtual"`
	PK     string         `sql:"pk(id)"`
	ID     int            `sql:"key"`
	Rev    int            `sql:"incremented"`
	Name   string         `sql:"index(a)"`
	Age    int            `sql:"index(a)"`
	Int8   int8           `sql:""`
	Int16  int16          `sql:""`
	Int32  int32          `sql:""`
	Bool   bool           `sql:""`
	Object TestEncoded    `sql:""`
	Slice  []string       `sql:""`
	Map    map[string]int `sql:""`
	D1     string         `sql:"d1"`
	D2     string         `sql:"d2"`
	D3     string         `sql:"d3"`
	D4     string         `sql:"d4"`
	labels Labels
}

func (m *TestObject) Pk() string {
	return fmt.Sprintf("%s", m.PK)
}

func (m *TestObject) String() string {
	return fmt.Sprintf(
		"TestObject: id: %d, name:%s",
		m.ID,
		m.Name)
}

func (m *TestObject) Labels() Labels {
	return m.labels
}

// received event.
type TestEvent struct {
	action uint8
	model  *TestObject
}

type TestHandler struct {
	options WatchOptions
	name    string
	started bool
	parity  bool
	all     []TestEvent
	created []int
	updated []int
	deleted []int
	err     []error
	done    bool
}

func (w *TestHandler) Options() WatchOptions {
	return w.options
}

func (w *TestHandler) Started(uint64) {
	w.started = true
}

func (w *TestHandler) Parity() {
	w.parity = true
}

func (w *TestHandler) Created(e Event) {
	if object, cast := e.Model.(*TestObject); cast {
		w.all = append(w.all, TestEvent{action: e.Action, model: object})
		w.created = append(w.created, object.ID)
	}
}

func (w *TestHandler) Updated(e Event) {
	if object, cast := e.Model.(*TestObject); cast {
		w.all = append(w.all, TestEvent{action: e.Action, model: object})
		w.updated = append(w.updated, object.ID)
	}
}
func (w *TestHandler) Deleted(e Event) {
	if object, cast := e.Model.(*TestObject); cast {
		w.all = append(w.all, TestEvent{action: e.Action, model: object})
		w.deleted = append(w.deleted, object.ID)
	}
}

func (w *TestHandler) Error(err error) {
	w.err = append(w.err, err)
}

func (w *TestHandler) End() {
	w.done = true
}

type MutatingHandler struct {
	options WatchOptions
	DB
	name    string
	started bool
	parity  bool
	created []int
	updated []int
}

func (w *MutatingHandler) Options() WatchOptions {
	return w.options
}

func (w *MutatingHandler) Started(uint64) {
	w.started = true
}

func (w *MutatingHandler) Parity() {
	w.parity = true
}

func (w *MutatingHandler) Created(e Event) {
	tx, _ := w.DB.Begin()
	tx.Get(e.Model)
	e.Model.(*TestObject).Age++
	tx.Update(e.Model)
	tx.Commit()
	w.created = append(w.created, e.Model.(*TestObject).ID)
}

func (w *MutatingHandler) Updated(e Event) {
	tx, _ := w.DB.Begin()
	tx.Get(e.Model)
	e.Model.(*TestObject).Age++
	tx.Update(e.Model)
	tx.Commit()
	w.updated = append(w.updated, e.Model.(*TestObject).ID)
}

func (w *MutatingHandler) Deleted(e Event) {
}

func (w *MutatingHandler) Error(err error) {
	return
}

func (w *MutatingHandler) End() {
}

func TestCRUD(t *testing.T) {
	var err error
	g := gomega.NewGomegaWithT(t)
	DB := New(
		"/tmp/test-crud.db",
		&Label{},
		&PlainObject{},
		&TestObject{})
	err = DB.Open(true)
	g.Expect(err).To(gomega.BeNil())

	plainA := &PlainObject{
		ID:   18,
		Name: "Ashley",
		Age:  17,
	}
	err = DB.Insert(plainA)
	g.Expect(err).To(gomega.BeNil())
	plainB := &PlainObject{ID: 18}
	err = DB.Get(plainB)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(plainA.Pk()).To(gomega.Equal(plainB.Pk()))
	g.Expect(plainA.ID).To(gomega.Equal(plainB.ID))
	g.Expect(plainA.Name).To(gomega.Equal(plainB.Name))
	g.Expect(plainA.Age).To(gomega.Equal(plainB.Age))

	objA := &TestObject{
		TestBase: TestBase{
			Parent: 0,
			Phone:  "1234",
		},
		ID:     0,
		Name:   "Elmer",
		Age:    18,
		Int8:   8,
		Int16:  16,
		Int32:  32,
		Bool:   true,
		Object: TestEncoded{Name: "json"},
		Slice:  []string{"hello", "world"},
		Map:    map[string]int{"A": 1, "B": 2},
		labels: Labels{
			"n1": "v1",
			"n2": "v2",
		},
	}
	assertEqual := func(a, b *TestObject) {
		g.Expect(a.PK).To(gomega.Equal(b.PK))
		g.Expect(a.ID).To(gomega.Equal(b.ID))
		g.Expect(a.Rev).To(gomega.Equal(b.Rev))
		g.Expect(a.Name).To(gomega.Equal(b.Name))
		g.Expect(a.Age).To(gomega.Equal(b.Age))
		g.Expect(a.Int8).To(gomega.Equal(b.Int8))
		g.Expect(a.Int16).To(gomega.Equal(b.Int16))
		g.Expect(a.Int32).To(gomega.Equal(b.Int32))
		g.Expect(a.Bool).To(gomega.Equal(b.Bool))
		g.Expect(a.Object).To(gomega.Equal(b.Object))
		g.Expect(a.Slice).To(gomega.Equal(b.Slice))
		g.Expect(a.Map).To(gomega.Equal(b.Map))
		for k, v := range objA.labels {
			l := &Label{
				Kind:   ref.ToKind(a),
				Parent: a.PK,
				Name:   k,
			}
			g.Expect(DB.Get(l)).To(gomega.BeNil())
			g.Expect(v).To(gomega.Equal(l.Value))
		}
	}
	// Insert
	err = DB.Insert(objA)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(objA.Rev).To(gomega.Equal(1))
	objB := &TestObject{ID: objA.ID}
	// Get
	err = DB.Get(objB)
	g.Expect(err).To(gomega.BeNil())
	assertEqual(objA, objB)
	// Update
	objA.Name = "Larry"
	objA.Age = 21
	objA.Bool = false
	err = DB.Update(objA)
	g.Expect(err).To(gomega.BeNil())
	// Get
	objB = &TestObject{ID: objA.ID}
	err = DB.Get(objB)
	g.Expect(err).To(gomega.BeNil())
	assertEqual(objA, objB)
	// Delete
	objA = &TestObject{ID: objA.ID}
	err = DB.Delete(objA)
	g.Expect(err).To(gomega.BeNil())
	// Get (not found)
	objB = &TestObject{ID: objA.ID}
	err = DB.Get(objB)
	g.Expect(errors.Is(err, NotFound)).To(gomega.BeTrue())
}

func TestTransactions(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	DB := New(
		"/tmp/test-transactions.db",
		&TestObject{})
	err := DB.Open(true)
	g.Expect(err).To(gomega.BeNil())
	for i := 0; i < 10; i++ {
		// Begin
		tx, err := DB.Begin()
		defer tx.End()
		g.Expect(err).To(gomega.BeNil())
		object := &TestObject{
			ID:   i,
			Name: "Elmer",
		}
		err = tx.Insert(object)
		g.Expect(err).To(gomega.BeNil())
		// Get (not found)
		object = &TestObject{ID: object.ID}
		err = DB.Get(object)
		g.Expect(errors.Is(err, NotFound)).To(gomega.BeTrue())
		tx.Commit()
		// Get (found)
		object = &TestObject{ID: object.ID}
		err = DB.Get(object)
		g.Expect(err).To(gomega.BeNil())
	}
}

func TestList(t *testing.T) {
	var err error
	g := gomega.NewGomegaWithT(t)
	DB := New(
		"/tmp/test-list.db",
		&TestObject{})
	err = DB.Open(true)
	g.Expect(err).To(gomega.BeNil())
	N := 10
	for i := 0; i < N; i++ {
		object := &TestObject{
			ID:     i,
			Name:   "Elmer",
			Age:    18,
			Int8:   8,
			Int16:  16,
			Int32:  32,
			Bool:   true,
			Object: TestEncoded{Name: "json"},
			Slice:  []string{"hello", "world"},
			Map:    map[string]int{"A": 1, "B": 2},
			D1:     "d-1",
			D2:     "d-2",
			D3:     "d-3",
			D4:     "d-4",
			labels: Labels{
				"id": fmt.Sprintf("v%d", i),
			},
		}
		err = DB.Insert(object)
		g.Expect(err).To(gomega.BeNil())
	}
	// List all; detail level=0
	list := []TestObject{}
	err = DB.List(&list, ListOptions{})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(10))
	g.Expect(list[0].Name).To(gomega.Equal(""))
	g.Expect(list[0].Slice).To(gomega.BeNil())
	g.Expect(list[0].D1).To(gomega.Equal(""))
	g.Expect(list[0].D2).To(gomega.Equal(""))
	g.Expect(list[0].D3).To(gomega.Equal(""))
	g.Expect(list[0].D4).To(gomega.Equal(""))
	// List detail level=1
	list = []TestObject{}
	err = DB.List(&list, ListOptions{Detail: 1})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(10))
	g.Expect(list[0].Name).To(gomega.Equal("Elmer"))
	g.Expect(len(list[0].Slice)).To(gomega.Equal(2))
	g.Expect(list[0].D1).To(gomega.Equal("d-1"))
	g.Expect(list[0].D2).To(gomega.Equal(""))
	g.Expect(list[0].D3).To(gomega.Equal(""))
	g.Expect(list[0].D4).To(gomega.Equal(""))
	// List detail level=2
	list = []TestObject{}
	err = DB.List(&list, ListOptions{Detail: 2})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(10))
	g.Expect(list[0].Name).To(gomega.Equal("Elmer"))
	g.Expect(len(list[0].Slice)).To(gomega.Equal(2))
	g.Expect(list[0].D1).To(gomega.Equal("d-1"))
	g.Expect(list[0].D2).To(gomega.Equal("d-2"))
	g.Expect(list[0].D3).To(gomega.Equal(""))
	g.Expect(list[0].D4).To(gomega.Equal(""))
	// List detail level=3
	list = []TestObject{}
	err = DB.List(&list, ListOptions{Detail: 3})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(10))
	g.Expect(list[0].Name).To(gomega.Equal("Elmer"))
	g.Expect(len(list[0].Slice)).To(gomega.Equal(2))
	g.Expect(list[0].D1).To(gomega.Equal("d-1"))
	g.Expect(list[0].D2).To(gomega.Equal("d-2"))
	g.Expect(list[0].D3).To(gomega.Equal("d-3"))
	g.Expect(list[0].D4).To(gomega.Equal(""))
	// List detail level=4
	list = []TestObject{}
	err = DB.List(&list, ListOptions{Detail: 4})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(10))
	g.Expect(list[0].Name).To(gomega.Equal("Elmer"))
	g.Expect(len(list[0].Slice)).To(gomega.Equal(2))
	g.Expect(list[0].D1).To(gomega.Equal("d-1"))
	g.Expect(list[0].D2).To(gomega.Equal("d-2"))
	g.Expect(list[0].D3).To(gomega.Equal("d-3"))
	g.Expect(list[0].D4).To(gomega.Equal("d-4"))
	// List = (single).
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Predicate: Eq("ID", 0),
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(1))
	g.Expect(list[0].ID).To(gomega.Equal(0))
	// List != AND
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Detail: 2,
			Predicate: And( // Even only.
				Neq("ID", 1),
				Neq("ID", 3),
				Neq("ID", 5),
				Neq("ID", 7),
				Neq("ID", 9)),
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(5))
	g.Expect(list[0].ID).To(gomega.Equal(0))
	g.Expect(list[1].ID).To(gomega.Equal(2))
	g.Expect(list[2].ID).To(gomega.Equal(4))
	g.Expect(list[3].ID).To(gomega.Equal(6))
	g.Expect(list[4].ID).To(gomega.Equal(8))
	// List OR =.
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Predicate: Or(
				Eq("ID", 0),
				Eq("ID", 6)),
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(2))
	g.Expect(list[0].ID).To(gomega.Equal(0))
	g.Expect(list[1].ID).To(gomega.Equal(6))
	// List < (lt).
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Predicate: Lt("ID", 2),
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(2))
	g.Expect(list[0].ID).To(gomega.Equal(0))
	g.Expect(list[1].ID).To(gomega.Equal(1))
	// List > (gt).
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Predicate: Gt("ID", 7),
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(2))
	g.Expect(list[0].ID).To(gomega.Equal(8))
	g.Expect(list[1].ID).To(gomega.Equal(9))
	// List > (gt) virtual.
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Predicate: Gt("RowID", N/2),
			Detail:    1,
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(N / 2))
	g.Expect(list[0].RowID).To(gomega.Equal(int64(N/2) + 1))
	// List (Eq) Field values.
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Predicate: Eq("RowID", Field{Name: "int8"}),
			Detail:    1,
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(1))
	g.Expect(list[0].RowID).To(gomega.Equal(int64(8)))
	// List (nEq) Field values.
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Predicate: Neq("RowID", Field{Name: "int8"}),
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(N - 1))
	// List (Lt) Field values.
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Predicate: Lt("int8", Field{Name: "int16"}),
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(N))
	// List (Gt) Field values.
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Predicate: Gt("RowID", Field{Name: "int8"}),
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(2))
	// By label.
	list = []TestObject{}
	err = DB.List(
		&list,
		ListOptions{
			Sort: []int{2},
			Predicate: Or(
				Match(Labels{"id": "v4"}),
				Eq("ID", 8)),
		})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(len(list)).To(gomega.Equal(2))
	g.Expect(list[0].ID).To(gomega.Equal(4))
	g.Expect(list[1].ID).To(gomega.Equal(8))
	// Test count all.
	count, err := DB.Count(&TestObject{}, nil)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(count).To(gomega.Equal(int64(10)))
	// Test count with predicate.
	count, err = DB.Count(&TestObject{}, Gt("ID", 0))
	g.Expect(err).To(gomega.BeNil())
	g.Expect(count).To(gomega.Equal(int64(9)))
}

func TestIter(t *testing.T) {
	var err error
	g := gomega.NewGomegaWithT(t)
	DB := New(
		"/tmp/test-iter.db",
		&TestObject{})
	err = DB.Open(true)
	g.Expect(err).To(gomega.BeNil())
	N := 10
	for i := 0; i < N; i++ {
		object := &TestObject{
			ID:     i,
			Name:   "Elmer",
			Age:    18,
			Int8:   8,
			Int16:  16,
			Int32:  32,
			Bool:   true,
			Object: TestEncoded{Name: "json"},
			Slice:  []string{"hello", "world"},
			Map:    map[string]int{"A": 1, "B": 2},
			D4:     "d-4",
			labels: Labels{
				"id": fmt.Sprintf("v%d", i),
			},
		}
		err = DB.Insert(object)
		g.Expect(err).To(gomega.BeNil())
	}
	// List all; detail level=0
	itr, err := DB.Iter(
		&TestObject{},
		ListOptions{})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(itr.Len()).To(gomega.Equal(10))
	var list []TestObject
	for {
		object := TestObject{}
		if itr.NextWith(&object) {
			g.Expect(err).To(gomega.BeNil())
			list = append(list, object)
		} else {
			break
		}
	}
	g.Expect(len(list)).To(gomega.Equal(10))
	// List all; detail level=0
	itr, err = DB.Iter(
		&TestObject{},
		ListOptions{})
	g.Expect(err).To(gomega.BeNil())
	for object, hasNext := itr.Next(); hasNext; object, hasNext = itr.Next() {
		g.Expect(err).To(gomega.BeNil())
		_, cast := object.(Model)
		g.Expect(cast).To(gomega.BeTrue())
	}
}

func TestWatch(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	DB := New("/tmp/test-watch.db", &TestObject{})
	err := DB.Open(true)
	defer func() {
		_ = DB.Close(false)
	}()
	g.Expect(err).To(gomega.BeNil())
	// Handler A
	handlerA := &TestHandler{
		options: WatchOptions{Snapshot: true},
		name:    "A",
	}
	watchA, err := DB.Watch(&TestObject{}, handlerA)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(watchA).ToNot(gomega.BeNil())
	g.Expect(watchA.Alive()).To(gomega.BeTrue())
	N := 10
	// Insert
	for i := 0; i < N; i++ {
		object := &TestObject{
			ID:   i,
			Name: "Elmer",
		}
		err = DB.Insert(object)
		g.Expect(err).To(gomega.BeNil())
	}
	// Handler B
	handlerB := &TestHandler{
		options: WatchOptions{Snapshot: true},
		name:    "B",
	}
	watchB, err := DB.Watch(&TestObject{}, handlerB)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(watchB).ToNot(gomega.BeNil())
	// Update
	for i := 0; i < N; i++ {
		object := &TestObject{
			ID:     i,
			Name:   "Fudd",
			Age:    18,
			Int8:   8,
			Int16:  16,
			Int32:  32,
			Bool:   true,
			Object: TestEncoded{Name: "json"},
			Slice:  []string{"hello", "world"},
			Map:    map[string]int{"A": 1, "B": 2},
			D4:     "d-4",
			labels: Labels{
				"id": fmt.Sprintf("v%d", i),
			},
		}
		err = DB.Update(object)
		g.Expect(err).To(gomega.BeNil())
	}
	// Handler C
	handlerC := &TestHandler{
		options: WatchOptions{Snapshot: true},
		name:    "C",
	}
	watchC, err := DB.Watch(&TestObject{}, handlerC)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(watchC).ToNot(gomega.BeNil())
	// Handler D (no snapshot)
	handlerD := &TestHandler{name: "D"}
	watchD, err := DB.Watch(&TestObject{}, handlerD)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(watchC).ToNot(gomega.BeNil())
	// Delete
	for i := 0; i < N; i++ {
		object := &TestObject{
			ID: i,
		}
		err = DB.Delete(object)
		g.Expect(err).To(gomega.BeNil())
	}
	for i := 0; i < N; i++ {
		time.Sleep(time.Millisecond * 10)
		if len(handlerA.created) != N ||
			len(handlerA.updated) != N ||
			len(handlerA.created) != N ||
			len(handlerB.created) != N ||
			len(handlerB.updated) != N ||
			len(handlerB.created) != N ||
			len(handlerC.created) != N ||
			len(handlerC.created) != N {
			continue
		} else {
			break
		}
	}
	g.Expect(handlerA.started).To(gomega.BeTrue())
	g.Expect(handlerB.started).To(gomega.BeTrue())
	g.Expect(handlerC.started).To(gomega.BeTrue())
	g.Expect(handlerD.started).To(gomega.BeTrue())
	g.Expect(handlerA.parity).To(gomega.BeTrue())
	g.Expect(handlerB.parity).To(gomega.BeTrue())
	g.Expect(handlerC.parity).To(gomega.BeTrue())
	g.Expect(handlerD.parity).To(gomega.BeTrue())
	//
	// The scenario is:
	// 1. handler A created
	// 2. (N) models created. handler A should get (N) CREATE events.
	// 3. handler B created.  handler B should get (N) CREATE events.
	// 4. (N) models updated. handler A & B should get (N) UPDATE events.
	// 5. Handler C created.  handler C should get (N) CREATE events.
	// 6. (N) models deleted. handler A,B,C should get (N) DELETE events.
	all := []TestEvent{}
	created := []TestEvent{}
	updated := []TestEvent{}
	deleted := []TestEvent{}
	for _, action := range []uint8{Created, Updated, Deleted} {
		for i := 0; i < N; i++ {
			switch action {
			case Created:
				created = append(
					created,
					TestEvent{
						action: action,
						model:  &TestObject{ID: i},
					})
			case Updated:
				updated = append(
					updated,
					TestEvent{
						action: action,
						model:  &TestObject{ID: i},
					})
			case Deleted:
				deleted = append(
					deleted,
					TestEvent{
						action: action,
						model:  &TestObject{ID: i},
					})
			}
			all = append(
				all,
				TestEvent{
					action: action,
					model:  &TestObject{ID: i},
				})
		}
	}
	g.Expect(func() (eq bool) {
		h := handlerA
		if len(all) != len(h.all) {
			return
		}
		for i := 0; i < len(all); i++ {
			if all[i].action != h.all[i].action ||
				all[i].model.ID != h.all[i].model.ID {
				return
			}
		}
		return true
	}()).To(gomega.BeTrue())
	g.Expect(func() (eq bool) {
		h := handlerB
		if len(all) != len(h.all) {
			return
		}
		for i := 0; i < len(all); i++ {
			if all[i].action != h.all[i].action ||
				all[i].model.ID != h.all[i].model.ID {
				return
			}
		}
		return true
	}()).To(gomega.BeTrue())
	all = []TestEvent{}
	for _, action := range []uint8{Created, Deleted} {
		for i := 0; i < N; i++ {
			all = append(
				all,
				TestEvent{
					action: action,
					model:  &TestObject{ID: i},
				})
		}
	}
	g.Expect(func() (eq bool) {
		h := handlerC
		if len(all) != len(h.all) {
			return
		}
		for i := 0; i < len(all); i++ {
			if all[i].action != h.all[i].action ||
				all[i].model.ID != h.all[i].model.ID {
				return
			}
		}
		return true
	}()).To(gomega.BeTrue())
	g.Expect(func() (eq bool) {
		h := handlerD
		if len(deleted) != len(h.deleted) {
			return
		}
		for i := 0; i < len(deleted); i++ {
			if deleted[i].model.ID != h.deleted[i] {
				return
			}
		}
		return true
	}()).To(gomega.BeTrue())

	//
	// Test watch end.
	watchA.End()
	watchB.End()
	watchC.End()
	watchD.End()
	ended := false
	for i := 0; i < 10; i++ {
		if watchA.started || watchB.started || watchC.started || watchD.started {
			time.Sleep(50 * time.Millisecond)
		} else {
			ended = true
			break
		}
	}
	g.Expect(len(watchA.journal.watches)).To(gomega.Equal(0))
	g.Expect(ended).To(gomega.BeTrue())
	g.Expect(handlerA.done).To(gomega.BeTrue())
	g.Expect(handlerB.done).To(gomega.BeTrue())
	g.Expect(handlerC.done).To(gomega.BeTrue())
	g.Expect(handlerD.done).To(gomega.BeTrue())
}

func TestCloseDB(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	DB := New("/tmp/test-close-db.db", &TestObject{})
	err := DB.Open(true)
	defer func() {
		_ = DB.Close(false)
	}()
	g.Expect(err).To(gomega.BeNil())
	handler := &TestHandler{
		options: WatchOptions{Snapshot: true},
		name:    "A",
	}
	watch, err := DB.Watch(&TestObject{}, handler)
	for i := 0; i < 10; i++ {
		if !watch.started {
			time.Sleep(50 * time.Millisecond)
		} else {
			break
		}
	}
	g.Expect(handler.started).To(gomega.BeTrue())
	g.Expect(handler.done).To(gomega.BeFalse())
	_ = DB.Close(true)
	for _, session := range DB.(*Client).pool.sessions {
		g.Expect(session.closed).To(gomega.BeTrue())
	}
	for i := 0; i < 100; i++ {
		if !watch.done {
			time.Sleep(50 * time.Millisecond)
		} else {
			break
		}
	}

	g.Expect(handler.done).To(gomega.BeTrue())
}

func TestMutatingWatch(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	DB := New("/tmp/test-mutating-watch.db", &TestObject{})
	err := DB.Open(true)

	g.Expect(err).To(gomega.BeNil())

	// Handler A
	handlerA := &MutatingHandler{
		options: WatchOptions{Snapshot: true},
		name:    "A",
		DB:      DB,
	}
	watchA, err := DB.Watch(&TestObject{}, handlerA)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(watchA).ToNot(gomega.BeNil())
	// Handler B
	handlerB := &MutatingHandler{
		options: WatchOptions{Snapshot: true},
		name:    "B",
		DB:      DB,
	}
	watchB, err := DB.Watch(&TestObject{}, handlerB)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(watchB).ToNot(gomega.BeNil())
	N := 10
	// Insert
	for i := 0; i < N; i++ {
		object := &TestObject{
			ID:   i,
			Name: "Elmer",
		}
		err = DB.Insert(object)
		g.Expect(err).To(gomega.BeNil())
	}

	for {
		time.Sleep(time.Millisecond * 10)
		if len(handlerA.updated) > 100 {
			break
		}
	}
}

func TestExecute(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	type Person struct {
		ID   int    `sql:"pk"`
		Name string `sql:""`
	}
	DB := New("/tmp/test-execute.db", &Person{})
	err := DB.Open(true)
	defer func() {
		_ = DB.Close(false)
	}()

	g.Expect(err).To(gomega.BeNil())

	result, err := DB.Execute(
		"INSERT INTO Person (id, name) values (0, 'john');")
	g.Expect(err).To(gomega.BeNil())
	g.Expect(result.RowsAffected()).To(gomega.Equal(int64(1)))
}

func TestSession(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	DB := New("/tmp/test-session.db", &TestObject{})
	DB.Open(true)
	defer func() {
		_ = DB.Close(false)
	}()

	pool := DB.(*Client).pool

	w := pool.Writer()
	g.Expect(w.id).To(gomega.Equal(0))
	for n := 1; n < 11; n++ {
		r := pool.Reader()
		g.Expect(r.id).To(gomega.Equal(n))
	}
}

//
// Remove leading __ to enable.
func __TestDbLocked(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	DB := New("/tmp/test-db-locked.db", &TestObject{}, &PlainObject{})
	err := DB.Open(true)
	defer func() {
		_ = DB.Close(false)
	}()
	errChan := make(chan error)
	endChan := make(chan int)
	go func() {
		tx, _ := DB.Begin()
		defer func() {
			errChan <- err
			close(errChan)
			_ = tx.End()
		}()
		for i := 0; i < 20000; i++ {
			object := &TestObject{
				ID:   i,
				Name: "Elmer",
			}
			err = tx.Insert(object)
			errChan <- err
			if err != nil {
				return
			}
		}
		err = tx.Commit()
	}()
	go func() {
		defer close(endChan)
		n := int64(0)
		for err = range errChan {
			g.Expect(err).To(gomega.BeNil())
			n, err = DB.Count(&TestObject{}, nil)
			g.Expect(err).To(gomega.BeNil())
		}
		fmt.Printf("Count:%d", n)
	}()

	for _ = range endChan {
	}
}

//
// Remove leading __ to enable.
func __TestConcurrency(t *testing.T) {
	var err error

	DB := New("/tmp/test-concurrency.db", &TestObject{})
	DB.Open(true)
	defer func() {
		_ = DB.Close(false)
	}()

	N := 1000

	direct := func(done chan int) {
		for i := 0; i < N; i++ {
			m := &TestObject{
				ID:   i,
				Name: "direct",
			}
			err := DB.Insert(m)
			if err != nil {
				panic(err)
			}
			fmt.Printf("direct|%d\n", i)
			time.Sleep(time.Millisecond * 10)
		}
		done <- 0
	}
	read := func(done chan int) {
		time.Sleep(time.Second)
		for i := 0; i < N; i++ {
			m := &TestObject{
				ID:   i,
				Name: "direct",
			}
			go func() {
				err := DB.Get(m)
				if err != nil {
					if errors.Is(err, NotFound) {
						fmt.Printf("read|%d _____%s\n", i, err)
					} else {
						panic(err)
					}
				}
				fmt.Printf("read|%d\n", i)
			}()
			time.Sleep(time.Millisecond * 100)
		}
		done <- 0
	}
	del := func(done chan int) {
		time.Sleep(time.Second * 3)
		for i := 0; i < N/2; i++ {
			m := &TestObject{
				ID: i,
			}
			go func() {
				err := DB.Delete(m)
				if err != nil {
					if errors.Is(err, NotFound) {
						fmt.Printf("del|%d _____%s\n", i, err)
					} else {
						panic(err)
					}
				}
				fmt.Printf("del|%d\n", i)
			}()
			time.Sleep(time.Millisecond * 300)
		}
		done <- 0
	}
	update := func(done chan int) {
		for i := 0; i < N; i++ {
			m := &TestObject{
				ID:   i,
				Name: "direct",
			}
			go func() {
				err := DB.Update(m)
				if err != nil {
					if errors.Is(err, NotFound) {
						fmt.Printf("update|%d _____%s\n", i, err)
					} else {
						panic(err)
					}
				}
				fmt.Printf("update|%d\n", i)
			}()
			time.Sleep(time.Millisecond * 20)
		}
		done <- 0
	}
	transaction := func(done chan int) {
		time.Sleep(time.Millisecond * 100)
		var tx *Tx
		defer func() {
			if tx != nil {
				err := tx.Commit()
				if err != nil {
					panic(err)
				}
			}
		}()
		threshold := float64(10)
		for i := N; i < N*2; i++ {
			if tx == nil {
				tx, err = DB.Begin()
				if err != nil {
					panic(err)
				}
			}
			m := &TestObject{
				ID:   i,
				Name: "transaction",
			}
			err = tx.Insert(m)
			if err != nil {
				panic(err)
			}
			//time.Sleep(time.Second*3)
			if math.Mod(float64(i), threshold) == 0 {
				err = tx.Commit()
				if err != nil {
					panic(err)
				}
				tx = nil
				fmt.Printf("commit|%d\n", i)
			}
			fmt.Printf("transaction|%d\n", i)
			time.Sleep(time.Millisecond * 100)
		}
		done <- 0
	}

	mark := time.Now()

	done := make(chan int)
	fnList := []func(chan int){
		direct,
		transaction,
		read,
		update,
		del,
	}
	for _, fn := range fnList {
		go fn(done)
	}
	for range fnList {
		<-done
	}

	fmt.Println(time.Since(mark))
}
