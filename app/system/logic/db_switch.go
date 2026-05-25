package logic

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	DBModeSQLite = "sqlite"
	DBModeMySQL  = "mysql"

	DBSwitchStatusIdle    = "idle"
	DBSwitchStatusRunning = "running"
	DBSwitchStatusFailed  = "failed"
	DBSwitchStatusSuccess = "success"

	mysqlSessionTimezone = "+08:00"
	mysqlLocationName    = "Asia/Shanghai"
)

type DBSwitchState struct {
	Mode         string               `json:"mode"`
	SwitchStatus string               `json:"switch_status"`
	Error        string               `json:"error"`
	StartedAt    string               `json:"started_at"`
	FinishedAt   string               `json:"finished_at"`
	MySQL        *DBSwitchMySQLConfig `json:"mysql,omitempty"`
}

type DBSwitchMySQLConfig struct {
	Host      string `json:"host"`
	Port      string `json:"port"`
	UserName  string `json:"user_name"`
	Password  string `json:"password"`
	DBName    string `json:"db_name"`
	Charset   string `json:"charset"`
	Prefix    string `json:"prefix"`
	SchemaSQL string `json:"-"`
}

type DBSwitchManager struct {
	mu sync.Mutex
}

type DBMigrationSource struct {
	Mode  string
	MySQL *DBSwitchMySQLConfig
}

var dbSwitchRuntimeState struct {
	mu      sync.Mutex
	running bool
}

func NewDBSwitchManager() *DBSwitchManager {
	return &DBSwitchManager{}
}

func (m *DBSwitchManager) GetState() (DBSwitchState, error) {
	return LoadDBSwitchStateByConfig(facade.GetConfig())
}

func (m *DBSwitchManager) StartSwitchToMySQL(mysqlConfig DBSwitchMySQLConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.GetState()
	if err != nil {
		return err
	}
	if state.SwitchStatus == DBSwitchStatusRunning {
		return errors.New("数据库迁移任务正在执行")
	}

	source, err := resolveDBMigrationSource(state)
	if err != nil {
		return err
	}
	if source.Mode == DBModeMySQL && isSameMySQLDatabase(*source.MySQL, mysqlConfig) {
		return errors.New("目标 MySQL 与当前正在使用的 MySQL 相同，无需迁移")
	}

	slog.Info("db switch start requested",
		"from", source.Mode,
		"to", DBModeMySQL,
		"host", strings.TrimSpace(mysqlConfig.Host),
		"port", normalizeMySQLPort(mysqlConfig.Port),
		"user_name", strings.TrimSpace(mysqlConfig.UserName),
		"db_name", strings.TrimSpace(mysqlConfig.DBName),
	)
	if err = validateMySQLConnectionConfig(mysqlConfig); err != nil {
		return err
	}
	if err = checkMySQLConnectivity(mysqlConfig); err != nil {
		return err
	}

	schemaFilePath := facade.GetConfig().GetString("database.mysql.schema_file")
	content, err := os.ReadFile(schemaFilePath)
	if err != nil {
		return err
	}
	mysqlConfig.SchemaSQL = string(content)

	state.SwitchStatus = DBSwitchStatusRunning
	state.Error = ""
	state.StartedAt = time.Now().Format(time.RFC3339)
	state.FinishedAt = ""
	state.MySQL = &DBSwitchMySQLConfig{
		Host:     strings.TrimSpace(mysqlConfig.Host),
		Port:     strings.TrimSpace(mysqlConfig.Port),
		UserName: strings.TrimSpace(mysqlConfig.UserName),
		Password: mysqlConfig.Password,
		DBName:   strings.TrimSpace(mysqlConfig.DBName),
		Charset:  normalizeMySQLCharset(mysqlConfig.Charset),
		Prefix:   normalizeMySQLPrefix(mysqlConfig.Prefix),
	}
	if err := SaveDBSwitchStateByConfig(facade.GetConfig(), state); err != nil {
		return err
	}
	setDBSwitchRunning(true)

	slog.Info("db switch state saved",
		"from", source.Mode,
		"status", state.SwitchStatus,
		"host", state.MySQL.Host,
		"port", normalizeMySQLPort(state.MySQL.Port),
		"user_name", state.MySQL.UserName,
		"db_name", state.MySQL.DBName,
	)

	go m.runSwitchToMySQL(source, mysqlConfig)
	return nil
}

