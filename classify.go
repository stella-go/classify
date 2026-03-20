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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// ==================== 语义匹配相关常量 ====================

// 动词形态映射表 - 将变体映射到原形
var verbForms = map[string]string{
	// 时间相关
	"created": "create", "creation": "create", "creating": "create", "crt": "create",
	"updated": "update", "upd": "update", "updating": "update",
	"deleted": "delete", "del": "delete", "deleting": "delete",
	"modified": "modify", "mod": "modify", "modifying": "modify",
	"generated": "generate", "gen": "generate",
	"processed": "process", "proc": "process",
	"approved": "approve", "rejected": "reject", "confirmed": "confirm",
	"submitted": "submit", "canceled": "cancel", "cancelled": "cancel",
	"activated": "activate", "disabled": "disable", "enabled": "enable",
	"locked": "lock", "unlocked": "unlock", "started": "start", "ended": "end",
	"finished": "finish", "established": "establish", "completed": "complete",
	"registered": "register", "reg": "register",
	"authenticated": "authenticate", "auth": "authenticate",
	"authorized": "authorize",
	"validated":  "validate", "verified": "verify",
	"synced": "sync", "synchronized": "sync",
	"imported": "import", "exported": "export", "exp": "export",
	"inserted": "insert", "ins": "insert",
	"selected": "select", "sel": "select",
	"calculated": "calculate", "calc": "calculate",
}

// 同义词组 - 每个组内的词语义等价
var synonymGroups = [][]string{
	// 状态/类型
	{"status", "state", "condition", "situation"},
	{"type", "kind", "category", "class", "sort"},
	{"level", "grade", "rank", "tier"},
	// 时间
	{"time", "date", "timestamp", "datetime", "at", "when", "moment"},
	{"create", "establish", "found", "initiate", "start"},
	{"update", "modify", "change", "alter", "revise"},
	{"delete", "remove", "erase", "clear", "del"},
	// 标识
	{"id", "identifier", "key", "code", "no", "number", "num"},
	{"name", "title", "label", "caption"},
	// 描述
	{"desc", "description", "detail", "info", "information", "note"},
	// 数量
	{"count", "num", "number", "quantity", "qty", "amount", "total"},
	{"price", "cost", "fee", "charge", "rate", "value", "worth"},
	// 人员
	{"user", "member", "customer", "client", "consumer", "buyer", "account"},
	{"admin", "administrator", "manager", "operator"},
	// 位置
	{"address", "addr", "location", "place", "site", "position"},
	{"city", "town", "municipality"},
	{"province", "state", "region"},
	// 联系
	{"phone", "tel", "telephone", "mobile", "cell", "handset"},
	{"email", "mail", "e-mail", "mailbox"},
	// 内容
	{"content", "text", "body", "detail", "context"},
	{"title", "subject", "topic", "headline", "heading"},
}

// ==================== 三层语义匹配架构 ====================

// normalizeMorphology 形态归一化层 - 将动词变体映射到原形
func (cl *Classifier) normalizeMorphology(words []string) []string {
	result := make([]string, len(words))
	for i, word := range words {
		if baseForm, exists := verbForms[word]; exists {
			result[i] = baseForm
		} else {
			result[i] = word
		}
	}
	return result
}

// expandSynonyms 同义词扩展层 - 将词扩展为其同义词组
func (cl *Classifier) expandSynonyms(words []string) []string {
	// 动态上限：原始词数 × 10
	maxExpandedTokens := len(words) * 10
	if maxExpandedTokens < 20 {
		maxExpandedTokens = 20 // 保底
	}

	seen := make(map[string]bool)
	var result []string

	for _, word := range words {
		if seen[word] {
			continue
		}
		seen[word] = true
		result = append(result, word)

		// 只扩展原始词，不扩展扩展出来的词（防止链式爆炸）
		for _, group := range synonymGroups {
			found := false
			for _, syn := range group {
				if syn == word {
					found = true
					break
				}
			}
			if found {
				for _, s := range group {
					if s != word && !seen[s] && len(result) < maxExpandedTokens {
						seen[s] = true
						result = append(result, s)
					}
				}
				break
			}
		}
	}
	return result
}

