// Lesson 27: reflection and struct tags.
//
// Reflection inspects types at run time. It powers encoding/json, ORMs,
// validators and config loaders. It is also slow, unchecked by the compiler
// and easy to get wrong - so use it to build GENERIC INFRASTRUCTURE, never
// for ordinary application logic.
//
// Run: go run ./lessons/27_reflection_tags
package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type User struct {
	Name     string `json:"name" validate:"required,min=2"      db:"full_name"`
	Email    string `json:"email" validate:"required,email"     db:"email"`
	Age      int    `json:"age" validate:"min=18,max=130"       db:"age"`
	Nickname string `json:"nickname,omitempty" validate:""      db:"nickname"`
	internal string
}

func main() {
	u := User{Name: "A", Email: "not-an-email", Age: 15}

	// --- Inspecting a type ---
	t := reflect.TypeOf(u)
	v := reflect.ValueOf(u)
	fmt.Printf("type=%s kind=%s fields=%d\n", t.Name(), t.Kind(), t.NumField())

	for i := range t.NumField() {
		f := t.Field(i)
		// Unexported fields can be seen but not read via reflection.
		fmt.Printf("  %-9s %-7s exported=%-5t json=%-18q validate=%q\n",
			f.Name, f.Type, f.IsExported(), f.Tag.Get("json"), f.Tag.Get("validate"))
		if f.IsExported() {
			_ = v.Field(i).Interface() // safe only when exported
		}
	}

	// --- A tiny validator driven by tags: this is how go-playground/validator
	//     and every config library work under the hood ---
	fmt.Println("\nvalidation errors:")
	for _, e := range Validate(u) {
		fmt.Println("  -", e)
	}

	good := User{Name: "Alice", Email: "alice@example.com", Age: 30}
	fmt.Println("valid user errors:", len(Validate(good)))

	// --- Building a SQL statement from db tags ---
	fmt.Println("\ngenerated SQL:", insertSQL("users", good))

	// --- Setting values requires an ADDRESSABLE value: pass a pointer ---
	target := &User{}
	if err := setField(target, "Name", "Set by reflection"); err != nil {
		fmt.Println("set error:", err)
	}
	fmt.Println("after setField:", target.Name)

	// This is the single most common reflection mistake:
	err := setField(good, "Name", "x") // not a pointer
	fmt.Println("non-pointer:", err)

	// --- Decoding a map into a struct, the way config loaders do ---
	var decoded User
	_ = decodeMap(map[string]string{
		"name":  "Mapped",
		"email": "m@example.com",
		"age":   "44",
	}, &decoded)
	fmt.Printf("decoded from map: %+v\n", decoded)

	// --- Comparing values ---
	fmt.Println("\nDeepEqual on slices:", reflect.DeepEqual([]int{1, 2}, []int{1, 2}))
	fmt.Println("DeepEqual on maps:", reflect.DeepEqual(
		map[string][]int{"a": {1}}, map[string][]int{"a": {1}}))
	// == would not even compile for those types. In TESTS, prefer
	// google/go-cmp over DeepEqual: the diffs are far more readable.

	// --- Kind vs Type: Kind is the underlying category ---
	type Celsius float64
	var c Celsius = 21.5
	ct := reflect.TypeOf(c)
	fmt.Printf("\nType=%v Kind=%v (so switch on Kind, not Type)\n", ct, ct.Kind())

	// --- The cost ---
	fmt.Println(`
When NOT to reflect:
  Use an interface when you need behaviour.
  Use generics when you need the same algorithm over many types.
  Reach for reflect only when the SHAPE of a type is the input to your code
  (serialisers, validators, config/DI containers, test helpers).`)
}

// --- Validate walks the struct tags and applies rules ---
func Validate(s any) []string {
	var errs []string
	v := reflect.ValueOf(s)
	// Accept both a struct and a pointer to one.
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return []string{"not a struct"}
	}
	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}
		value := v.Field(i)

		for _, rule := range strings.Split(tag, ",") {
			name, arg, _ := strings.Cut(rule, "=")
			switch name {
			case "required":
				if value.IsZero() {
					errs = append(errs, field.Name+" is required")
				}
			case "email":
				if s, ok := value.Interface().(string); ok && s != "" && !strings.Contains(s, "@") {
					errs = append(errs, field.Name+" must be a valid email")
				}
			case "min":
				n, _ := strconv.Atoi(arg)
				switch value.Kind() {
				case reflect.String:
					if value.Len() < n {
						errs = append(errs, fmt.Sprintf("%s must be at least %d characters", field.Name, n))
					}
				case reflect.Int, reflect.Int64:
					if value.Int() < int64(n) {
						errs = append(errs, fmt.Sprintf("%s must be at least %d", field.Name, n))
					}
				}
			case "max":
				n, _ := strconv.Atoi(arg)
				if value.Kind() == reflect.Int && value.Int() > int64(n) {
					errs = append(errs, fmt.Sprintf("%s must be at most %d", field.Name, n))
				}
			}
		}
	}
	return errs
}

// insertSQL builds a statement from the db tags.
func insertSQL(table string, s any) string {
	t := reflect.TypeOf(s)
	cols, placeholders := []string{}, []string{}
	for i := range t.NumField() {
		f := t.Field(i)
		col := f.Tag.Get("db")
		if col == "" || col == "-" || !f.IsExported() {
			continue
		}
		cols = append(cols, col)
		placeholders = append(placeholders, "$"+strconv.Itoa(len(cols)))
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
}

// setField assigns to a field by name. ptr MUST be a pointer to a struct.
func setField(ptr any, name string, value any) error {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("setField: need a non-nil pointer, got %T", ptr)
	}
	elem := v.Elem()
	field := elem.FieldByName(name)
	if !field.IsValid() {
		return fmt.Errorf("setField: no field %q", name)
	}
	if !field.CanSet() {
		return fmt.Errorf("setField: %q is not settable (unexported?)", name)
	}
	nv := reflect.ValueOf(value)
	if !nv.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("setField: cannot assign %s to %s", nv.Type(), field.Type())
	}
	field.Set(nv)
	return nil
}

// decodeMap fills a struct from string values, matching on json tags and
// converting to the field's type. This is the core of every config loader.
func decodeMap(src map[string]string, dst any) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("decodeMap: need a pointer to a struct")
	}
	elem := v.Elem()
	t := elem.Type()

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		key, _, _ := strings.Cut(f.Tag.Get("json"), ",") // strip ",omitempty"
		if key == "" {
			key = strings.ToLower(f.Name)
		}
		raw, ok := src[key]
		if !ok {
			continue
		}
		field := elem.Field(i)
		switch field.Kind() {
		case reflect.String:
			field.SetString(raw)
		case reflect.Int, reflect.Int64:
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return fmt.Errorf("field %s: %w", f.Name, err)
			}
			field.SetInt(n)
		case reflect.Bool:
			b, err := strconv.ParseBool(raw)
			if err != nil {
				return fmt.Errorf("field %s: %w", f.Name, err)
			}
			field.SetBool(b)
		}
	}
	return nil
}
