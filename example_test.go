package flac_test

import (
	"fmt"
	"log"

	"github.com/pchchv/flac"
)

func ExampleParseFile() {
	// parse metadata of love.flac
	stream, err := flac.ParseFile("testdata/love.flac")
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	fmt.Printf("unencoded audio md5sum: %032x\n", stream.Info.MD5sum[:])
	for i, block := range stream.Blocks {
		fmt.Printf("block %d: %v\n", i, block.Type)
	}
	// Output:
	// unencoded audio md5sum: bdf6f7d31f77cb696a02b2192d192a89
	// block 0: seek table
	// block 1: vorbis comment
	// block 2: padding
}
