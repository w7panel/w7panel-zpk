package logic

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/function"
)

const formulaFilesMigratedMarker = ".files-migrated"

var formulaManifestPathPattern = regexp.MustCompile(`(^|/)manifest\.yaml$`)

func IsFormulaManifestPath(path string) bool {
	return formulaManifestPathPattern.MatchString(normalizeFormulaFilePath(path))
}

// MigrateFormulaFiles moves non-manifest files out of version directories.
// It is idempotent and runs before business loops start.
func (d *Depot) MigrateFormulaFiles() error {
	formulas, err := dao.Q.Formula.Find()
	if err != nil {
		return err
	}
	for _, formulaRow := range formulas {
		if d.formulaFilesMigrated(formulaRow.Name) {
			continue
		}
		formula, formulaErr := d.GetFormula(formulaRow.Name, "", nil)
		if formulaErr != nil {
			return fmt.Errorf("迁移制品 %s 文件失败: %w", formulaRow.Name, formulaErr)
		}
		if migrateErr := d.migrateFormulaFiles(formula); migrateErr != nil {
			return fmt.Errorf("迁移制品 %s 文件失败: %w", formulaRow.Name, migrateErr)
		}
	}
	return nil
}

func (d *Depot) migrateFormulaFiles(formula *Formula) error {
	if d.formulaFilesMigrated(formula.Name) {
		return nil
	}

	latestFormula := *formula
	if formula.LatestVersionId > 0 {
		latestFormula.VersionId = formula.LatestVersionId
	}
	latestFilesDir := filepath.Join(d.basePath, latestFormula.GetFilesRelativeDir())
	latestFiles, err := d.getFileList(latestFilesDir)
	if err != nil {
		return err
	}
	if err = d.mergeSharedFiles(formula, latestFiles); err != nil {
		return err
	}
	if !containsSharedFormulaFiles(latestFiles) {
		if err = d.seedSharedFilesFromOci(&latestFormula); err != nil {
			return err
		}
	}

	versions, err := dao.Q.Version.Where(dao.Q.Version.FormulaID.Eq(formula.ID)).Find()
	if err != nil {
		return err
	}
	for _, version := range versions {
		versionFormula := &Formula{Name: formula.Name, VersionId: version.ID}
		versionFilesDir := filepath.Join(d.basePath, versionFormula.GetFilesRelativeDir())
		versionFiles, listErr := d.getFileList(versionFilesDir)
		if listErr != nil {
			return listErr
		}
		for path := range versionFiles {
			if IsFormulaManifestPath(path) {
				continue
			}
			if err = d.saveFile(versionFilesDir, path, ""); err != nil {
				return err
			}
		}
	}
	return d.markFormulaFilesMigrated(formula.Name)
}

func (d *Depot) mergeSharedFiles(formula *Formula, files map[string]string) error {
	sharedFilesDir := d.getSharedFilesDir(formula.Name)
	for path, content := range files {
		if IsFormulaManifestPath(path) {
			continue
		}
		filePath, err := resolveFormulaFilePath(sharedFilesDir, normalizeFormulaFilePath(path))
		if err != nil {
			return err
		}
		if function.FileExists(filePath) {
			continue
		}
		if err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}
		if err = os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func containsSharedFormulaFiles(files map[string]string) bool {
	for path := range files {
		if !IsFormulaManifestPath(path) {
			return true
		}
	}
	return false
}

func (d *Depot) formulaFilesMigrated(formulaName string) bool {
	_, err := os.Stat(d.formulaFilesMigratedMarkerPath(formulaName))
	return err == nil
}

func (d *Depot) markFormulaFilesMigrated(formulaName string) error {
	markerPath := d.formulaFilesMigratedMarkerPath(formulaName)
	if err := os.MkdirAll(filepath.Dir(markerPath), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(markerPath, []byte("1"), os.ModePerm)
}

func (d *Depot) formulaFilesMigratedMarkerPath(formulaName string) string {
	return filepath.Join(d.basePath, GetFormulaRelativeDir(formulaName, 0), formulaFilesMigratedMarker)
}

func normalizeFormulaFilePath(path string) string {
	return strings.ReplaceAll(strings.TrimPrefix(path, "/"), "\\", "/")
}
