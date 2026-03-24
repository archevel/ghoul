// Package testpkg provides simple mathematical functions for testing wraith tool
package testpkg

import "errors"

// Add adds two integers and returns the result
func Add(a, b int) int {
	return a + b
}

// Subtract subtracts b from a
func Subtract(a, b int) int {
	return a - b
}

// Multiply multiplies two integers
func Multiply(a, b int) int {
	return a * b
}

// Divide divides a by b and returns an error if b is zero
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

// IsEven checks if a number is even
func IsEven(n int) bool {
	return n%2 == 0
}

// Greeting returns a greeting string
func Greeting(name string) string {
	return "Hello, " + name + "!"
}

// Sum calculates the sum of a slice of integers
func Sum(numbers []int) int {
	total := 0
	for _, n := range numbers {
		total += n
	}
	return total
}

// Person represents a simple person struct
type Person struct {
	Name string
	Age  int
}

// GetAge returns the person's age
func (p Person) GetAge() int {
	return p.Age
}

// SetAge sets the person's age
func (p *Person) SetAge(age int) {
	p.Age = age
}

// NewPerson creates a new Person instance
func NewPerson(name string, age int) *Person {
	return &Person{Name: name, Age: age}
}

// Greeter is an interface for things that can greet
type Greeter interface {
	Greet(name string) string
}

// Apply calls the given function with two integers and returns the result
func Apply(f func(int, int) int, a, b int) int {
	return f(a, b)
}

// Transform applies a transformation function to each element of a slice
func Transform(numbers []int, f func(int) int) []int {
	result := make([]int, len(numbers))
	for i, n := range numbers {
		result[i] = f(n)
	}
	return result
}

// WithError calls a function that may return an error
func WithError(f func(string) (int, error), input string) (int, error) {
	return f(input)
}

// SendOnChannel sends a value on a channel (unwrappable — channels not supported)
func SendOnChannel(ch chan int, val int) {
	ch <- val
}

// LookupMap looks up a key in a map (unwrappable — maps not supported)
func LookupMap(m map[string]int, key string) (int, bool) {
	v, ok := m[key]
	return v, ok
}

// Variadic takes variadic args (unwrappable — variadic not supported)
func Variadic(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// JoinWith joins strings with a separator (mixed: regular + variadic params)
func JoinWith(sep string, parts ...string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

// Type alias for int
type Score int

// AddScores adds two scores
func AddScores(a, b Score) Score {
	return a + b
}

// Named function type
type IntTransform func(int) int

// ApplyTransform applies a named function type to a value
func ApplyTransform(f IntTransform, val int) int {
	return f(val)
}

// unexported type returned from an exported function
type result struct {
	Value int
}

// MakeResult creates an unexported result type
func MakeResult(val int) *result {
	return &result{Value: val}
}

// GetResultValue extracts the value from a result
func GetResultValue(r *result) int {
	return r.Value
}

// MaxValue is an exported constant
const MaxValue = 1000

// Pi is an exported float constant
const Pi = 3.14159

// DefaultName is an exported string constant
const DefaultName = "Ghoul"

// Counter is an exported variable
var Counter int = 0

// ForEach calls a void callback for each element
func ForEach(numbers []int, f func(int)) {
	for _, n := range numbers {
		f(n)
	}
}