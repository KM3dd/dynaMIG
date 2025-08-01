package config

import "github.com/KM3dd/dynaMIG/internal/types"

var A100_PROFILES = map[string]types.Profile{
	"1g.5gb": {
		GID:  0,
		CID:  0,
		Size: 1,
	},
	"1g.10gb": {
		GID:  9,
		CID:  9,
		Size: 2,
	},
	"2g.10gb": {
		GID:  1,
		CID:  1,
		Size: 2,
	},
	"3g.20gb": {
		GID:  2,
		CID:  2,
		Size: 4,
	},
	"4g.20gb": {
		GID:  3,
		CID:  3,
		Size: 4,
	},
	"7g.40gb": {
		GID:  4,
		CID:  4,
		Size: 8,
	},
}
