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
	"testing"
)

// loadConfigs 使用内置默认配置
func loadConfigs(t *testing.T) (string, string) {
	// 使用 embed 嵌入的默认配置
	return DefaultConfig, DefaultEntity
}

// TestClassifier_UserInfoTable 测试用户信息表分类
func TestClassifier_UserInfoTable(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	// 构建用户信息表输入
	input := &Table{
		Name:        "t_user_info",
		Description: "用户基本信息表，存储用户账户信息",
		Columns: []Column{
			{Name: "id", Description: "主键ID", DataType: "bigint"},
			{Name: "user_name", Description: "用户登录名", DataType: "varchar(64)"},
			{Name: "real_name", Description: "用户真实姓名", DataType: "varchar(32)"},
			{Name: "mobile", Description: "手机号码", DataType: "varchar(11)"},
			{Name: "email", Description: "电子邮箱", DataType: "varchar(128)"},
			{Name: "id_card_no", Description: "身份证号", DataType: "varchar(18)"},
			{Name: "password_hash", Description: "密码哈希值", DataType: "varchar(256)"},
			{Name: "avatar_url", Description: "头像链接", DataType: "varchar(512)"},
			{Name: "gender", Description: "性别", DataType: "tinyint"},
			{Name: "birth_date", Description: "出生日期", DataType: "date"},
			{Name: "home_address", Description: "家庭住址", DataType: "varchar(256)"},
			{Name: "create_time", Description: "创建时间", DataType: "datetime"},
			{Name: "update_time", Description: "更新时间", DataType: "datetime"},
			{Name: "is_active", Description: "是否激活", DataType: "tinyint"},
		},
	}

	result, err := classifier.Classify(input)
	if err != nil {
		t.Fatalf("分类失败: %v", err)
	}

	// 验证表级别分类
	t.Logf("=== 用户信息表分类结果 ===")
	t.Logf("匹配的标准表: %s (相似度: %.2f)", result.MatchedTable, result.MatchScore)
	t.Logf("表分类: %s", result.Classification)
	t.Logf("表级别: %d - %s", result.Table.Level, result.Table.LevelName)

	// 验证敏感字段
	sensitiveFields := []string{"mobile", "id_card_no", "password_hash"}
	for _, field := range sensitiveFields {
		if col, exists := result.Columns[field]; exists {
			t.Logf("字段 [%s] - 级别: %d, 类型: %s, 精确匹配: %v",
				field, col.Level, col.EntityType, col.IsExact)
			if col.Level < 3 {
				t.Errorf("敏感字段 %s 级别应该 >= 3，实际: %d", field, col.Level)
			}
		}
	}

	// 输出所有列分类结果
	t.Logf("\n=== 列分类详情 ===")
	for colName, colResult := range result.Columns {
		t.Logf("%-15s | 级别: %d | 分类: %-10s | 实体: %-20s | 置信度: %.2f | 精确: %v",
			colName, colResult.Level, colResult.Category, colResult.EntityType,
			colResult.Confidence, colResult.IsExact)
	}

	// 验证未匹配列
	t.Logf("\n未精确匹配的列: %v", result.UnmatchedCols)
}