func (m *DBSwitchManager) runSwitchToMySQL(source DBMigrationSource, mysqlConfig DBSwitchMySQLConfig) {
	defer setDBSwitchRunning(false)

	config := facade.GetConfig()
	state, _ := LoadDBSwitchStateByConfig(config)
	slog.Info("db switch migration started",
		"from", source.Mode,
		"host", strings.TrimSpace(mysqlConfig.Host),
		"port", normalizeMySQLPort(mysqlConfig.Port),
		"user_name", strings.TrimSpace(mysqlConfig.UserName),
		"db_name", strings.TrimSpace(mysqlConfig.DBName),
	)

	mysqlGormDB, err := migrateToMySQL(config, source, mysqlConfig)
	if err != nil {
		slog.Error("db switch migration failed",
			"from", source.Mode,
			"host", strings.TrimSpace(mysqlConfig.Host),
			"port", normalizeMySQLPort(mysqlConfig.Port),
			"user_name", strings.TrimSpace(mysqlConfig.UserName),
			"db_name", strings.TrimSpace(mysqlConfig.DBName),
			"err", err,
		)
		state.Mode = source.Mode
		state.SwitchStatus = DBSwitchStatusFailed
		state.Error = err.Error()
		state.FinishedAt = time.Now().Format(time.RFC3339)
		state.MySQL = cloneMySQLConfig(source.MySQL)
		_ = SaveDBSwitchStateByConfig(config, state)
		return
	}

	dao.SetDefault(mysqlGormDB)
	slog.Info("db switch dao hot swapped",
		"mode", DBModeMySQL,
		"host", strings.TrimSpace(mysqlConfig.Host),
		"port", normalizeMySQLPort(mysqlConfig.Port),
		"db_name", strings.TrimSpace(mysqlConfig.DBName),
	)
	state.Mode = DBModeMySQL
	state.SwitchStatus = DBSwitchStatusSuccess
	state.Error = ""
	state.FinishedAt = time.Now().Format(time.RFC3339)
	state.MySQL = &DBSwitchMySQLConfig{
		Host:     strings.TrimSpace(mysqlConfig.Host),
		Port:     normalizeMySQLPort(mysqlConfig.Port),
		UserName: strings.TrimSpace(mysqlConfig.UserName),
		Password: mysqlConfig.Password,
		DBName:   strings.TrimSpace(mysqlConfig.DBName),
		Charset:  normalizeMySQLCharset(mysqlConfig.Charset),
		Prefix:   normalizeMySQLPrefix(mysqlConfig.Prefix),
	}
	_ = SaveDBSwitchStateByConfig(config, state)
	slog.Info("db switch completed",
		"mode", state.Mode,
		"status", state.SwitchStatus,
		"finished_at", state.FinishedAt,
	)
}

func ResolveDBSwitchStateFile(config *viper.Viper) string {
	return strings.TrimSpace(config.GetString("setting.state_file"))
}

func defaultDBSwitchState() DBSwitchState {
	return DBSwitchState{
		Mode:         DBModeSQLite,
		SwitchStatus: DBSwitchStatusIdle,
	}
}

func LoadDBSwitchStateByConfig(config *viper.Viper) (DBSwitchState, error) {
	state, err := LoadDBSwitchState(ResolveDBSwitchStateFile(config), config)
	if err != nil {
		return state, err
	}
	if !shouldRecoverInterruptedSwitch(state) {
		return state, nil
	}

	state.SwitchStatus = DBSwitchStatusFailed
	state.Error = "数据库迁移任务因服务重启中断，请重新发起切换"
	if state.FinishedAt == "" {
		state.FinishedAt = time.Now().Format(time.RFC3339)
	}
	_ = SaveDBSwitchStateByConfig(config, state)
	return state, nil
}

func LoadDBSwitchState(path string, config *viper.Viper) (DBSwitchState, error) {
	state := defaultDBSwitchState()
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return state, nil
	}
	decryptedContent, err := decryptDBSwitchStateContent(content, config)
	if err != nil {
		decryptedContent = content
	}
	if err := json.Unmarshal(decryptedContent, &state); err != nil {
		return state, err
	}
	if state.Mode == "" {
		state.Mode = DBModeSQLite
	}
	if state.SwitchStatus == "" {
		state.SwitchStatus = DBSwitchStatusIdle
	}
	return state, nil
}

