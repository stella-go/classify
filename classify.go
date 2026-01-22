// Copyright 2010-NOW the original author or authors.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package classify

import (
	"encoding/json"
	"strings"
	"sync"
	"unicode"
)

// ==================== 配置结构体定义 ====================

// Entity 命名实体定义
type Entity struct {
	Type         string   `json:"type"`         // 实体类型标识
	Name         string   `json:"name"`         // 实体名称（中文）
	Description  string   `json:"description"`  // 实体描述
	Keywords     []string `json:"keywords"`     // 关键词列表，用于匹配
	DefaultLevel int      `json:"defaultLevel"` // 默认敏感级别
}

// EntityConfig 命名实体配置
type EntityConfig struct {
	Entities []Entity `json:"entities"`
}

// LevelInfo 级别信息
type LevelInfo struct {
	Name        string `json:"name"`        // 级别名称
	Description string `json:"description"` // 级别描述
}

// ColumnConfig 列配置定义
type ColumnConfig struct {
	ColumnName   string `json:"columnName"`   // 列名
	ColumnNameCn string `json:"columnNameCn"` // 列名（中文）
	Category     string `json:"category"`     // 分类
	Level        int    `json:"level"`        // 敏感级别 1-4
	EntityType   string `json:"entityType"`   // 实体类型
	Description  string `json:"description"`  // 描述
}

// TableConfig 表配置定义
type TableConfig struct {
	TableName   string         `json:"tableName"`   // 表名
	TableNameCn string         `json:"tableNameCn"` // 表名（中文）
	Keywords    []string       `json:"keywords"`    // 关键词列表，用于表匹配
	Category    string         `json:"category"`    // 表分类
	Level       int            `json:"level"`       // 表敏感级别
	Description string         `json:"description"` // 表描述
	Columns     []ColumnConfig `json:"columns"`     // 列配置列表
}

// Config 分类分级配置
type Config struct {
	Version     string               `json:"version"`     // 配置版本
	Description string               `json:"description"` // 配置描述
	Levels      map[string]LevelInfo `json:"levels"`      // 级别定义
	Tables      []TableConfig        `json:"tables"`      // 表配置列表
}

// ==================== 预处理索引结构体 ====================

// tableIndex 预构建的表索引（用于快速查找）
type tableIndex struct {
	tablePtr       *TableConfig             // 指向原始表配置
	tableNameLower string                   // 预处理的小写表名
	keywordsLower  []string                 // 预处理的小写关键词列表
	columnIndex    map[string]*ColumnConfig // 预构建的列索引（小写列名 -> 列配置）
}

// ==================== 分类器定义 ====================

// Classifier 分类分级器
type Classifier struct {
	config       *Config       // 分类分级配置
	entityConfig *EntityConfig // 命名实体配置

	// 实体类型到实体的映射，用于快速查找
	entityMap map[string]*Entity
	// 关键词到实体类型的映射，用于快速匹配（预处理为小写）
	keywordToEntity map[string]string

	// 预构建的表索引
	tableNameIndex map[string]*tableIndex // 表名精确匹配索引（小写表名 -> 表索引）
	tableIndexList []*tableIndex          // 所有表索引列表（用于模糊匹配）

	// 级别名称缓存
	levelNames map[int]string
}

