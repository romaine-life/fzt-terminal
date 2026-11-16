package main

import "github.com/romaine-life/fzt/render"

// version is the release the suite stamps into main (pipeline-templates' release suite);
// the DLL reports it as render.Version, which a local build leaves "UNSET".
var version string

func init() {
	if version != "" {
		render.Version = version
	}
}
