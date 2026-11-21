package uitest

// Export unexported helpers for testing within uitest_test package.
var (
	Normalise = normalise
	GoldenAt  = goldenAt
	DiffLines = diffLines
)