// NewClassifier 创建分类器实例
// config: 分类分级配置JSON字符串
// entityConfig: 命名实体配置JSON字符串
func NewClassifier(config string, entityConfig string) (*Classifier, error) {
	c := &Config{}
	if err := json.Unmarshal([]byte(config), c); err != nil {
		return nil, err
	}

	ec := &EntityConfig{}
	if err := json.Unmarshal([]byte(entityConfig), ec); err != nil {
		return nil, err
	}

	classifier := &Classifier{
		config:          c,
		entityConfig:    ec,
		entityMap:       make(map[string]*Entity),
		keywordToEntity: make(map[string]string),
		tableNameIndex:  make(map[string]*tableIndex),
		tableIndexList:  make([]*tableIndex, 0, len(c.Tables)),
		levelNames:      make(map[int]string),
	}

	// 构建实体映射索引
	for i := range ec.Entities {
		entity := &ec.Entities[i]
		classifier.entityMap[entity.Type] = entity
		// 构建关键词索引（预处理为小写）
		for _, keyword := range entity.Keywords {
			classifier.keywordToEntity[strings.ToLower(keyword)] = entity.Type
		}
	}

	// 预构建表索引
	for i := range c.Tables {
		table := &c.Tables[i]
		idx := &tableIndex{
			tablePtr:       table,
			tableNameLower: strings.ToLower(table.TableName),
			keywordsLower:  make([]string, len(table.Keywords)),
			columnIndex:    make(map[string]*ColumnConfig),
		}

		// 预处理关键词为小写
		for j, kw := range table.Keywords {
			idx.keywordsLower[j] = strings.ToLower(kw)
		}

		// 预构建列索引
		for j := range table.Columns {
			col := &table.Columns[j]
			idx.columnIndex[strings.ToLower(col.ColumnName)] = col
		}

		// 添加到表名精确匹配索引
		classifier.tableNameIndex[idx.tableNameLower] = idx
		// 添加到列表（用于模糊匹配）
		classifier.tableIndexList = append(classifier.tableIndexList, idx)
	}

	// 预构建级别名称缓存
	for i := 1; i <= 4; i++ {
		classifier.levelNames[i] = classifier.computeLevelName(i)
	}

	return classifier, nil
}

// NewDefaultClassifier 使用内置默认配置创建分类器实例
// 默认配置文件通过 go:embed 嵌入到二进制中
func NewDefaultClassifier() (*Classifier, error) {
	return NewClassifier(DefaultConfig, DefaultEntity)
}

// ==================== 输入输出结构体 ====================

// Table 输入表元数据
type Table struct {
	Name        string   // 表名
	Description string   // 表描述
	Columns     []Column // 列列表
}

// Column 输入列元数据
type Column struct {
	Name        string // 列名
	Description string // 列描述
	DataType    string // 数据类型
}

// Result 分类分级结果
type Result struct {
	Table          *TableClassificationResult             // 表分类结果
	Columns        map[string]*ColumnClassificationResult // 列分类结果，key为列名
	MatchedTable   string                                 // 匹配到的标准表名
	MatchScore     float64                                // 表匹配相似度分数
	UnmatchedCols  []string                               // 未能精确匹配的列名列表
	Classification string                                 // 整体分类
}

// TableClassificationResult 表分类结果
type TableClassificationResult struct {
	Category    string  // 分类
	Level       int     // 敏感级别 1-4
	LevelName   string  // 级别名称
	Description string  // 描述
	Confidence  float64 // 置信度 0-1
}

// ColumnClassificationResult 列分类结果
type ColumnClassificationResult struct {
	Category    string  // 分类
	Level       int     // 敏感级别 1-4
	LevelName   string  // 级别名称
	EntityType  string  // 识别的实体类型
	Description string  // 描述
	Confidence  float64 // 置信度 0-1
	IsExact     bool    // 是否精确匹配
}

// ==================== 分类分级核心逻辑 ====================