// TestClassifier_CompanyInfoTable 测试企业信息表分类
func TestClassifier_CompanyInfoTable(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	// 构建企业信息表输入
	input := &Table{
		Name:        "enterprise_info",
		Description: "企业基本信息表",
		Columns: []Column{
			{Name: "enterprise_id", Description: "企业ID", DataType: "bigint"},
			{Name: "corp_name", Description: "企业名称", DataType: "varchar(128)"},
			{Name: "credit_code", Description: "统一社会信用代码", DataType: "varchar(18)"},
			{Name: "legal_person", Description: "法定代表人", DataType: "varchar(32)"},
			{Name: "reg_capital", Description: "注册资本", DataType: "decimal(18,2)"},
			{Name: "industry_type", Description: "行业类型", DataType: "varchar(64)"},
			{Name: "reg_address", Description: "注册地址", DataType: "varchar(256)"},
			{Name: "contact_tel", Description: "联系电话", DataType: "varchar(20)"},
			{Name: "contact_mail", Description: "联系邮箱", DataType: "varchar(128)"},
			{Name: "setup_date", Description: "成立日期", DataType: "date"},
			{Name: "biz_status", Description: "经营状态", DataType: "varchar(16)"},
			{Name: "bank_account_no", Description: "对公银行账号", DataType: "varchar(32)"},
			{Name: "opening_bank", Description: "开户银行", DataType: "varchar(64)"},
		},
	}

	result, err := classifier.Classify(input)
	if err != nil {
		t.Fatalf("分类失败: %v", err)
	}

	t.Logf("=== 企业信息表分类结果 ===")
	t.Logf("匹配的标准表: %s (相似度: %.2f)", result.MatchedTable, result.MatchScore)
	t.Logf("表分类: %s", result.Classification)
	t.Logf("表级别: %d - %s", result.Table.Level, result.Table.LevelName)

	// 验证银行账号应该是高敏感
	if col, exists := result.Columns["bank_account_no"]; exists {
		t.Logf("银行账号字段级别: %d, 实体类型: %s", col.Level, col.EntityType)
		if col.Level < 4 {
			t.Errorf("银行账号字段级别应该为4，实际: %d", col.Level)
		}
	}

	t.Logf("\n=== 列分类详情 ===")
	for colName, colResult := range result.Columns {
		t.Logf("%-18s | 级别: %d | 分类: %-12s | 实体: %-20s | 置信度: %.2f",
			colName, colResult.Level, colResult.Category, colResult.EntityType, colResult.Confidence)
	}
}

// TestClassifier_OrderInfoTable 测试交易信息表分类
func TestClassifier_OrderInfoTable(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	// 构建订单信息表输入
	input := &Table{
		Name:        "t_order",
		Description: "用户订单交易表",
		Columns: []Column{
			{Name: "order_id", Description: "订单ID", DataType: "bigint"},
			{Name: "order_sn", Description: "订单编号", DataType: "varchar(32)"},
			{Name: "buyer_id", Description: "买家用户ID", DataType: "bigint"},
			{Name: "seller_id", Description: "卖家用户ID", DataType: "bigint"},
			{Name: "goods_id", Description: "商品ID", DataType: "bigint"},
			{Name: "goods_name", Description: "商品名称", DataType: "varchar(128)"},
			{Name: "buy_count", Description: "购买数量", DataType: "int"},
			{Name: "goods_price", Description: "商品单价", DataType: "decimal(10,2)"},
			{Name: "order_amount", Description: "订单总金额", DataType: "decimal(12,2)"},
			{Name: "pay_type", Description: "支付方式", DataType: "varchar(16)"},
			{Name: "pay_status", Description: "支付状态", DataType: "tinyint"},
			{Name: "delivery_addr", Description: "收货地址", DataType: "varchar(256)"},
			{Name: "receiver", Description: "收货人姓名", DataType: "varchar(32)"},
			{Name: "receiver_mobile", Description: "收货人手机号", DataType: "varchar(11)"},
			{Name: "order_state", Description: "订单状态", DataType: "tinyint"},
			{Name: "order_date", Description: "下单日期", DataType: "datetime"},
			{Name: "pay_time", Description: "支付时间", DataType: "datetime"},
			{Name: "order_remark", Description: "订单备注", DataType: "varchar(512)"},
		},
	}

	result, err := classifier.Classify(input)
	if err != nil {
		t.Fatalf("分类失败: %v", err)
	}

	t.Logf("=== 订单信息表分类结果 ===")
	t.Logf("匹配的标准表: %s (相似度: %.2f)", result.MatchedTable, result.MatchScore)
	t.Logf("表分类: %s", result.Classification)
	t.Logf("表级别: %d - %s", result.Table.Level, result.Table.LevelName)

	// 验证收货人手机号应该是高敏感
	if col, exists := result.Columns["receiver_mobile"]; exists {
		t.Logf("收货人手机号字段级别: %d, 实体类型: %s", col.Level, col.EntityType)
		if col.Level < 4 {
			t.Errorf("收货人手机号字段级别应该为4，实际: %d", col.Level)
		}
	}

	// 验证订单金额敏感级别
	if col, exists := result.Columns["order_amount"]; exists {
		t.Logf("订单金额字段级别: %d, 实体类型: %s", col.Level, col.EntityType)
		if col.Level < 2 {
			t.Errorf("订单金额字段级别应该 >= 2，实际: %d", col.Level)
		}
	}

	t.Logf("\n=== 列分类详情 ===")
	for colName, colResult := range result.Columns {
		t.Logf("%-18s | 级别: %d | 分类: %-12s | 实体: %-20s | 置信度: %.2f",
			colName, colResult.Level, colResult.Category, colResult.EntityType, colResult.Confidence)
	}
}

