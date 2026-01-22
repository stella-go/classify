# Classify

[![Go Version](https://img.shields.io/badge/Go-1.20+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A Go library for automatic data classification and sensitivity grading of table metadata. It identifies entity types from column names and descriptions, then assigns appropriate sensitivity levels based on configurable rules.

## Features

- **Automatic Entity Recognition**: Identifies 100+ entity types (personal identifiers, phone numbers, emails, bank accounts, etc.)
- **Sensitivity Grading**: 4-level sensitivity classification (Public, Internal, Sensitive, Highly Sensitive)
- **Flexible Matching**: Supports exact matching, keyword matching, tokenization (underscore and camelCase), and context-aware identification
- **Built-in Defaults**: Comes with embedded default configurations for common internet company data patterns
- **Concurrent Processing**: Batch classification with configurable worker pools
- **Zero Dependencies**: Pure Go implementation with no external dependencies

## Installation

```bash
go get github.com/stella-go/classify
```

## Quick Start

### Using Default Configuration

```go
package main

import (
    "fmt"
    "github.com/stella-go/classify"
)

func main() {
    // Create classifier with built-in default configuration
    classifier, err := classify.NewDefaultClassifier()
    if err != nil {
        panic(err)
    }

    // Define table metadata
    table := &classify.Table{
        Name:        "t_user_info",
        Description: "User information table",
        Columns: []classify.Column{
            {Name: "id", Description: "Primary key", DataType: "bigint"},
            {Name: "user_name", Description: "Username", DataType: "varchar"},
            {Name: "mobile", Description: "Phone number", DataType: "varchar"},
            {Name: "email", Description: "Email address", DataType: "varchar"},
            {Name: "id_card_no", Description: "ID card number", DataType: "varchar"},
        },
    }

    // Classify the table
    result, err := classifier.Classify(table)
    if err != nil {
        panic(err)
    }

    // Print results
    fmt.Printf("Table Classification: %s\n", result.Classification)
    fmt.Printf("Table Sensitivity Level: %d - %s\n", result.Table.Level, result.Table.LevelName)
    
    for colName, colResult := range result.Columns {
        fmt.Printf("  Column [%s]: Level %d, Entity: %s\n", 
            colName, colResult.Level, colResult.EntityType)
    }
}
```

### Using Custom Configuration

```go
// Load your custom configuration
configJSON := `{"version": "1.0", "tables": [...]}`
entityJSON := `{"entities": [...]}`

classifier, err := classify.NewClassifier(configJSON, entityJSON)
if err != nil {
    panic(err)
}
```

### Batch Processing

```go
tables := []*classify.Table{
    {Name: "t_user", ...},
    {Name: "t_order", ...},
    {Name: "t_payment", ...},
}

// Process with 4 concurrent workers
results, err := classifier.ClassifyBatch(tables, 4)
```

## Sensitivity Levels

| Level | Name | Description |
|-------|------|-------------|
| 1 | Public Data | Non-sensitive, can be publicly disclosed |
| 2 | Internal Data | Internal use only, not for external sharing |
| 3 | Sensitive Data | Contains personal or business sensitive information |
| 4 | Highly Sensitive | Financial, authentication, or critical personal data |

## Supported Entity Types

The library recognizes 100+ entity types, including:

**Personal Information**
- `personal_identifier` - User IDs, member IDs
- `name` - Real names, usernames
- `id_card` - ID card numbers
- `phone` - Phone numbers
- `email` - Email addresses
- `address` - Physical addresses

**Financial Information**
- `bank_account` - Bank account numbers
- `credit_card` - Credit card numbers
- `salary` - Salary information
- `transaction_amount` - Transaction amounts

**Authentication**
- `password` - Passwords
- `token` - Access tokens
- `api_key` - API keys

**Behavioral Data**
- `ip_address` - IP addresses
- `device_id` - Device identifiers
- `location_track` - GPS coordinates
- `behavior_log` - User behavior logs

See [default-entity.json](default-entity.json) for the complete list.

## Configuration

### Entity Configuration

Entities are defined in JSON format:

```json
{
  "entities": [
    {
      "type": "phone",
      "name": "Phone Number",
      "description": "Mobile or landline phone numbers",
      "keywords": ["phone", "mobile", "tel", "telephone", "cell"],
      "defaultLevel": 4
    }
  ]
}
```

### Table Configuration

Standard tables define expected columns and their classifications:

```json
{
  "tables": [
    {
      "tableName": "user_basic_info",
      "category": "User Data",
      "level": 3,
      "columns": [
        {
          "columnName": "phone",
          "category": "Contact Info",
          "level": 4,
          "entityType": "phone"
        }
      ]
    }
  ]
}
```

## API Reference

### Types

```go
// Classifier - Main classifier instance
type Classifier struct { ... }

// Table - Input table metadata
type Table struct {
    Name        string
    Description string
    Columns     []Column
}

// Column - Input column metadata
type Column struct {
    Name        string
    Description string
    DataType    string
}

// Result - Classification result
type Result struct {
    Table          *TableClassificationResult
    Columns        map[string]*ColumnClassificationResult
    MatchedTable   string
    MatchScore     float64
    Classification string
}
```

### Functions

```go
// Create classifier with custom configuration
func NewClassifier(config, entityConfig string) (*Classifier, error)

// Create classifier with built-in defaults
func NewDefaultClassifier() (*Classifier, error)

// Classify a single table
func (cl *Classifier) Classify(input *Table) (*Result, error)

// Classify multiple tables concurrently
func (cl *Classifier) ClassifyBatch(inputs []*Table, workerCount int) ([]*Result, error)

// Get all table categories
func (cl *Classifier) GetTableCategories() []string

// Get all entity types
func (cl *Classifier) GetEntityTypes() []string
```

## Performance

Benchmark results on Intel Core i7-8750H @ 2.20GHz:

| Operation | Time | Memory | Allocations |
|-----------|------|--------|-------------|
| Single Classify | ~6μs | 1.5KB | 30 allocs |
| Batch 100 tables (4 workers) | ~275μs | 109KB | 1909 allocs |

## Thread Safety

The `Classifier` instance is safe for concurrent use. All internal state is read-only after initialization, allowing multiple goroutines to call `Classify` simultaneously.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