// Classify 执行分类分级
// 分类分级逻辑说明：
// 1. 首先根据输入表名和描述，通过关键词匹配找出相似度最高的标准表
// 2. 根据匹配的标准表，提取其包含的列配置信息
// 3. 对输入表的每一列，尝试与标准表列进行精确匹配
// 4. 对于无法精确匹配的列，通过实体类型识别进行语义匹配
// 5. 最终输出表和列的分类分级结果
func (cl *Classifier) Classify(input *Table) (*Result, error) {
	result := &Result{
		Columns:       make(map[string]*ColumnClassificationResult),
		UnmatchedCols: []string{},
	}

	// Step 1: 匹配最相似的标准表（使用预构建索引）
	matchedTableIdx, matchScore := cl.matchTable(input)
	result.MatchedTable = matchedTableIdx.tablePtr.TableName
	result.MatchScore = matchScore
	result.Classification = matchedTableIdx.tablePtr.Category

	// Step 2: 设置表级别分类结果
	result.Table = &TableClassificationResult{
		Category:    matchedTableIdx.tablePtr.Category,
		Level:       matchedTableIdx.tablePtr.Level,
		LevelName:   cl.getLevelName(matchedTableIdx.tablePtr.Level),
		Description: matchedTableIdx.tablePtr.Description,
		Confidence:  matchScore,
	}

	// Step 3: 对每个输入列进行分类分级（使用预构建的列索引）
	for _, col := range input.Columns {
		colResult := cl.classifyColumn(col, matchedTableIdx.columnIndex, matchedTableIdx.tablePtr)
		result.Columns[col.Name] = colResult
		if !colResult.IsExact {
			result.UnmatchedCols = append(result.UnmatchedCols, col.Name)
		}
	}

	return result, nil
}

// ClassifyBatch 批量执行分类分级（并发处理）
// inputs: 输入表元数据列表
// workerCount: 并发工作协程数量，0表示使用输入数量（每个表一个协程）
// 返回：与输入顺序对应的结果列表
func (cl *Classifier) ClassifyBatch(inputs []*Table, workerCount int) ([]*Result, error) {
	if len(inputs) == 0 {
		return []*Result{}, nil
	}

	// 如果只有一个输入，直接处理
	if len(inputs) == 1 {
		result, err := cl.Classify(inputs[0])
		if err != nil {
			return nil, err
		}
		return []*Result{result}, nil
	}

	// 确定工作协程数量
	if workerCount <= 0 || workerCount > len(inputs) {
		workerCount = len(inputs)
	}

	// 创建结果切片（保持顺序）
	results := make([]*Result, len(inputs))
	errors := make([]error, len(inputs))

	// 使用 WaitGroup 等待所有任务完成
	var wg sync.WaitGroup

	// 创建任务通道
	type task struct {
		index int
		table *Table
	}
	taskChan := make(chan task, len(inputs))

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskChan {
				result, err := cl.Classify(t.table)
				results[t.index] = result
				errors[t.index] = err
			}
		}()
	}

	// 发送任务
	for i, input := range inputs {
		taskChan <- task{index: i, table: input}
	}
	close(taskChan)

	// 等待所有任务完成
	wg.Wait()

	// 检查是否有错误
	for _, err := range errors {
		if err != nil {
			return results, err
		}
	}

	return results, nil
}

// matchTable 匹配最相似的标准表（使用预构建索引）
// 匹配策略：
// 1. 首先尝试表名完全匹配（O(1)查找）
// 2. 然后通过关键词匹配计算相似度
// 3. 返回相似度最高的标准表索引
func (cl *Classifier) matchTable(input *Table) (*tableIndex, float64) {
	inputNameLower := strings.ToLower(input.Name)
	inputDescLower := strings.ToLower(input.Description)

	// 优先尝试表名精确匹配（O(1)）
	if idx, exists := cl.tableNameIndex[inputNameLower]; exists {
		return idx, 1.0
	}

	var bestMatch *tableIndex
	var bestScore float64 = 0

	// 遍历所有表进行模糊匹配
	for _, idx := range cl.tableIndexList {
		score := 0.0

		// 计算表名包含关系得分（使用预处理的小写表名）
		if strings.Contains(inputNameLower, idx.tableNameLower) {
			score += 0.5
		}

		// 计算关键词匹配得分（使用预处理的小写关键词）
		matchedKeywords := 0
		for _, keywordLower := range idx.keywordsLower {
			if strings.Contains(inputNameLower, keywordLower) {
				matchedKeywords++
			}
			if strings.Contains(inputDescLower, keywordLower) {
				matchedKeywords++
			}
		}
		if len(idx.keywordsLower) > 0 {
			score += float64(matchedKeywords) / float64(len(idx.keywordsLower)*2) * 0.5
		}

		if score > bestScore {
			bestScore = score
			bestMatch = idx
		}
	}

	// 如果没有匹配到任何表，返回第一个表作为默认
	if bestMatch == nil && len(cl.tableIndexList) > 0 {
		bestMatch = cl.tableIndexList[0]
		bestScore = 0.1
	}

	return bestMatch, bestScore
}