// levenshteinDistance 计算两个字符串的编辑距离
func (cl *Classifier) levenshteinDistance(s1, s2 string) int {
	m, n := len(s1), len(s2)
	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}

	// 使用动态规划，只使用两行来节省空间
	prev := make([]int, n+1)
	curr := make([]int, n+1)

	// 初始化第一行
	for j := 0; j <= n; j++ {
		prev[j] = j
	}

	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			curr[j] = min(prev[j]+1, // 删除
				min(curr[j-1]+1, // 插入
					prev[j-1]+cost)) // 替换
		}
		// 交换行
		prev, curr = curr, prev
	}

	return prev[n]
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// abs 返回整数的绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// dynamicThreshold 根据词长计算动态编辑距离阈值
func (cl *Classifier) dynamicThreshold(length int) int {
	switch {
	case length <= 2:
		return 0 // 必须精确匹配
	case length <= 4:
		return 1 // 允许1个误差
	default:
		return 2 // 标准阈值
	}
}

// fuzzyMatchEntity 使用编辑距离进行模糊匹配
// 根据列名长度自动选择匹配阈值
func (cl *Classifier) fuzzyMatchEntity(colName string) (string, float64) {
	length := utf8.RuneCountInString(strings.ToLower(colName))
	minScore := 0.8
	switch length {
	case 3:
		minScore = 0.67 // 允许距离1: (3-1)/3 = 0.67
	case 4:
		minScore = 0.75 // 允许距离1: (4-1)/4 = 0.75
	}
	return cl.fuzzyMatchEntityWithThreshold(colName, minScore)
}

// fuzzyMatchEntityWithThreshold 带自定义阈值的模糊匹配
func (cl *Classifier) fuzzyMatchEntityWithThreshold(colName string, minScore float64) (string, float64) {
	colNameLower := strings.ToLower(colName)
	length := utf8.RuneCountInString(colNameLower) // 按字符而非字节
	maxDistance := cl.dynamicThreshold(length)

	if maxDistance == 0 {
		return "", 0.0
	}

	bestMatch := ""
	bestScore := 0.0

	for keyword, entityType := range cl.keywordToEntity {
		keywordLen := utf8.RuneCountInString(keyword)
		if abs(keywordLen-length) > maxDistance {
			continue
		}

		distance := cl.levenshteinDistance(colNameLower, keyword)
		if distance > maxDistance {
			continue
		}

		maxLen := keywordLen
		if length > maxLen {
			maxLen = length
		}
		score := 1.0 - float64(distance)/float64(maxLen)

		if score >= minScore && score > bestScore {
			bestScore = score
			bestMatch = entityType
		}
	}

	return bestMatch, bestScore
}

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
	SampleData  []any  // 样例数据（可选，支持字符串、数值等类型，用于辅助识别实体类型）
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
	Category         string  // 分类
	Level            int     // 敏感级别 1-4
	LevelName        string  // 级别名称
	EntityType       string  // 识别的实体类型
	Description      string  // 描述
	Confidence       float64 // 置信度 0-1
	IsExact          bool    // 是否精确匹配
	SampleEntityType string  // 通过样例数据识别的实体类型（如果有）
	SampleConfidence float64 // 样例数据识别的置信度
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

	// 预先通过样例数据识别实体类型
	var sampleEntityType string
	var sampleConfidence float64
	if len(col.SampleData) > 0 {
		sampleEntityType, sampleConfidence = cl.identifyEntityBySample(col.SampleData)
	}

	// Step 1: 尝试精确匹配
	if configCol, exists := columnIndex[colNameLower]; exists {
		result := &ColumnClassificationResult{
			Category:         configCol.Category,
			Level:            configCol.Level,
			LevelName:        cl.getLevelName(configCol.Level),
			EntityType:       configCol.EntityType,
			Description:      configCol.Description,
			Confidence:       1.0,
			IsExact:          true,
			SampleEntityType: sampleEntityType,
			SampleConfidence: sampleConfidence,
		}
		// 如果样例数据识别的实体类型与配置不一致，可能需要警告
		// 这里暂时只记录，不改变分类结果
		return result
	}

	// Step 2: 尝试模糊匹配列名
	// 注意：短列名（如id）不应该随意匹配到包含它的长列名（如id_card）
	for configColName, configCol := range columnIndex {
		// 输入列名包含标准列名：如 user_name_cn 包含 user_name
		if len(configColName) >= 3 && strings.Contains(colNameLower, configColName) {
			return &ColumnClassificationResult{
				Category:         configCol.Category,
				Level:            configCol.Level,
				LevelName:        cl.getLevelName(configCol.Level),
				EntityType:       configCol.EntityType,
				Description:      configCol.Description,
				Confidence:       0.8,
				IsExact:          false,
				SampleEntityType: sampleEntityType,
				SampleConfidence: sampleConfidence,
			}
		}
		// 标准列名包含输入列名：仅当输入列名足够长（>=4字符）时才匹配
		// 避免 id 匹配到 id_card 这种情况
		if len(colNameLower) >= 4 && strings.Contains(configColName, colNameLower) {
			return &ColumnClassificationResult{
				Category:         configCol.Category,
				Level:            configCol.Level,
				LevelName:        cl.getLevelName(configCol.Level),
				EntityType:       configCol.EntityType,
				Description:      configCol.Description,
				Confidence:       0.8,
				IsExact:          false,
				SampleEntityType: sampleEntityType,
				SampleConfidence: sampleConfidence,
			}
		}
	}

	// Step 3: 通过实体类型识别进行语义匹配（基于列名、描述，样例数据作为最低优先级）
	entityType := cl.identifyEntityType(col)
	if entityType != "" {
		// 查找标准表中相同实体类型的列
		for _, configCol := range columnIndex {
			if configCol.EntityType == entityType {
				return &ColumnClassificationResult{
					Category:         configCol.Category,
					Level:            configCol.Level,
					LevelName:        cl.getLevelName(configCol.Level),
					EntityType:       entityType,
					Description:      col.Description,
					Confidence:       0.6,
					IsExact:          false,
					SampleEntityType: sampleEntityType,
					SampleConfidence: sampleConfidence,
				}
			}
		}

		// 如果标准表中没有相同实体类型，使用实体默认级别
		if entity, exists := cl.entityMap[entityType]; exists {
			return &ColumnClassificationResult{
				Category:         entity.Name,
				Level:            entity.DefaultLevel,
				LevelName:        cl.getLevelName(entity.DefaultLevel),
				EntityType:       entityType,
				Description:      entity.Description,
				Confidence:       0.4,
				IsExact:          false,
				SampleEntityType: sampleEntityType,
				SampleConfidence: sampleConfidence,
			}
		}
	}

	// Step 5: 无法识别，返回默认低级别
	return &ColumnClassificationResult{
		Category:         "未分类",
		Level:            1,
		LevelName:        cl.getLevelName(1),
		EntityType:       "unknown",
		Description:      "无法识别的列",
		Confidence:       0.1,
		IsExact:          false,
		SampleEntityType: sampleEntityType,
		SampleConfidence: sampleConfidence,
	}
}