// TestClassifier_PaymentRecordTable 测试支付记录表分类
func TestClassifier_PaymentRecordTable(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	// 构建支付记录表输入
	input := &Table{
		Name:        "payment_log",
		Description: "支付交易记录表",
		Columns: []Column{
			{Name: "pay_id", Description: "支付ID", DataType: "bigint"},
			{Name: "order_id", Description: "关联订单ID", DataType: "bigint"},
			{Name: "payer_id", Description: "支付用户ID", DataType: "bigint"},
			{Name: "pay_amount", Description: "支付金额", DataType: "decimal(12,2)"},
			{Name: "pay_channel", Description: "支付渠道", DataType: "varchar(32)"},
			{Name: "card_no", Description: "银行卡号", DataType: "varchar(19)"},
			{Name: "trans_no", Description: "交易流水号", DataType: "varchar(64)"},
			{Name: "pay_state", Description: "支付状态", DataType: "tinyint"},
			{Name: "pay_time", Description: "支付时间", DataType: "datetime"},
		},
	}

	result, err := classifier.Classify(input)
	if err != nil {
		t.Fatalf("分类失败: %v", err)
	}

	t.Logf("=== 支付记录表分类结果 ===")
	t.Logf("匹配的标准表: %s (相似度: %.2f)", result.MatchedTable, result.MatchScore)
	t.Logf("表分类: %s", result.Classification)
	t.Logf("表级别: %d - %s", result.Table.Level, result.Table.LevelName)

	// 验证银行卡号应该是最高敏感级别
	if col, exists := result.Columns["card_no"]; exists {
		t.Logf("银行卡号字段级别: %d, 实体类型: %s", col.Level, col.EntityType)
		if col.Level < 4 {
			t.Errorf("银行卡号字段级别应该为4，实际: %d", col.Level)
		}
	}

	t.Logf("\n=== 列分类详情 ===")
	for colName, colResult := range result.Columns {
		t.Logf("%-15s | 级别: %d | 分类: %-12s | 实体: %-20s | 置信度: %.2f",
			colName, colResult.Level, colResult.Category, colResult.EntityType, colResult.Confidence)
	}
}

// TestClassifier_EmployeeInfoTable 测试员工信息表分类
func TestClassifier_EmployeeInfoTable(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	// 构建员工信息表输入
	input := &Table{
		Name:        "hr_employee",
		Description: "员工人事信息表",
		Columns: []Column{
			{Name: "emp_id", Description: "员工ID", DataType: "bigint"},
			{Name: "emp_name", Description: "员工姓名", DataType: "varchar(32)"},
			{Name: "id_number", Description: "身份证号码", DataType: "varchar(18)"},
			{Name: "mobile_phone", Description: "手机号码", DataType: "varchar(11)"},
			{Name: "work_email", Description: "工作邮箱", DataType: "varchar(128)"},
			{Name: "dept_name", Description: "部门名称", DataType: "varchar(64)"},
			{Name: "job_title", Description: "职位名称", DataType: "varchar(64)"},
			{Name: "base_salary", Description: "基本工资", DataType: "decimal(10,2)"},
			{Name: "salary_card", Description: "工资卡号", DataType: "varchar(19)"},
			{Name: "hire_date", Description: "入职日期", DataType: "date"},
			{Name: "education_level", Description: "学历", DataType: "varchar(16)"},
			{Name: "emp_status", Description: "在职状态", DataType: "tinyint"},
		},
	}

	result, err := classifier.Classify(input)
	if err != nil {
		t.Fatalf("分类失败: %v", err)
	}

	t.Logf("=== 员工信息表分类结果 ===")
	t.Logf("匹配的标准表: %s (相似度: %.2f)", result.MatchedTable, result.MatchScore)
	t.Logf("表分类: %s", result.Classification)
	t.Logf("表级别: %d - %s", result.Table.Level, result.Table.LevelName)

	// 验证薪资相关字段应该是高敏感
	salaryFields := []string{"base_salary", "salary_card"}
	for _, field := range salaryFields {
		if col, exists := result.Columns[field]; exists {
			t.Logf("字段 [%s] - 级别: %d, 类型: %s", field, col.Level, col.EntityType)
			if col.Level < 4 {
				t.Errorf("薪资相关字段 %s 级别应该为4，实际: %d", field, col.Level)
			}
		}
	}

	t.Logf("\n=== 列分类详情 ===")
	for colName, colResult := range result.Columns {
		t.Logf("%-18s | 级别: %d | 分类: %-12s | 实体: %-20s | 置信度: %.2f",
			colName, colResult.Level, colResult.Category, colResult.EntityType, colResult.Confidence)
	}
}