// classifyColumn 对单个列进行分类分级
// 分类策略：
// 1. 首先尝试精确匹配标准表中的列名
// 2. 如果无法精确匹配，则通过识别列名中的实体类型进行语义匹配
// 3. 根据实体类型查找最相似的标准列配置
func (cl *Classifier) classifyColumn(col Column, columnIndex map[string]*ColumnConfig, matchedTable *TableConfig) *ColumnClassificationResult {
	colNameLower := strings.ToLower(col.Name)

	// Step 1: 尝试精确匹配
	if configCol, exists := columnIndex[colNameLower]; exists {
		return &ColumnClassificationResult{
			Category:    configCol.Category,
			Level:       configCol.Level,
			LevelName:   cl.getLevelName(configCol.Level),
			EntityType:  configCol.EntityType,
			Description: configCol.Description,
			Confidence:  1.0,
			IsExact:     true,
		}
	}

	// Step 2: 尝试模糊匹配列名
	// 注意：短列名（如id）不应该随意匹配到包含它的长列名（如id_card）
	for configColName, configCol := range columnIndex {
		// 输入列名包含标准列名：如 user_name_cn 包含 user_name
		if len(configColName) >= 3 && strings.Contains(colNameLower, configColName) {
			return &ColumnClassificationResult{
				Category:    configCol.Category,
				Level:       configCol.Level,
				LevelName:   cl.getLevelName(configCol.Level),
				EntityType:  configCol.EntityType,
				Description: configCol.Description,
				Confidence:  0.8,
				IsExact:     false,
			}
		}
		// 标准列名包含输入列名：仅当输入列名足够长（>=4字符）时才匹配
		// 避免 id 匹配到 id_card 这种情况
		if len(colNameLower) >= 4 && strings.Contains(configColName, colNameLower) {
			return &ColumnClassificationResult{
				Category:    configCol.Category,
				Level:       configCol.Level,
				LevelName:   cl.getLevelName(configCol.Level),
				EntityType:  configCol.EntityType,
				Description: configCol.Description,
				Confidence:  0.8,
				IsExact:     false,
			}
		}
	}

	// Step 3: 通过实体类型识别进行语义匹配
	entityType := cl.identifyEntityType(col)
	if entityType != "" {
		// 查找标准表中相同实体类型的列
		for _, configCol := range columnIndex {
			if configCol.EntityType == entityType {
				return &ColumnClassificationResult{
					Category:    configCol.Category,
					Level:       configCol.Level,
					LevelName:   cl.getLevelName(configCol.Level),
					EntityType:  entityType,
					Description: col.Description,
					Confidence:  0.6,
					IsExact:     false,
				}
			}
		}

		// 如果标准表中没有相同实体类型，使用实体默认级别
		if entity, exists := cl.entityMap[entityType]; exists {
			return &ColumnClassificationResult{
				Category:    entity.Name,
				Level:       entity.DefaultLevel,
				LevelName:   cl.getLevelName(entity.DefaultLevel),
				EntityType:  entityType,
				Description: entity.Description,
				Confidence:  0.4,
				IsExact:     false,
			}
		}
	}

	// Step 4: 无法识别，返回默认低级别
	return &ColumnClassificationResult{
		Category:    "未分类",
		Level:       1,
		LevelName:   cl.getLevelName(1),
		EntityType:  "unknown",
		Description: "无法识别的列",
		Confidence:  0.1,
		IsExact:     false,
	}
}