func CurrentDBMode(config *viper.Viper) string {
	state, err := LoadDBSwitchStateByConfig(config)
	if err != nil {
		return DBModeSQLite
	}
	if state.Mode == DBModeMySQL {
		return DBModeMySQL
	}
	return DBModeSQLite
}

func SaveDBSwitchStateByConfig(config *viper.Viper, state DBSwitchState) error {
	return SaveDBSwitchState(ResolveDBSwitchStateFile(config), state)
}

func SaveDBSwitchState(path string, state DBSwitchState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	encryptedContent, err := encryptDBSwitchStateContent(content)
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, encryptedContent, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func encryptDBSwitchStateContent(content []byte) ([]byte, error) {
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	encrypted, err := function.AesEncrypt(string(content), key)
	if err != nil {
		return nil, err
	}
	return []byte(encrypted), nil
}

func decryptDBSwitchStateContent(content []byte, config *viper.Viper) ([]byte, error) {
	key := function.GetMd5(config.GetString("setting.secret"))
	decrypted, err := function.AesDecrypt(string(bytes.TrimSpace(content)), key)
	if err != nil {
		return nil, err
	}
	return []byte(decrypted), nil
}

func ApplyDBSwitchConfig(config *viper.Viper) error {
	state, err := LoadDBSwitchStateByConfig(config)
	if err != nil {
		return err
	}
	if state.Mode != DBModeMySQL || state.MySQL == nil {
		return nil
	}

	mysqlCfg := map[string]interface{}{
		"driver":    "mysql",
		"host":      state.MySQL.Host,
		"port":      normalizeMySQLPort(state.MySQL.Port),
		"user_name": state.MySQL.UserName,
		"password":  state.MySQL.Password,
		"db_name":   state.MySQL.DBName,
		"charset":   normalizeMySQLCharset(state.MySQL.Charset),
		"prefix":    normalizeMySQLPrefix(state.MySQL.Prefix),
	}
	defaultCfg := config.GetStringMap("database.mysql")
	for _, key := range []string{"show_sql", "logger", "slow_threshold", "max_idle_conn", "max_conn"} {
		if _, exists := mysqlCfg[key]; !exists {
			if value, ok := defaultCfg[key]; ok {
				mysqlCfg[key] = value
			}
		}
	}
	config.Set("database.default", mysqlCfg)
	return nil
}

func shouldRecoverInterruptedSwitch(state DBSwitchState) bool {
	return state.Mode != DBModeMySQL && state.SwitchStatus == DBSwitchStatusRunning && !isDBSwitchRunning()
}

func setDBSwitchRunning(running bool) {
	dbSwitchRuntimeState.mu.Lock()
	defer dbSwitchRuntimeState.mu.Unlock()
	dbSwitchRuntimeState.running = running
}

func isDBSwitchRunning() bool {
	dbSwitchRuntimeState.mu.Lock()
	defer dbSwitchRuntimeState.mu.Unlock()
	return dbSwitchRuntimeState.running
}

func ShouldBlockWrites(config *viper.Viper) bool {
	state, err := LoadDBSwitchStateByConfig(config)
	if err != nil {
		return false
	}
	return state.SwitchStatus == DBSwitchStatusRunning
}

func TestMySQLConnection(mysqlConfig DBSwitchMySQLConfig) error {
	if err := validateMySQLConnectionConfig(mysqlConfig); err != nil {
		return err
	}
	return checkMySQLConnectivity(mysqlConfig)
}

func resolveDBMigrationSource(state DBSwitchState) (DBMigrationSource, error) {
	if state.Mode == DBModeMySQL {
		if state.MySQL == nil {
			return DBMigrationSource{}, errors.New("当前数据库模式为 MySQL，但缺少当前 MySQL 配置")
		}
		return DBMigrationSource{
			Mode:  DBModeMySQL,
			MySQL: cloneMySQLConfig(state.MySQL),
		}, nil
	}
	return DBMigrationSource{Mode: DBModeSQLite}, nil
}

func cloneMySQLConfig(config *DBSwitchMySQLConfig) *DBSwitchMySQLConfig {
	if config == nil {
		return nil
	}
	cloned := *config
	return &cloned
}

func isSameMySQLDatabase(source DBSwitchMySQLConfig, target DBSwitchMySQLConfig) bool {
	return strings.TrimSpace(source.Host) == strings.TrimSpace(target.Host) &&
		normalizeMySQLPort(source.Port) == normalizeMySQLPort(target.Port) &&
		strings.TrimSpace(source.UserName) == strings.TrimSpace(target.UserName) &&
		strings.TrimSpace(source.DBName) == strings.TrimSpace(target.DBName)
}

func migrateToMySQL(config *viper.Viper, source DBMigrationSource, mysqlConfig DBSwitchMySQLConfig) (*gorm.DB, error) {
	if source.Mode == DBModeMySQL {
		return migrateMySQLToMySQL(*source.MySQL, mysqlConfig)
	}
	return migrateSQLiteToMySQL(config, mysqlConfig)
}

func migrateSQLiteToMySQL(config *viper.Viper, mysqlConfig DBSwitchMySQLConfig) (*gorm.DB, error) {
	slog.Info("db migrate open sqlite source",
		"sqlite_db", config.GetString("database.default.db_name"),
	)
	sqliteDB, err := sql.Open("sqlite3", config.GetString("database.default.db_name"))
	if err != nil {
		return nil, fmt.Errorf("sqlite connect failed: %w", err)
	}
	defer sqliteDB.Close()

	if err := ensureMySQLDatabaseExists(mysqlConfig); err != nil {
		return nil, err
	}

	slog.Info("db migrate open mysql target",
		"host", strings.TrimSpace(mysqlConfig.Host),
		"port", normalizeMySQLPort(mysqlConfig.Port),
		"user_name", strings.TrimSpace(mysqlConfig.UserName),
		"db_name", strings.TrimSpace(mysqlConfig.DBName),
	)
	mysqlDB, err := sql.Open("mysql", buildMySQLDSN(mysqlConfig, true))
	if err != nil {
		return nil, fmt.Errorf("mysql connect failed: %w", err)
	}
	defer mysqlDB.Close()

	if err := mysqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("mysql ping failed: %w", err)
	}
	if err := applyMySQLSessionTimezone(mysqlDB); err != nil {
		return nil, err
	}
	slog.Info("db migrate mysql ping success",
		"host", strings.TrimSpace(mysqlConfig.Host),
		"port", normalizeMySQLPort(mysqlConfig.Port),
		"db_name", strings.TrimSpace(mysqlConfig.DBName),
	)

	if err := executeMySQLSchema(mysqlDB, mysqlConfig.SchemaSQL); err != nil {
		return nil, err
	}

	tables, err := listSQLiteTables(sqliteDB)
	if err != nil {
		return nil, err
	}
	if len(tables) == 0 {
		return nil, errors.New("sqlite has no business tables to migrate")
	}
	slog.Info("db migrate sqlite tables loaded", "table_count", len(tables))

	for _, table := range tables {
		if table == "ims_formula_setting" || table == "ims_formula_setting_value" || strings.Contains(table, "202") {
			slog.Info("db migrate table skipped by rule", "table", table)
			continue
		}

		hasRows, err := mysqlTableHasRows(mysqlDB, table)
		if err != nil {
			return nil, err
		}
		if hasRows {
			slog.Info("db migrate table skipped because target not empty", "table", table)
			continue
		}
		slog.Info("db migrate table started", "table", table)
		if err := migrateSQLiteTableToMySQL(sqliteDB, mysqlDB, table); err != nil {
			return nil, fmt.Errorf("migrate table %s failed: %w", table, err)
		}
		slog.Info("db migrate table completed", "table", table)
	}

	return openMySQLGormDB(mysqlConfig)
}

