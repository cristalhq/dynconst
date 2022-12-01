package dynconst

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
)

// Var is an abstract type for all dynconst variables.
type Var interface {
	// String returns a valid JSON value for the variable.
	// Types with String methods that do not return valid JSON
	// (such as time.Time) must not be used as a Var.
	String() string
}

// Int is a 64-bit integer variable that satisfies the Var interface.
type Int struct {
	i int64
}

func NewInt(value int64, name string) *Int {
	v := new(Int)
	atomic.StoreInt64(&v.i, value)
	publish(name, v)
	return v
}

func (v *Int) String() string {
	return strconv.FormatInt(v.Value(), 10)
}

func (v *Int) Value() int64 {
	return atomic.LoadInt64(&v.i)
}

func (v *Int) set(s string) error {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	atomic.StoreInt64(&v.i, i)
	return nil
}

// Float is a 64-bit float variable that satisfies the Var interface.
type Float struct {
	f atomic.Uint64
}

func NewFloat(value float64, name string) *Float {
	v := new(Float)
	v.f.Store(math.Float64bits(value))
	publish(name, v)
	return v
}

func (v *Float) String() string {
	return strconv.FormatFloat(v.Value(), 'g', -1, 64)
}

func (v *Float) Value() float64 {
	return math.Float64frombits(v.f.Load())
}

func (v *Float) set(s string) error {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	v.f.Store(math.Float64bits(f))
	return nil
}

// String is a string variable, and satisfies the Var interface.
type String struct {
	s atomic.Value
}

func NewString(value, name string) *String {
	v := new(String)
	v.s.Store(value)
	publish(name, v)
	return v
}

// String implements the Var interface.
// To get the unquoted string use Value.
func (v *String) String() string {
	b, _ := json.Marshal(v.Value())
	return string(b)
}

func (v *String) Value() string {
	p, _ := v.s.Load().(string)
	return p
}

func (v *String) set(s string) error {
	v.s.Store(s)
	return nil
}

var (
	vars      sync.Map
	varKeysMu sync.RWMutex
	varKeys   []string // sorted
)

// publish declares a named exported variable. This should be called from a
// package's init function when it creates its Vars. If the name is already
// registered then this will panic.
func publish(name string, v Var) {
	if _, dup := vars.LoadOrStore(name, v); dup {
		panic("dynconst: reuse of exported var name: " + name)
	}

	varKeysMu.Lock()
	defer varKeysMu.Unlock()

	varKeys = append(varKeys, name)
	sort.Strings(varKeys)
}

// Walk calls fn for each exported variable.
// The global variable map is locked during the iteration,
// but existing entries may be concurrently updated.
func Walk(fn func(key string, value Var)) {
	varKeysMu.RLock()
	defer varKeysMu.RUnlock()

	for _, key := range varKeys {
		value, _ := vars.Load(key)
		fn(key, value.(Var))
	}
}

// Handler returns the dynconst HTTP Handler.
func Handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ViewHandler(w, r)
	case http.MethodPost:
		SetHandler(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func SetHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	name := params.Get("name")
	value := params.Get("value")

	if name == "" || value == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	v, ok := vars.Load(name)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	val, ok := v.(interface{ set(string) error })
	if !ok {
		panic("unreachable")
	}

	if err := val.set(value); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func ViewHandler(w http.ResponseWriter, r *http.Request) {
	switch format := r.URL.Query().Get("format"); format {
	case "", "json":
		writeJSON(w)
	case "text":
		writeText(w)
	default:
		msg := fmt.Sprintf("unknown format: %q (want 'json' or 'text'", format)
		http.Error(w, msg, http.StatusBadRequest)
	}
}

func writeJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	fmt.Fprintf(w, "{\n")
	first := true
	Walk(func(key string, value Var) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", key, value)
	})
	fmt.Fprintf(w, "\n}\n")
}

func writeText(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	first := true
	Walk(func(key string, value Var) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%s: %s", key, value)
	})
}
