package builder

import "strings"

// LoggerFlags are static settings of tracer/logger package.
// tracer/logger package can increase performance by some settings.
// LoggerFlags.EditContent() can change settings by editing source codes on tracer/logger package.
//
// Specification:
//   Flag Definition Line:
//     * Flag MUST be a constant instead of variable.
//     * Flag type MUST be bool.
//     * Flag definition line format is "{flagName} = false //@@GAT#FLAG#"
//
//   Switchable Comment:
//     * Comment MUST have a prefix of "//@@GAT@{flagName}@"
//     * If the specific flag is true, comments will be removed and commented out codes can be executable.
type LoggerFlags struct {
	UseCallersFrames      bool
	UseNonStandardRuntime bool
}

func (f LoggerFlags) EditContent(content string) string {
	if f.UseNonStandardRuntime {
		content = f.enableFlag(content, "useNonStandardRuntime")
	}

	if f.UseCallersFrames {
		content = f.enableFlag(content, "useCallersFrames")
	}
	return content
}

func (LoggerFlags) enableFlag(content string, flagName string) string {
	constSuffix := "//@@GAT#FLAG#"
	constDef := flagName + " = true"
	comment := "//@@GAT@" + flagName + "@"

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasSuffix(line, constSuffix) && strings.Contains(line, flagName) {
			line = constDef
		}
		line = strings.Replace(line, comment, "", -1)
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}
