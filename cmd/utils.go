package cmd

import (
	"strings"
)

func verbosePrintln(line string) {
	if verbose {
		println(line)
	}
}

// isChannelFile checks if the path matches the pattern for channel JSON files.
// Pattern: [channel]/[date].json (e.g., "general/2024-01-15.json")
//
// Slack exports store message history in per-channel directories, with each day's
// messages in a separate JSON file. These files contain posts that may reference
// file attachments we need to download. We skip directories starting with "__"
// (like __uploads) which are used for other purposes.
func isChannelFile(name string) bool {
	splits := strings.Split(name, "/")
	return len(splits) == 2 && !strings.HasPrefix(splits[0], "__") && strings.HasSuffix(splits[1], ".json")
}