// TestClassifier_BehaviorLogTable 测试用户行为日志表分类
func TestClassifier_BehaviorLogTable(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	// 构建用户行为日志表输入
	input := &Table{
		Name:        "user_action_log",
		Description: "用户行为埋点日志表",
		Columns: []Column{
			{Name: "log_id", Description: "日志ID", DataType: "bigint"},
			{Name: "uid", Description: "用户ID", DataType: "bigint"},
			{Name: "action_name", Description: "行为名称", DataType: "varchar(64)"},
			{Name: "page_path", Description: "页面路径", DataType: "varchar(256)"},
			{Name: "refer_page", Description: "来源页面", DataType: "varchar(256)"},
			{Name: "device_uuid", Description: "设备唯一ID", DataType: "varchar(64)"},
			{Name: "client_ip", Description: "客户端IP", DataType: "varchar(45)"},
			{Name: "ua_string", Description: "User-Agent", DataType: "varchar(512)"},
			{Name: "geo_lat", Description: "地理纬度", DataType: "decimal(10,7)"},
			{Name: "geo_lng", Description: "地理经度", DataType: "decimal(10,7)"},
			{Name: "event_ts", Description: "事件时间戳", DataType: "bigint"},
		},
	}

	result, err := classifier.Classify(input)
	if err != nil {
		t.Fatalf("分类失败: %v", err)
	}

	t.Logf("=== 用户行为日志表分类结果 ===")
	t.Logf("匹配的标准表: %s (相似度: %.2f)", result.MatchedTable, result.MatchScore)
	t.Logf("表分类: %s", result.Classification)
	t.Logf("表级别: %d - %s", result.Table.Level, result.Table.LevelName)

	// 验证位置信息应该是敏感级别
	locationFields := []string{"geo_lat", "geo_lng"}
	for _, field := range locationFields {
		if col, exists := result.Columns[field]; exists {
			t.Logf("字段 [%s] - 级别: %d, 类型: %s", field, col.Level, col.EntityType)
			if col.Level < 3 {
				t.Errorf("位置信息字段 %s 级别应该 >= 3，实际: %d", field, col.Level)
			}
		}
	}

	t.Logf("\n=== 列分类详情 ===")
	for colName, colResult := range result.Columns {
		t.Logf("%-15s | 级别: %d | 分类: %-12s | 实体: %-20s | 置信度: %.2f",
			colName, colResult.Level, colResult.Category, colResult.EntityType, colResult.Confidence)
	}
}

// TestClassifier_ResultJSON 测试结果JSON序列化
func TestClassifier_ResultJSON(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	input := &Table{
		Name:        "user_basic_info",
		Description: "用户基本信息",
		Columns: []Column{
			{Name: "user_id", Description: "用户ID", DataType: "bigint"},
			{Name: "phone", Description: "手机号", DataType: "varchar(11)"},
			{Name: "email", Description: "邮箱", DataType: "varchar(128)"},
		},
	}

	result, err := classifier.Classify(input)
	if err != nil {
		t.Fatalf("分类失败: %v", err)
	}

	// 序列化为JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("JSON序列化失败: %v", err)
	}

	t.Logf("分类结果JSON:\n%s", string(jsonData))
}

// TestClassifier_GetCategories 测试获取分类列表
func TestClassifier_GetCategories(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	categories := classifier.GetTableCategories()
	t.Logf("所有表分类: %v", categories)

	if len(categories) == 0 {
		t.Error("应该至少有一个表分类")
	}
}

