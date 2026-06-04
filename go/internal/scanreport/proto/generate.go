// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

// Package proto contains generated protobuf types for SonarScanner report submission.
//
// To regenerate after modifying .proto files:
//
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative constants.proto scanner-report.proto
package proto