func migrateMySQLToMySQL(sourceConfig DBSwitchMySQLConfig, targetConfig DBSwitchMySQLConfig) (*gorm.DB, error) {
	slog.Info("db migrate open mysql source",
		"host", strings.TrimSpace(sourceConfig.Host),
		"port", normalizeMySQLPort(sourceConfig.Port),
		"user_name", strings.TrimSpace(sourceConfig.UserName),
		"db_name", strings.TrimSpace(sourceConfig.DBName),
	)
	sourceDB, err := sql.Open("mysql", buildMySQLDSN(sourceConfig, false))
	if err != nil {
		return nil, fmt.Errorf("source mysql connect failed: %w", err)
	}
	defer sourceDB.Close()

	if err := sourceDB.Ping(); err != nil {
		return nil, fmt.Errorf("source mysql ping failed: %w", err)
	}
	if err := applyMySQLSessionTimezone(sourceDB); err != nil {
		return nil, err
	}

	if err := ensureMySQLDatabaseExists(targetConfig); err != nil {
		return nil, err
	}

	slog.Info("db migrate open mysql target",
		"host", strings.TrimSpace(targetConfig.Host),
		"port", normalizeMySQLPort(targetConfig.Port),
		"user_name", strings.TrimSpace(targetConfig.UserName),
		"db_name", strings.TrimSpace(targetConfig.DBName),
	)
	targetDB, err := sql.Open("mysql", buildMySQLDSN(targetConfig, true))
	if err != nil {
		return nil, fmt.Errorf("target mysql connect failed: %w", err)
	}
	defer targetDB.Close()

	if err := targetDB.Ping(); err != nil {
		return nil, fmt.Errorf("target mysql ping failed: %w", err)
	}
	if err := applyMySQLSessionTimezone(targetDB); err != nil {
		return nil, err
	}

	if err := executeMySQLSchema(targetDB, targetConfig.SchemaSQL); err != nil {
		return nil, err
	}

	tables, err := listMySQLTables(sourceDB)
	if err != nil {
		return nil, err
	}
	if len(tables) == 0 {
		return nil, errors.New("source mysql has no business tables to migrate")
	}
	slog.Info("db migrate mysql tables loaded", "table_count", len(tables))

	for _, table := range tables {
		if table == "ims_formula_setting" || table == "ims_formula_setting_value" || strings.Contains(table, "202") {
			slog.Info("db migrate table skipped by rule", "table", table)
			continue
		}

		hasRows, err := mysqlTableHasRows(targetDB, table)
		if err != nil {
			return nil, err
		}
		if hasRows {
			slog.Info("db migrate table skipped because target not empty", "table", table)
			continue
		}
		slog.Info("db migrate table started", "table", table)
		if err := migrateMySQLTableToMySQL(sourceDB, targetDB, table); err != nil {
			return nil, fmt.Errorf("migrate table %s failed: %w", table, err)
		}
		slog.Info("db migrate table completed", "table", table)
	}

	return openMySQLGormDB(targetConfig)
}