// identifyEntityType 识别列的实体类型 - 三层语义匹配架构
// Layer 1: 形态归一化 - 处理动词时态/语态变体
// Layer 2: 编辑距离（模糊匹配）- 在形态归一化后的词上做模糊匹配
// Layer 3: 同义词扩展 - 仅用于生成精确匹配候选，降级模糊匹配
// 最后通过样例数据分析识别实体类型（最低优先级）
func (cl *Classifier) identifyEntityType(col Column) string {
	colNameLower := strings.ToLower(col.Name)
	colDescLower := strings.ToLower(col.Description)
	colTypeLower := strings.ToLower(col.DataType)

	// ========== 空值保护 ==========
	if len(colNameLower) == 0 {
		return ""
	}

	// 纯数字保护（纯数字通常不应匹配实体）
	isAllDigits := true
	for _, c := range colNameLower {
		if c < '0' || c > '9' {
			isAllDigits = false
			break
		}
	}
	if isAllDigits {
		return ""
	}

	// Step 0: 上下文判断（对于通用名称如id，结合描述和数据类型判断）
	if entityType := cl.identifyByContext(colNameLower, colDescLower, colTypeLower); entityType != "" {
		return entityType
	}

	// Step 1: 精确匹配（原始名称）
	if entityType, exists := cl.keywordToEntity[colNameLower]; exists {
		return entityType
	}

	// 分词
	words := cl.splitColumnName(colNameLower)

	// ========== Layer 1: 形态归一化 ==========
	normalizedWords := cl.normalizeMorphology(words)
	normalizedName := strings.Join(normalizedWords, "_")

	// 标准化后的精确匹配
	if entityType, exists := cl.keywordToEntity[normalizedName]; exists {
		return entityType
	}
	// 标准化后的分词匹配
	for _, word := range normalizedWords {
		if entityType, exists := cl.keywordToEntity[word]; exists {
			return entityType
		}
	}

	// ========== Layer 2: 编辑距离（模糊匹配）==========
	// 在形态归一化后的词上做模糊匹配
	if entityType, score := cl.fuzzyMatchEntity(normalizedName); entityType != "" && score >= 0.8 {
		return entityType
	}
	for _, word := range normalizedWords {
		if entityType, score := cl.fuzzyMatchEntity(word); entityType != "" && score >= 0.8 {
			return entityType
		}
	}

	// ========== Layer 3: 同义词扩展 ==========
	expandedWords := cl.expandSynonyms(normalizedWords)

	// 3a: 同义词精确匹配
	for _, word := range expandedWords {
		if entityType, exists := cl.keywordToEntity[word]; exists {
			return entityType
		}
	}

	// 3b: 同义词降级模糊匹配（降低阈值到0.65）
	for _, word := range expandedWords {
		if entityType, score := cl.fuzzyMatchEntityWithThreshold(word, 0.65); entityType != "" && score >= 0.65 {
			return entityType
		}
		// 相邻词组合
		for _, other := range expandedWords {
			if word != other {
				combo := word + "_" + other
				if entityType, exists := cl.keywordToEntity[combo]; exists {
					return entityType
				}
			}
		}
	}

	// 保留原有的描述匹配逻辑
	descWords := cl.splitDescriptionToWords(colDescLower)
	for _, descWord := range descWords {
		if entityType, exists := cl.keywordToEntity[descWord]; exists {
			return entityType
		}
	}

	// 保留原有的关键词遍历匹配
	for _, entity := range cl.entityConfig.Entities {
		for _, keyword := range entity.Keywords {
			keywordLower := strings.ToLower(keyword)
			for _, word := range normalizedWords {
				if word == keywordLower {
					return entity.Type
				}
			}
		}
	}

	// Step 6: 通过样例数据识别实体类型（最低优先级）
	if len(col.SampleData) > 0 {
		if sampleEntityType, confidence := cl.identifyEntityBySample(col.SampleData); sampleEntityType != "" && confidence >= 0.6 {
			return sampleEntityType
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

	// 去重
	seen := make(map[string]bool)
	var uniqueWords []string
	for _, w := range result {
		if !seen[w] {
			seen[w] = true
			uniqueWords = append(uniqueWords, w)
		}
	}

	return uniqueWords
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

// ==================== 样例数据分析 ====================

// samplePattern 样例数据模式定义
type samplePattern struct {
	entityType string
	pattern    *regexp.Regexp
	validator  func(string) bool // 可选的额外验证函数
}

// 预编译的正则表达式模式
var samplePatterns = []samplePattern{
	// 手机号：1开头的11位数字
	{entityType: "phone", pattern: regexp.MustCompile(`^1[3-9]\d{9}$`)},
	// 固定电话：区号-号码格式
	{entityType: "phone", pattern: regexp.MustCompile(`^0\d{2,3}-?\d{7,8}$`)},
	// 身份证号：18位，最后一位可能是X
	{entityType: "id_card", pattern: regexp.MustCompile(`^\d{17}[\dXx]$`), validator: validateIDCard},
	// 15位老身份证
	{entityType: "id_card", pattern: regexp.MustCompile(`^\d{15}$`)},
	// 邮箱
	{entityType: "email", pattern: regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)},
	// 银行卡号：16-19位数字
	{entityType: "bank_account", pattern: regexp.MustCompile(`^\d{16,19}$`)},
	// IP地址（IPv4）
	{entityType: "ip_address", pattern: regexp.MustCompile(`^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$`)},
	// MAC地址
	{entityType: "mac_address", pattern: regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)},
	// 车牌号（中国大陆）- 使用 Unicode 范围 \p{Han} 匹配汉字
	{entityType: "license_plate", pattern: regexp.MustCompile(`^\p{Han}[A-Z][A-Z0-9]{5,6}$`)},
	// 邮政编码（中国）
	{entityType: "postal_code", pattern: regexp.MustCompile(`^\d{6}$`)},
	// 护照号（中国）
	{entityType: "passport", pattern: regexp.MustCompile(`^[EeKkGgDdSsPpHh]\d{8}$`)},
	// 军官证 - 使用 \p{Han} 匹配汉字
	{entityType: "military_id", pattern: regexp.MustCompile(`^\p{Han}{1,2}\d{7,8}$`)},
	// 社会统一信用代码
	{entityType: "unified_social_credit_code", pattern: regexp.MustCompile(`^[0-9A-Z]{18}$`)},
	// 日期格式 YYYY-MM-DD
	{entityType: "date", pattern: regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}$`)},
	// 日期时间格式
	{entityType: "datetime", pattern: regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}[\sT]\d{2}:\d{2}(:\d{2})?`)},
	// 时间格式 HH:MM:SS
	{entityType: "time", pattern: regexp.MustCompile(`^\d{2}:\d{2}(:\d{2})?$`)},
	// URL
	{entityType: "url", pattern: regexp.MustCompile(`^https?://[^\s]+$`)},
	// 中文姓名（2-4个汉字）- 使用 \p{Han} 匹配汉字
	{entityType: "name", pattern: regexp.MustCompile(`^\p{Han}{2,4}$`), validator: validateChineseName},
	// 经纬度
	{entityType: "coordinate", pattern: regexp.MustCompile(`^-?\d{1,3}\.\d+,\s*-?\d{1,3}\.\d+$`)},
	// 金额（带小数点和可选货币符号）
	{entityType: "amount", pattern: regexp.MustCompile(`^[¥$€£]?\d{1,3}(,\d{3})*(\.\d{1,2})?$`)},
}

