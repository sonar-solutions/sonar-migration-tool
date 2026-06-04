// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package extract

import "github.com/sonar-solutions/sonar-migration-tool/internal/common"

// DataStore is an alias for common.DataStore.
type DataStore = common.DataStore

// NewDataStore creates a DataStore rooted at the given extract directory.
var NewDataStore = common.NewDataStore