func buildMySQLDSN(mysqlConfig DBSwitchMySQLConfig, multiStatements bool) string {
	charset := normalizeMySQLCharset(mysqlConfig.Charset)
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=%s",
		strings.TrimSpace(mysqlConfig.UserName),
		mysqlConfig.Password,
		strings.TrimSpace(mysqlConfig.Host),
		normalizeMySQLPort(mysqlConfig.Port),
		strings.TrimSpace(mysqlConfig.DBName),
		charset,
		url.QueryEscape(mysqlLocationName),
	)
	if multiStatements {
		dsn += "&multiStatements=true"
	}
	return dsn
}

func openMySQLGormDB(mysqlConfig DBSwitchMySQLConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(buildMySQLDSN(mysqlConfig, false)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   normalizeMySQLPrefix(mysqlConfig.Prefix),
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	if err := applyMySQLSessionTimezone(sqlDB); err != nil {
		return nil, err
	}
	return db, nil
}

func executeMySQLSchema(mysqlDB *sql.DB, schemaSQL string) error {
	statements, err := ReadSQLStatementsFromContent(schemaSQL)
	if err != nil {
		return fmt.Errorf("parse mysql schema sql failed: %w", err)
	}
	if len(statements) == 0 {
		return errors.New("mysql schema sql is empty")
	}
	slog.Info("db migrate execute mysql schema", "statement_count", len(statements))
	for _, stmt := range statements {
		if _, err := mysqlDB.Exec(stmt); err != nil {
			lowerErr := strings.ToLower(err.Error())
			if strings.Contains(lowerErr, "already exists") {
				continue
			}
			return fmt.Errorf("execute mysql schema statement failed: %w", err)
		}
	}
	return nil
}

func ensureMySQLDatabaseExists(mysqlConfig DBSwitchMySQLConfig) error {
	if strings.TrimSpace(mysqlConfig.DBName) == "" {
		return errors.New("mysql db_name 不能为空")
	}

	slog.Info("db migrate ensure mysql database exists",
		"host", strings.TrimSpace(mysqlConfig.Host),
		"port", normalizeMySQLPort(mysqlConfig.Port),
		"db_name", strings.TrimSpace(mysqlConfig.DBName),
	)
	mysqlDB, err := sql.Open("mysql", buildMySQLServerDSN(mysqlConfig))
	if err != nil {
		return fmt.Errorf("mysql connect failed: %w", err)
	}
	defer mysqlDB.Close()

	if err := mysqlDB.Ping(); err != nil {
		return fmt.Errorf("mysql ping failed: %w", err)
	}
	if err := applyMySQLSessionTimezone(mysqlDB); err != nil {
		return err
	}

	query := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s",
		strings.ReplaceAll(strings.TrimSpace(mysqlConfig.DBName), "`", "``"),
		normalizeMySQLCharset(mysqlConfig.Charset),
	)
	if _, err := mysqlDB.Exec(query); err != nil {
		return fmt.Errorf("create mysql database failed: %w", err)
	}
	slog.Info("db migrate mysql database ready", "db_name", strings.TrimSpace(mysqlConfig.DBName))
	return nil
}

