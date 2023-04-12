package database

import (
	"github.com/iotaledger/hive.go/app"
)

// ParametersDatabase contains the definition of the parameters used by the ParametersDatabase.
type ParametersDatabase struct {
	Tangle struct {
		// Path defines the path to the tangle database folder.
		Path string `default:"database/tangle" usage:"the path to the tangle database folder"`
	}

	UTXO struct {
		// Path defines the path to the UTXO database folder.
		Path string `default:"database/utxo" usage:"the path to the UTXO database folder"`
	}

	// Debug defines whether to ignore the check for corrupted databases (should only be used for debug reasons).
	Debug bool `default:"false" usage:"ignore the check for corrupted databases (should only be used for debug reasons)"`
}

var ParamsDatabase = &ParametersDatabase{}

var params = &app.ComponentParams{
	Params: map[string]any{
		"db": ParamsDatabase,
	},
	Masked: nil,
}
