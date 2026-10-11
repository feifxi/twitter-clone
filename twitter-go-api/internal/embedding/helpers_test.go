package embedding_test

import "github.com/pgvector/pgvector-go"

func vectorPtr(values []float32) *pgvector.Vector {
	v := pgvector.NewVector(values)
	return &v
}
