package logic

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

const (
	FORMULA_HIDE      = 1
	FORMULA_DISPLAY   = 2
	FORMULA_RECOMMEND = 99
)

const (
	PUSH_OFFICIAL_STORE_STATUS_AUDIT_ING     = 1
	PUSH_OFFICIAL_STORE_STATUS_AUDIT_FAIL    = 2
	PUSH_OFFICIAL_STORE_STATUS_AUDIT_SUCCESS = 2
)

const (
	FORMULA_AUDIT_SUCCESS = 3
	FORMULA_AUDIT_FAIL    = 2
	FOEMULA_AUDIT_ING     = 1
)

const (
	FORMULA_PRODUCT_CONSOLE_APP = int32(1)
	FORMULA_PRODUCT_LOCAL_APP   = int32(2)
)
const FORMULA_FREE_UPGRADE = 0

func GetFormulaFounderToken(uid int32) string {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return function.GetMd5(today.String() + "-" + strconv.Itoa(int(uid)))
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

	_, _ = dao.Q.Formula.Where(dao.Formula.Name.Eq(formulaName)).Delete()
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
	IsFreeUpgrade        int32
	ProductType          int32
	GoodsId              int32
	GoodsProductId       int32
	RemoteFormulaInfoUrl string
	AuditStatus          int32
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

func (self *Formula) GetZipRelativeDir() string {
	return filepath.Join("Storage", self.Name)
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