// identifyEntityBySample 通过样例数据识别实体类型
// 返回识别的实体类型和置信度
func (cl *Classifier) identifyEntityBySample(samples []any) (string, float64) {
	if len(samples) == 0 {
		return "", 0
	}

	// 统计每种实体类型的匹配数量
	typeMatchCount := make(map[string]int)
	validSamples := 0

	for _, sampleRaw := range samples {
		// 将样例数据转换为字符串
		sample := convertToString(sampleRaw)
		sample = strings.TrimSpace(sample)
		if sample == "" || sample == "NULL" || sample == "null" || sample == "nil" || sample == "<nil>" {
			continue
		}
		validSamples++

		for _, sp := range samplePatterns {
			if sp.pattern.MatchString(sample) {
				// 如果有额外验证函数，执行验证
				if sp.validator != nil {
					if sp.validator(sample) {
						typeMatchCount[sp.entityType]++
					}
				} else {
					typeMatchCount[sp.entityType]++
				}
				break // 每个样例只匹配第一个模式
			}
		}
	}

	if validSamples == 0 {
		return "", 0
	}

	// 找出匹配最多的实体类型
	var bestType string
	var bestCount int
	for entityType, count := range typeMatchCount {
		if count > bestCount {
			bestCount = count
			bestType = entityType
		}
	}

	if bestType == "" {
		return "", 0
	}

	// 计算置信度：匹配数量 / 有效样例数量
	confidence := float64(bestCount) / float64(validSamples)
	return bestType, confidence
}

