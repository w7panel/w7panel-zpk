package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	systemlogic "github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
	"gorm.io/gorm"
	"sigs.k8s.io/yaml"
)

const environmentApplicationCategory = "环境应用"

type environmentCategoryDepot interface {
	GetFormula(string, string, *entity.RegistryUser) (*formula.Formula, error)
	GetManifestFileList(*formula.Formula) (map[string]string, error)
	SaveManifestFile(*formula.Formula, string, string) error
}

type SetEnvironmentCategory struct {
	console.Abstract
}

func (c SetEnvironmentCategory) GetName() string {
	return "respo:set-environment-category"
}

func (c SetEnvironmentCategory) GetDescription() string {
	return "将指定标识的制品分类修改为环境应用，并将最新版本设为原生应用"
}

func (c SetEnvironmentCategory) Configure(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
	cmd.Flags().String("identifie", "", "要修改分类和应用类型的制品标识")
	_ = cmd.MarkFlagRequired("identifie")
}

func (c SetEnvironmentCategory) Handle(cmd *cobra.Command, args []string) {
	identifie, err := cmd.Flags().GetString("identifie")
	if err != nil {
		panic(err)
	}
	if systemlogic.ShouldBlockWrites(facade.GetConfig()) {
		panic("数据库迁移进行中，当前已禁止写操作")
	}
	depot, err := formula.NewDepot()
	if err != nil {
		panic(err)
	}
	if err := setEnvironmentCategory(identifie, depot); err != nil {
		panic(err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "制品 %s 的分类已修改为%s，最新版本的应用类型已修改为原生应用（docker）\n", strings.ReplaceAll(strings.TrimSpace(identifie), "_", "-"), environmentApplicationCategory)
}

func setEnvironmentCategory(identifie string, depot environmentCategoryDepot) error {
	identifie = strings.ReplaceAll(strings.TrimSpace(identifie), "_", "-")
	if err := formula.ValidateIdentifie(identifie); err != nil {
		return err
	}
	row, err := dao.Q.Formula.Where(dao.Q.Formula.Name.Eq(identifie)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("制品 %q 不存在", identifie)
	}
	if err != nil {
		return fmt.Errorf("查询制品 %q 失败: %w", identifie, err)
	}
	if row.VersionLatestID <= 0 {
		return fmt.Errorf("制品 %q 没有最新版本", identifie)
	}
	current, err := depot.GetFormula(identifie, "", nil)
	if err != nil {
		return fmt.Errorf("读取制品 %q 最新版本失败: %w", identifie, err)
	}
	files, err := depot.GetManifestFileList(current)
	if err != nil {
		return fmt.Errorf("读取制品 %q manifest 失败: %w", identifie, err)
	}
	original := files["manifest.yaml"]
	var manifest map[string]interface{}
	if err := yaml.Unmarshal([]byte(original), &manifest, func(decoder *json.Decoder) *json.Decoder {
		decoder.UseNumber()
		return decoder
	}); err != nil {
		return fmt.Errorf("解析制品 %q manifest 失败: %w", identifie, err)
	}
	application, ok := manifest["application"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("制品 %q manifest 缺少有效的 application 配置", identifie)
	}
	changed := application["type"] != commonlogic.Docker_App
	if changed {
		// 只修改类型，保留未在 Manifest 结构体中声明的扩展字段。
		application["type"] = commonlogic.Docker_App
		content, err := yaml.Marshal(manifest)
		if err != nil {
			return fmt.Errorf("生成制品 %q manifest 失败: %w", identifie, err)
		}
		if err := depot.SaveManifestFile(current, "manifest.yaml", string(content)); err != nil {
			return restoreEnvironmentManifest(depot, current, original, fmt.Errorf("保存制品 %q manifest 失败: %w", identifie, err))
		}
	}
	if _, err := (logic.Category{}).SaveFormulaCategory(row.ID, environmentApplicationCategory); err != nil {
		err = fmt.Errorf("修改制品 %q 的分类失败: %w", identifie, err)
		if changed {
			return restoreEnvironmentManifest(depot, current, original, err)
		}
		return err
	}
	return nil
}

func restoreEnvironmentManifest(depot environmentCategoryDepot, current *formula.Formula, original string, cause error) error {
	if err := depot.SaveManifestFile(current, "manifest.yaml", original); err != nil {
		return errors.Join(cause, fmt.Errorf("恢复制品 %q 原 manifest 失败，请检查该版本配置: %w", current.Name, err))
	}
	return cause
}
