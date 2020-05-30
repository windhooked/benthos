package condition

import "github.com/windhooked/benthos/v3/lib/x/docs"

var partFieldSpec = docs.FieldAdvanced(
	"part",
	`The index of a message within a batch to test the condition against. This
field is only applicable when batching messages
[at the input level](/docs/configuration/batching).

Indexes can be negative, and if so the part will be selected from the end
counting backwards starting from -1.`,
)
