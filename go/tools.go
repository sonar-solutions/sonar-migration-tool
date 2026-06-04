// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

//go:build tools

package tools

// Force minimum versions for transitive dependencies with known CVEs.
// These imports prevent go mod tidy from removing the version overrides.
import _ "golang.org/x/image/draw"