// identifyEntityType 识别列的实体类型
// 通过关键词匹配识别列名对应的实体类型
// 匹配策略（按优先级）：
// 1. 综合判断：对于通用名称（如id），结合描述和数据类型判断
// 2. 完整列名精确匹配关键词
// 3. 列名分词后精确匹配关键词
// 4. 列名中相邻词组合匹配关键词（如 bank_account_no 中的 bank_account）
// 5. 描述中精确匹配关键词（按词边界）
// 6. 分词后的单词精确匹配实体关键词
func (cl *Classifier) identifyEntityType(col Column) string {
	// 将列名分词并转小写
	colNameLower := strings.ToLower(col.Name)
	colDescLower := strings.ToLower(col.Description)
	colTypeLower := strings.ToLower(col.DataType)

	// Step 0: 对于通用名称（如 id），综合描述和数据类型判断
	// 避免把表主键误判为 personal_identifier
	if entityType := cl.identifyByContext(colNameLower, colDescLower, colTypeLower); entityType != "" {
		return entityType
	}

	// Step 1: 尝试完整列名精确匹配
	if entityType, exists := cl.keywordToEntity[colNameLower]; exists {
		return entityType
	}

	// Step 2: 分词匹配（按下划线、驼峰分词）
	words := cl.splitColumnName(colNameLower)
	for _, word := range words {
		if entityType, exists := cl.keywordToEntity[word]; exists {
			return entityType
		}
	}

	// Step 3: 尝试相邻词组合匹配（如 bank_account_no -> bank_account, account_no）
	// 这解决了 bank_account_no 无法匹配 bank_account 的问题
	for i := 0; i < len(words)-1; i++ {
		// 尝试两个相邻词的组合
		combo := words[i] + "_" + words[i+1]
		if entityType, exists := cl.keywordToEntity[combo]; exists {
			return entityType
		}
		// 不带下划线的组合
		comboNoUnderscore := words[i] + words[i+1]
		if entityType, exists := cl.keywordToEntity[comboNoUnderscore]; exists {
			return entityType
		}
	}

	// Step 4: 在描述中按词边界查找关键词
	// 将描述分词，避免部分匹配导致的误判
	descWords := cl.splitDescriptionToWords(colDescLower)
	for _, descWord := range descWords {
		if entityType, exists := cl.keywordToEntity[descWord]; exists {
			return entityType
		}
	}

	// Step 5: 遍历所有实体的关键词，与分词后的单词进行精确匹配
	// 注意：这里使用精确匹配而非部分匹配，避免 "page" 匹配到 "age" 的问题
	for _, entity := range cl.entityConfig.Entities {
		for _, keyword := range entity.Keywords {
			keywordLower := strings.ToLower(keyword)
			// 检查分词后的单词是否与关键词完全匹配
			for _, word := range words {
				if word == keywordLower {
					return entity.Type
				}
			}
		}
	}

	return ""
}