// validateIDCard 验证身份证号码校验位
func validateIDCard(id string) bool {
	if len(id) != 18 {
		return false
	}
	// 权重因子
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	// 校验码对照表
	checkCodes := "10X98765432"

	sum := 0
	for i := 0; i < 17; i++ {
		digit := int(id[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}
		sum += digit * weights[i]
	}

	expectedCheck := checkCodes[sum%11]
	actualCheck := id[17]
	if actualCheck >= 'a' && actualCheck <= 'z' {
		actualCheck -= 32 // 转大写
	}

	return expectedCheck == actualCheck
}

// validateChineseName 验证中文姓名
// 排除一些明显不是姓名的中文字符串
func validateChineseName(name string) bool {
	// 常见的非姓名词汇
	nonNameWords := []string{
		"有限", "公司", "集团", "银行", "医院", "学校", "大学", "中心",
		"部门", "科技", "网络", "信息", "管理", "服务", "投资", "发展",
		"北京", "上海", "广州", "深圳", "杭州", "成都", "武汉", "西安",
	}
	for _, word := range nonNameWords {
		if strings.Contains(name, word) {
			return false
		}
	}
	return true
}

// convertToString 将任意类型转换为字符串
// 支持常见的数据库字段类型
func convertToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.Itoa(val)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		// 尝试使用 fmt.Sprint 作为最后手段
		return fmt.Sprint(v)
	}
}

// IdentifyEntityBySampleData 公开的样例数据识别接口
// 可用于单独分析样例数据的实体类型
func (cl *Classifier) IdentifyEntityBySampleData(samples []any) (entityType string, confidence float64) {
	return cl.identifyEntityBySample(samples)
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
