package main

import "embed"

// files holds everything the binary serves: posts, templates and the stylesheet.
//
//go:embed content templates static
var files embed.FS