// TestClassifier_GetEntityTypes 测试获取实体类型列表
func TestClassifier_GetEntityTypes(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	entityTypes := classifier.GetEntityTypes()
	t.Logf("所有实体类型: %v", entityTypes)

	if len(entityTypes) == 0 {
		t.Error("应该至少有一个实体类型")
	}
}

// TestClassifier_ClassifyBatch 测试批量分类
func TestClassifier_ClassifyBatch(t *testing.T) {
	configData, entityData := loadConfigs(t)

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		t.Fatalf("创建分类器失败: %v", err)
	}

	// 构建多个测试表
	tables := []*Table{
		{
			Name:        "t_user",
			Description: "用户表",
			Columns: []Column{
				{Name: "id", Description: "主键", DataType: "bigint"},
				{Name: "name", Description: "姓名", DataType: "varchar"},
				{Name: "phone", Description: "手机号", DataType: "varchar"},
			},
		},
		{
			Name:        "t_order",
			Description: "订单表",
			Columns: []Column{
				{Name: "order_id", Description: "订单ID", DataType: "bigint"},
				{Name: "amount", Description: "金额", DataType: "decimal"},
			},
		},
		{
			Name:        "t_payment",
			Description: "支付表",
			Columns: []Column{
				{Name: "pay_id", Description: "支付ID", DataType: "bigint"},
				{Name: "card_no", Description: "银行卡号", DataType: "varchar"},
			},
		},
	}

	// 测试批量处理
	results, err := classifier.ClassifyBatch(tables, 0)
	if err != nil {
		t.Fatalf("批量分类失败: %v", err)
	}

	if len(results) != len(tables) {
		t.Errorf("结果数量不匹配: 期望 %d, 实际 %d", len(tables), len(results))
	}

	t.Log("=== 批量分类结果 ===")
	for i, result := range results {
		t.Logf("表 %d [%s]: 分类=%s, 级别=%d", i+1, tables[i].Name, result.Classification, result.Table.Level)
	}

	// 验证结果顺序正确
	if results[0].MatchedTable == "" {
		t.Error("第一个结果应该有匹配的表")
	}
	if results[1].MatchedTable == "" {
		t.Error("第二个结果应该有匹配的表")
	}
	if results[2].MatchedTable == "" {
		t.Error("第三个结果应该有匹配的表")
	}
}

// BenchmarkClassify 性能基准测试
func BenchmarkClassify(b *testing.B) {
	configData, entityData := DefaultConfig, DefaultEntity

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		b.Fatalf("创建分类器失败: %v", err)
	}

	input := &Table{
		Name:        "t_user_info",
		Description: "用户基本信息表",
		Columns: []Column{
			{Name: "id", Description: "主键ID", DataType: "bigint"},
			{Name: "user_name", Description: "用户名", DataType: "varchar"},
			{Name: "mobile", Description: "手机号", DataType: "varchar"},
			{Name: "email", Description: "邮箱", DataType: "varchar"},
			{Name: "id_card_no", Description: "身份证号", DataType: "varchar"},
			{Name: "password_hash", Description: "密码哈希", DataType: "varchar"},
			{Name: "create_time", Description: "创建时间", DataType: "datetime"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = classifier.Classify(input)
	}
}

// BenchmarkClassifyBatch 批量处理性能基准测试
func BenchmarkClassifyBatch(b *testing.B) {
	configData, entityData := DefaultConfig, DefaultEntity

	classifier, err := NewClassifier(configData, entityData)
	if err != nil {
		b.Fatalf("创建分类器失败: %v", err)
	}

	// 创建100个测试表
	tables := make([]*Table, 100)
	for i := 0; i < 100; i++ {
		tables[i] = &Table{
			Name:        "t_user_info",
			Description: "用户基本信息表",
			Columns: []Column{
				{Name: "id", Description: "主键ID", DataType: "bigint"},
				{Name: "user_name", Description: "用户名", DataType: "varchar"},
				{Name: "mobile", Description: "手机号", DataType: "varchar"},
				{Name: "email", Description: "邮箱", DataType: "varchar"},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = classifier.ClassifyBatch(tables, 4)
	}
}