func validateMySQLConnectionConfig(mysqlConfig DBSwitchMySQLConfig) error {
	if strings.TrimSpace(mysqlConfig.Host) == "" {
		return errors.New("mysql host 不能为空")
	}
	if strings.TrimSpace(mysqlConfig.UserName) == "" {
		return errors.New("mysql user_name 不能为空")
	}
	return nil
}

func checkMySQLConnectivity(mysqlConfig DBSwitchMySQLConfig) error {
	mysqlDB, err := sql.Open("mysql", buildMySQLServerDSN(mysqlConfig))
	if err != nil {
		return fmt.Errorf("mysql connect failed: %w", err)
	}
	defer mysqlDB.Close()

	if err := mysqlDB.Ping(); err != nil {
		return fmt.Errorf("mysql ping failed: %w", err)
	}
	if err := applyMySQLSessionTimezone(mysqlDB); err != nil {
		return err
	}
	slog.Info("db switch mysql connectivity ok",
		"host", strings.TrimSpace(mysqlConfig.Host),
		"port", normalizeMySQLPort(mysqlConfig.Port),
		"user_name", strings.TrimSpace(mysqlConfig.UserName),
	)
	return nil
}

func normalizeMySQLCharset(charset string) string {
	charset = strings.TrimSpace(charset)
	if charset == "" {
		return "utf8mb4"
	}
	return charset
}

func normalizeMySQLPort(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return "3306"
	}
	return port
}

func normalizeMySQLPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "ims_"
	}
	return prefix
}

func buildMySQLServerDSN(mysqlConfig DBSwitchMySQLConfig) string {
	charset := normalizeMySQLCharset(mysqlConfig.Charset)
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?charset=%s&parseTime=True&loc=%s",
		strings.TrimSpace(mysqlConfig.UserName),
		mysqlConfig.Password,
		strings.TrimSpace(mysqlConfig.Host),
		normalizeMySQLPort(mysqlConfig.Port),
		charset,
		url.QueryEscape(mysqlLocationName),
	)
}

func applyMySQLSessionTimezone(mysqlDB *sql.DB) error {
	if _, err := mysqlDB.Exec("SET time_zone = ?", mysqlSessionTimezone); err != nil {
		return fmt.Errorf("set mysql session time_zone failed: %w", err)
	}
	return nil
}

func listSQLiteTables(sqliteDB *sql.DB) ([]string, error) {
	rows, err := sqliteDB.Query(`
		SELECT name
		FROM sqlite_master
		WHERE type='table'
		  AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func listMySQLTables(mysqlDB *sql.DB) ([]string, error) {
	rows, err := mysqlDB.Query(`
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func mysqlTableHasRows(mysqlDB *sql.DB, table string) (bool, error) {
	var count int64
	row := mysqlDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table))
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("check mysql table %s failed: %w", table, err)
	}
	return count > 0, nil
}

func migrateSQLiteTableToMySQL(sqliteDB, mysqlDB *sql.DB, table string) error {
	sqliteColumns, err := getSQLiteTableColumns(sqliteDB, table)
	if err != nil {
		return err
	}
	return migrateTableToMySQL(sqliteDB, mysqlDB, table, sqliteColumns)
}

