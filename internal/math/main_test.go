package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbs(t *testing.T) {
	tests := []struct {
		name   string
		values float64
		want   float64
	}{
		{
			name:   "simple test #1",
			values: -3,
			want:   3,
		},
		{
			name:   "simple test #2",
			values: 3,
			want:   3,
		},
		{
			name:   "simple test #3",
			values: -2.000001,
			want:   2.000001,
		},
		{
			name:   "simple test #4",
			values: -0.000000003,
			want:   0.000000003,
		},
		{
			name:   "simple test #5",
			values: -0,
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs := Abs(tt.values)
			assert.Equal(t, abs, tt.want)
		})
	}
}

func TestFullName(t *testing.T) {
	tests := []struct {
		name   string
		values User
		want   string
	}{
		{
			name:   "simple test #1",
			values: User{FirstName: "Sergey", LastName: "Porubov"},
			want:   "Sergey Porubov",
		},
		{
			name:   "simple test #2",
			values: User{FirstName: "", LastName: "Porubov"},
			want:   " Porubov",
		},
		{
			name:   "simple test #3",
			values: User{FirstName: "Sergey", LastName: ""},
			want:   "Sergey ",
		},
		{
			name:   "simple test #4",
			values: User{FirstName: "", LastName: ""},
			want:   " ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			full := tt.values.FullName()
			assert.Equal(t, full, tt.want)
		})
	}
}

func TestAddNew(t *testing.T) {
	tests := []struct {
		name         string
		famous       Family
		newPerson    Person
		Relationship Relationship
		want         struct {
			Error   error
			MapSize int
		}
	}{
		{
			name:         "simple test #1: add first person",
			famous:       Family{},
			newPerson:    Person{FirstName: "Sergey", LastName: "Porubov", Age: 30},
			Relationship: Father,
			want: struct {
				Error   error
				MapSize int
			}{Error: nil, MapSize: 1},
		},
		{
			name:         "simple test #2: add same role",
			famous:       Family{Members: map[Relationship]Person{Father: {FirstName: "Valentin", LastName: "Porubov", Age: 55}}},
			newPerson:    Person{FirstName: "Sergey", LastName: "Porubov", Age: 30},
			Relationship: Father,
			want: struct {
				Error   error
				MapSize int
			}{Error: ErrRelationshipAlreadyExists, MapSize: 1},
		},
		{
			name:         "simple test #2: add second role",
			famous:       Family{Members: map[Relationship]Person{Father: {FirstName: "Valentin", LastName: "Porubov", Age: 55}}},
			newPerson:    Person{FirstName: "Sergey", LastName: "Porubov", Age: 30},
			Relationship: Child,
			want: struct {
				Error   error
				MapSize int
			}{Error: nil, MapSize: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.famous.AddNew(tt.Relationship, tt.newPerson)
			if err == nil {
				assert.Equal(t, tt.famous.Members[tt.Relationship], tt.newPerson)
				assert.Equal(t, len(tt.famous.Members), tt.want.MapSize)
			} else {
				assert.Equal(t, err, ErrRelationshipAlreadyExists)
			}
		})
	}
}
