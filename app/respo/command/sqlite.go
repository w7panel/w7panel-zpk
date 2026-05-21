package command

import (
	"database/sql"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
	"log"
	"log/slog"
	"strings"
	"time"
)

type Sqlite struct {
	console.Abstract
}

func (pack Sqlite) GetName() string {
	return "respo:sqlite"
}

func (pack Sqlite) GetDescription() string {
	return "respo command"
}

func (pack Sqlite) Handle(cmd *cobra.Command, args []string) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", facade.GetConfig().GetString("database.mysql.user_name"), facade.GetConfig().GetString("database.mysql.password"), facade.GetConfig().GetString("database.mysql.host"), facade.GetConfig().GetString("database.mysql.port"), facade.GetConfig().GetString("database.mysql.db_name"))
	mysqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("MySQL连接失败: ", err)
	}
	defer mysqlDB.Close()

	// 2. 连接 SQLite
	sqliteDB, err := sql.Open("sqlite3", facade.GetConfig().GetString("database.default.db_name"))
	if err != nil {
		log.Fatal("SQLite连接失败: ", err)
	}
	defer sqliteDB.Close()

	// 3. 获取 MySQL 表列表
	tables, err := mysqlDB.Query("SHOW TABLES")
	if err != nil {
		log.Fatal("获取表列表失败: ", err)
	}
	defer tables.Close()

	// 4. 遍历每张表迁移数据
	for tables.Next() {
		var tableName string
		if err := tables.Scan(&tableName); err != nil {
			log.Printf("读取表名失败: %v", err)
			continue
		}

		if strings.Contains(tableName, "registry") {
			continue
		}

		if err := migrateTable(mysqlDB, sqliteDB, tableName); err != nil {
			log.Printf("迁移表 %s 失败: %v", tableName, err)
		} else {
			log.Printf("✅ 表 %s 迁移完成", tableName)
		}
	}
}

func migrateTable(mysqlDB, sqliteDB *sql.DB, tableName string) error {
	colTypes, err := getMySQLColumnTypes(mysqlDB, tableName)
	if err != nil {
		return err
	}

	// 3. 分页读取 MySQL 数据
	pageSize := 100
	offset := 0
	for {
		rows, err := mysqlDB.Query(fmt.Sprintf(
			"SELECT * FROM %s LIMIT %d OFFSET %d",
			tableName, pageSize, offset,
		))
		if err != nil {
			return err
		}

		// 4. 获取列信息
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return err
		}

		// 5. 批量插入 SQLite
		tx, _ := sqliteDB.Begin()
		stmt, _ := tx.Prepare(buildInsertSQL(tableName, columns))
		defer stmt.Close()

		var rowCount int
		for rows.Next() {
			// 动态创建接收值的切片
			values := make([]interface{}, len(columns))
			for i := range values {
				// 根据列类型创建适当的接收器
				switch colTypes[i] {
				// 处理所有文本类型（包括 LONGTEXT）
				case "CHAR", "VARCHAR", "TEXT", "LONGTEXT", "MEDIUMTEXT", "TINYTEXT",
					"ENUM", "SET", "JSON", "DATE", "DATETIME", "TIMESTAMP", "TIME":
					values[i] = new(sql.NullString)
				case "BLOB", "LONGBLOB", "MEDIUMBLOB", "TINYBLOB", "BINARY", "VARBINARY":
					values[i] = new([]byte)
				case "INT", "INTEGER", "BIGINT", "TINYINT", "SMALLINT", "MEDIUMINT":
					values[i] = new(sql.NullInt64)
				case "FLOAT", "DOUBLE", "DECIMAL", "NUMERIC":
					values[i] = new(sql.NullFloat64)
				case "BIT":
					values[i] = new([]byte) // BIT 类型作为二进制处理
				case "YEAR":
					values[i] = new(sql.NullInt64) // YEAR 作为整数处理
				default:
					values[i] = new(interface{})
					log.Printf("警告: 表 %s 列 %d (%s) 使用默认类型处理",
						tableName, i, colTypes[i])
				}
			}

			if err := rows.Scan(values...); err != nil {
				log.Printf("行扫描失败: %v", err)
				continue
			}

			// 转换为实际插入值
			insertValues := make([]interface{}, len(values))
			for i, val := range values {
				switch v := val.(type) {
				case *sql.NullString:
					if v.Valid {
						// 特殊处理：如果是时间类型字段，转换为标准格式
						if isTimeType(colTypes[i]) {
							if t, err := parseMySQLTime(v.String); err == nil {
								insertValues[i] = t.Format(time.RFC3339)
							} else {
								insertValues[i] = v.String
							}
						} else {
							insertValues[i] = v.String
						}
					} else {
						insertValues[i] = nil
					}
				case *[]byte:
					// 特殊处理：BIT 类型转换为整数
					if colTypes[i] == "BIT" {
						insertValues[i] = convertBitToInt(*v)
					} else {
						insertValues[i] = *v
					}
				case *sql.NullInt64:
					if v.Valid {
						insertValues[i] = v.Int64
					} else {
						insertValues[i] = nil
					}
				case *sql.NullFloat64:
					if v.Valid {
						insertValues[i] = v.Float64
					} else {
						insertValues[i] = nil
					}
				default:
					insertValues[i] = *(val.(*interface{}))
				}
			}
			if _, err := stmt.Exec(insertValues...); err != nil {
				log.Printf("插入失败: %v", err)
			}

			rowCount++
		}

		if err := tx.Commit(); err != nil {
			slog.Error("事务提交失败", "err", err)
		}
		rows.Close()

		// 终止条件
		if rowCount < pageSize {
			break
		}
		offset += pageSize
	}
	return nil
}

func convertBitToInt(bitData []byte) int64 {
	var result int64
	for _, b := range bitData {
		result = (result << 8) | int64(b)
	}
	return result
}

func buildInsertSQL(table string, columns []string) string {
	placeholders := strings.Repeat("?,", len(columns))
	return fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`,
		table,
		`"`+strings.Join(columns, `","`)+`"`,
		placeholders[:len(placeholders)-1],
	)
}

func getMySQLColumnTypes(db *sql.DB, table string) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf(
		`SELECT 
			COLUMN_TYPE, 
			DATA_TYPE 
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = '%s' 
		ORDER BY ORDINAL_POSITION`, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var columnType, dataType string
		if err := rows.Scan(&columnType, &dataType); err != nil {
			return nil, err
		}

		// 优先使用更具体的 COLUMN_TYPE
		colType := strings.ToUpper(columnType)
		dataType = strings.ToUpper(dataType)

		// 特殊处理：识别 LONGTEXT 等类型
		if strings.Contains(colType, "LONGTEXT") {
			types = append(types, "LONGTEXT")
		} else if strings.Contains(colType, "LONGBLOB") {
			types = append(types, "LONGBLOB")
		} else {
			types = append(types, dataType)
		}
	}
	return types, nil
}

func isTimeType(dataType string) bool {
	switch dataType {
	case "DATE", "DATETIME", "TIMESTAMP", "TIME":
		return true
	default:
		return false
	}
}

// 辅助函数：解析MySQL时间格式
func parseMySQLTime(timeStr string) (time.Time, error) {
	// MySQL常见时间格式
	formats := []string{
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析的时间格式: %s", timeStr)
}