func migrateMySQLTableToMySQL(sourceDB, targetDB *sql.DB, table string) error {
	sourceColumns, err := getMySQLTableColumns(sourceDB, table)
	if err != nil {
		return err
	}
	return migrateTableToMySQL(sourceDB, targetDB, table, sourceColumns)
}

func migrateTableToMySQL(sourceDB, targetDB *sql.DB, table string, sourceColumns []string) error {
	mysqlColumns, err := getMySQLTableColumns(targetDB, table)
	if err != nil {
		return err
	}
	columns := intersectMigrationColumns(sourceColumns, mysqlColumns)
	if len(columns) == 0 {
		slog.Warn("db migrate table skipped because no shared columns", "table", table)
		return nil
	}
	slog.Info("db migrate table columns resolved",
		"table", table,
		"source_column_count", len(sourceColumns),
		"mysql_column_count", len(mysqlColumns),
		"migrate_column_count", len(columns),
	)

	rows, err := sourceDB.Query(buildSourceSelectSQL(table, columns))
	if err != nil {
		return err
	}
	defer rows.Close()

	insertSQL := buildMySQLInsertSQL(table, columns)
	tx, err := targetDB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	rowCount := 0
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			_ = tx.Rollback()
			return err
		}

		insertValues := make([]interface{}, len(values))
		for i, value := range values {
			insertValues[i] = normalizeSQLiteValue(value)
		}

		if _, err := stmt.Exec(insertValues...); err != nil {
			_ = tx.Rollback()
			return err
		}
		rowCount++
		if rowCount%500 == 0 {
			_ = stmt.Close()
			if err := tx.Commit(); err != nil {
				return err
			}
			tx, err = targetDB.Begin()
			if err != nil {
				return err
			}
			stmt, err = tx.Prepare(insertSQL)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	if err := rows.Err(); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	slog.Info("db migrate table rows inserted", "table", table, "row_count", rowCount)
	return nil
}

func normalizeSQLiteValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		return v
	}
}

func getSQLiteTableColumns(sqliteDB *sql.DB, table string) ([]string, error) {
	rows, err := sqliteDB.Query(fmt.Sprintf("SELECT * FROM `%s` LIMIT 0", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rows.Columns()
}

func getMySQLTableColumns(mysqlDB *sql.DB, table string) ([]string, error) {
	rows, err := mysqlDB.Query(
		`SELECT COLUMN_NAME
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`, table,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make([]string, 0)
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	return columns, rows.Err()
}

func intersectMigrationColumns(sqliteColumns, mysqlColumns []string) []string {
	mysqlColumnSet := make(map[string]struct{}, len(mysqlColumns))
	for _, column := range mysqlColumns {
		mysqlColumnSet[column] = struct{}{}
	}

	columns := make([]string, 0, len(sqliteColumns))
	for _, column := range sqliteColumns {
		if _, exists := mysqlColumnSet[column]; !exists {
			continue
		}
		columns = append(columns, column)
	}
	return columns
}

func buildSourceSelectSQL(table string, columns []string) string {
	quotedColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, "`"+column+"`")
	}
	return fmt.Sprintf("SELECT %s FROM `%s`", strings.Join(quotedColumns, ","), table)
}

func buildMySQLInsertSQL(table string, columns []string) string {
	quotedColumns := make([]string, 0, len(columns))
	holders := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, "`"+column+"`")
		holders = append(holders, "?")
	}
	return fmt.Sprintf(
		"INSERT INTO `%s` (%s) VALUES (%s)",
		table,
		strings.Join(quotedColumns, ","),
		strings.Join(holders, ","),
	)
}

func ReadSQLStatementsFromContent(content string) ([]string, error) {
	reader := bufio.NewReader(strings.NewReader(content))
	var (
		builder strings.Builder
		result  []string
	)

	for {
		line, err := reader.ReadString('\n')
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "--") {
			builder.WriteString(line)
		}
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(builder.String())
			if stmt != "" {
				result = append(result, stmt)
			}
			builder.Reset()
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	if tail := strings.TrimSpace(builder.String()); tail != "" {
		result = append(result, tail)
	}
	return result, nil
}
