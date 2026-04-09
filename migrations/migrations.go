// SPDX-License-Identifier: BUSL-1.1

package migrations

import "embed"

//go:embed postgres/*.sql
var PostgresFS embed.FS