// identifyByContext 根据上下文（列名、描述、数据类型）综合判断实体类型
// 主要处理一些通用名称的歧义情况
func (cl *Classifier) identifyByContext(colName, colDesc, colType string) string {
	// 处理 "id" 这种通用列名
	if colName == "id" {
		// 如果描述包含主键相关词，判定为主键
		primaryKeyHints := []string{"主键", "自增", "primary", "auto", "序号", "行号", "记录id", "唯一标识"}
		for _, hint := range primaryKeyHints {
			if strings.Contains(colDesc, hint) {
				return "primary_key"
			}
		}
		// 如果数据类型是整数类型且描述简单，很可能是主键
		integerTypes := []string{"int", "bigint", "integer", "serial", "number"}
		for _, intType := range integerTypes {
			if strings.Contains(colType, intType) {
				// 描述不包含用户、会员等个人相关词，判定为主键
				personalHints := []string{"用户", "会员", "客户", "买家", "卖家", "员工", "user", "member", "customer"}
				isPersonal := false
				for _, hint := range personalHints {
					if strings.Contains(colDesc, hint) {
						isPersonal = true
						break
					}
				}
				if !isPersonal {
					return "primary_key"
				}
			}
		}
	}

	// 处理 xxx_id 形式的列名
	if strings.HasSuffix(colName, "_id") && len(colName) > 3 {
		prefix := colName[:len(colName)-3]
		// 检查前缀是否明确指向个人
		personalPrefixes := []string{"user", "member", "customer", "buyer", "seller", "payer", "emp", "staff", "person"}
		for _, pp := range personalPrefixes {
			if prefix == pp || strings.HasSuffix(prefix, "_"+pp) {
				return "personal_identifier"
			}
		}
		// 检查前缀是否指向其他实体类型
		entityPrefixes := map[string]string{
			"order":    "order_id",
			"product":  "product_id",
			"goods":    "product_id",
			"item":     "product_id",
			"company":  "company_id",
			"corp":     "company_id",
			"shop":     "company_id",
			"store":    "company_id",
			"merchant": "company_id",
			"pay":      "order_id",
			"trans":    "order_id",
			"log":      "primary_key",
			"record":   "primary_key",
		}
		for ep, entityType := range entityPrefixes {
			if prefix == ep || strings.HasPrefix(prefix, ep+"_") {
				return entityType
			}
		}
	}

	return ""
}

// splitDescriptionToWords 将描述文本分词
// 按空格、标点符号等分割成单词
func (cl *Classifier) splitDescriptionToWords(desc string) []string {
	var words []string
	var currentWord strings.Builder

	for _, r := range desc {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			currentWord.WriteRune(r)
		} else {
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// splitColumnName 将列名分词
// 支持下划线分隔和驼峰命名
func (cl *Classifier) splitColumnName(name string) []string {
	// 首先按下划线分割
	parts := strings.Split(name, "_")

	var result []string
	for _, part := range parts {
		// 对每个部分处理驼峰命名
		words := cl.splitCamelCase(part)
		result = append(result, words...)
	}

	return result
}

// splitCamelCase 分割驼峰命名
func (cl *Classifier) splitCamelCase(s string) []string {
	var words []string
	var currentWord strings.Builder

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			if currentWord.Len() > 0 {
				words = append(words, strings.ToLower(currentWord.String()))
				currentWord.Reset()
			}
		}
		currentWord.WriteRune(r)
	}

	if currentWord.Len() > 0 {
		words = append(words, strings.ToLower(currentWord.String()))
	}

	return words
}

// getLevelName 获取级别名称（使用缓存）
func (cl *Classifier) getLevelName(level int) string {
	if name, exists := cl.levelNames[level]; exists {
		return name
	}
	return cl.computeLevelName(level)
}

// computeLevelName 计算级别名称
func (cl *Classifier) computeLevelName(level int) string {
	levelStr := string(rune('0' + level))
	if info, exists := cl.config.Levels[levelStr]; exists {
		return info.Name
	}
	switch level {
	case 1:
		return "公开数据"
	case 2:
		return "内部数据"
	case 3:
		return "敏感数据"
	case 4:
		return "高度敏感数据"
	default:
		return "未知级别"
	}
}

// ==================== 辅助方法 ====================

// GetTableCategories 获取所有表分类
func (cl *Classifier) GetTableCategories() []string {
	categoryMap := make(map[string]bool)
	for _, table := range cl.config.Tables {
		categoryMap[table.Category] = true
	}
	var categories []string
	for cat := range categoryMap {
		categories = append(categories, cat)
	}
	return categories
}

// GetEntityTypes 获取所有实体类型
func (cl *Classifier) GetEntityTypes() []string {
	var types []string
	for _, entity := range cl.entityConfig.Entities {
		types = append(types, entity.Type)
	}
	return types
}

// GetLevelInfo 获取级别信息
func (cl *Classifier) GetLevelInfo(level int) *LevelInfo {
	levelStr := string(rune('0' + level))
	if info, exists := cl.config.Levels[levelStr]; exists {
		return &info
	}
	return nil
}
