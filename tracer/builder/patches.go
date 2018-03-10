package builder

const runtimePatch = `
package runtime

// GoID returns the Goroutine ID.
//go:nosplit
func GoID() int64 {
	gp := getg()
	return gp.goid
}
`

