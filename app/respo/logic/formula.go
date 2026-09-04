package logic

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

const (
	FORMULA_HIDE      = 1
	FORMULA_DISPLAY   = 2
	FORMULA_RECOMMEND = 99
)

const FORMULA_FREE_UPGRADE = 0

var applicationIdentifiePattern = regexp.MustCompile(`^[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*$`)

// IsValidIdentifie reports whether a new artifact or child-application
// identifier uses the supported form. Existing data is still read through
// the legacy underscore-to-dash compatibility paths.
func IsValidIdentifie(value string) bool {
	return applicationIdentifiePattern.MatchString(value)
}

// ValidateIdentifie validates an identifier at a new artifact/child
// application input boundary. It intentionally does not normalize values.
func ValidateIdentifie(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("应用标识不能为空")
	}
	if strings.TrimSpace(value) != value || !IsValidIdentifie(value) {
		return fmt.Errorf("应用标识格式无效: %q，仅支持字母、数字和中划线，且不能以中划线开头或结尾", value)
	}
	return nil
}

func GetFormulaByName(formulaName string) *entity.Formula {
	formulaName = strings.ReplaceAll(formulaName, "_", "-")

	row, _ := dao.Q.Formula.Where(dao.Formula.Name.Eq(formulaName)).First()
	if row == nil {
		return nil
	}
	return row
}

func DeleteFormulaByName(formulaName string) {
	formulaName = strings.ReplaceAll(formulaName, "_", "-")

	formula, _ := dao.Q.Formula.Where(dao.Formula.Name.Eq(formulaName)).First()
	if formula == nil {
		return
	}

	_ = dao.Q.Transaction(func(tx *dao.Query) error {
		return deleteFormulaData(tx, formula.ID)
	})
}

func deleteFormulaData(tx *dao.Query, formulaID int32) error {
	if _, err := tx.TagFormula.Where(tx.TagFormula.FormulaID.Eq(formulaID)).Delete(); err != nil {
		return err
	}
	if _, err := tx.Version.Where(tx.Version.FormulaID.Eq(formulaID)).Delete(); err != nil {
		return err
	}

	_, err := tx.Formula.Where(tx.Formula.ID.Eq(formulaID)).Delete()
	return err
}

type Formula struct {
	ID                   int32
	UserId               int32
	Name                 string
	Title                string
	VersionId            int32
	LatestVersionId      int32
	Version              string
	Icon                 string
	Manifest             *logic2.Manifest
	AllManifest          []*logic2.Manifest
	ZipPath              string
	WebZipPaths          map[string]string
	HelmPaths            map[string]string
	CosPath              string
	IsCosFile            bool
	Tags                 []entity.Tag
	ConsoleUid           int32
	InstallServiceFee    float64
	ServicePackages      *accessor.ServicePackagesOption
	VersionPrices        *accessor.VersionPricesOption
	CrossUpgradeFormulas *accessor.CrossUpgradeFormulasOption
	GoodsId              int32
	GoodsProductId       int32
	Setting              *accessor.FormulaSettingOption
}

func (self *Formula) GetVersionByName(version string) *entity.Version {
	versionRow, _ := dao.Q.Version.Where(
		dao.Version.FormulaID.Eq(self.ID),
		dao.Version.Name.Eq(version),
	).First()
	return versionRow
}

func (self *Formula) GetIconName() string {
	return self.Name + ".icon.jpg"
}

func (self *Formula) GetIconRelativePath() string {
	return filepath.Join("icon", self.GetIconName())
}

func (self *Formula) GetFilesRelativeDir() string {
	return filepath.Join(GetFormulaRelativeDir(self.Name, self.VersionId), "files")
}

func GetFormulaRelativeDir(name string, versionId int32) string {
	if versionId == 0 {
		return filepath.Join("Formula", name)
	}
	return filepath.Join("Formula", name, strconv.Itoa(int(versionId)))
}
