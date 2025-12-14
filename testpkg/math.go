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