package main

import (
	"crypto/rand"
	"encoding/binary"
	"strconv"
)

type IDGenerator interface {
	getID() string
}

type RandomGenerator struct{}

func (g *RandomGenerator) getID() string {
	var n uint64
	binary.Read(rand.Reader, binary.LittleEndian, &n)
	return strconv.FormatUint(n, 36)
}
