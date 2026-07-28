package command

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
	"sigs.k8s.io/yaml"
)

type ImportHigressPlugins struct {
	console.Abstract
}

func (ImportHigressPlugins) GetName() string {
	return "respo:import-higress-plugins"
}

func (ImportHigressPlugins) GetDescription() string {
	return "import Higress built-in plugins as gateway plugin artifacts"
}

func (ImportHigressPlugins) Configure(cmd *cobra.Command) {
	cmd.Flags().Int("user-id", 0, "console UID used when publishing artifacts")
	cmd.Flags().Bool("auto-publish-to-zpk-market", false, "enable automatic publishing to the ZPK market")
}

func (ImportHigressPlugins) Handle(cmd *cobra.Command, _ []string) {
	consoleUID, err := cmd.Flags().GetInt("user-id")
	if err != nil {
		panic(err)
	}
	autoPublishToZpkMarket, err := cmd.Flags().GetBool("auto-publish-to-zpk-market")
	if err != nil {
		panic(err)
	}
	registryUsername := facade.GetConfig().GetString("registry_cli.default.username")
	user, err := dao.Q.RegistryUser.Where(dao.Q.RegistryUser.Username.Eq(registryUsername)).First()
	if err != nil || user == nil {
		panic(fmt.Errorf("default registry user %q not found", registryUsername))
	}

	depot, err := logic.NewDepot()
	if err != nil {
		panic(err)
	}
	operationsTag, _ := dao.Q.Tag.Where(dao.Q.Tag.Name.Eq("云原生运维")).First()
	if operationsTag == nil {
		operationsTag = &entity.Tag{Name: "云原生运维"}
		if err = dao.Q.Tag.Create(operationsTag); err != nil {
			panic(fmt.Errorf("create artifact tag: %w", err))
		}
	}

	created := 0
	overwritten := 0
	for _, plugin := range higressBuiltinPlugins() {
		existing, queryErr := dao.Q.Formula.Where(dao.Q.Formula.Name.Eq(plugin.Identifie)).First()
		isOverwrite := queryErr == nil && existing != nil

		if err = depot.AddFormula(plugin.Identifie, higressBuiltinPluginVersion, user); err != nil {
			panic(fmt.Errorf("create %s: %w", plugin.Identifie, err))
		}
		formulaRow, err := dao.Q.Formula.Where(dao.Q.Formula.Name.Eq(plugin.Identifie)).First()
		if err != nil || formulaRow == nil {
			panic(fmt.Errorf("load created artifact %s: %w", plugin.Identifie, err))
		}
		versionRow, err := dao.Q.Version.
			Where(dao.Q.Version.FormulaID.Eq(formulaRow.ID)).
			Where(dao.Q.Version.Name.Eq(higressBuiltinPluginVersion)).
			First()
		if err != nil || versionRow == nil {
			panic(fmt.Errorf("load created artifact version %s: %w", plugin.Identifie, err))
		}

		manifest := newHigressBuiltinPluginManifest(plugin)
		content, err := yaml.Marshal(manifest)
		if err != nil {
			panic(fmt.Errorf("marshal %s manifest: %w", plugin.Identifie, err))
		}
		formula := &logic.Formula{
			Name:      plugin.Identifie,
			VersionId: versionRow.ID,
		}
		if err = depot.SaveManifestFile(formula, "manifest.yaml", string(content)); err != nil {
			panic(fmt.Errorf("save %s manifest: %w", plugin.Identifie, err))
		}
		if err = saveHigressPluginIcon(formula, plugin.IconURL); err != nil {
			panic(fmt.Errorf("save %s icon: %w", plugin.Identifie, err))
		}
		if err = depot.SaveSharedFile(formula, "readme.md", plugin.Readme); err != nil {
			panic(fmt.Errorf("save %s readme: %w", plugin.Identifie, err))
		}
		if formulaRow.Setting == nil {
			formulaRow.Setting = &accessor.FormulaSettingOption{}
		}
		formulaRow.Setting.SupportAutoPublishToZpkMarket = autoPublishToZpkMarket
		if _, err = dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(formulaRow.ID)).Updates(entity.Formula{
			Title:   plugin.Title,
			Setting: formulaRow.Setting,
		}); err != nil {
			panic(fmt.Errorf("update %s title: %w", plugin.Identifie, err))
		}
		formulaTag, _ := dao.Q.TagFormula.Where(
			dao.Q.TagFormula.TagID.Eq(operationsTag.ID),
			dao.Q.TagFormula.FormulaID.Eq(formulaRow.ID),
		).First()
		if formulaTag == nil {
			if err = dao.Q.TagFormula.Create(&entity.TagFormula{
				TagID:     operationsTag.ID,
				FormulaID: formulaRow.ID,
			}); err != nil {
				panic(fmt.Errorf("tag %s: %w", plugin.Identifie, err))
			}
		}
		publishFormula, err := depot.GetFormula(plugin.Identifie, higressBuiltinPluginVersion, user)
		if err != nil {
			panic(fmt.Errorf("load %s for publish: %w", plugin.Identifie, err))
		}
		if err = (logic.Version{}).PublishFormula(int32(consoleUID), publishFormula); err != nil {
			panic(fmt.Errorf("publish %s: %w", plugin.Identifie, err))
		}

		if isOverwrite {
			fmt.Fprintf(cmd.OutOrStdout(), "overwritten %s (%s)\n", plugin.Identifie, strings.TrimPrefix(plugin.Image, "oci://"))
			overwritten++
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "created %s (%s)\n", plugin.Identifie, strings.TrimPrefix(plugin.Image, "oci://"))
			created++
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Higress plugin import complete: created=%d overwritten=%d excluded_cluster_rate_limit=1\n", created, overwritten)
}

func saveHigressPluginIcon(formula *logic.Formula, iconURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iconURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024+1))
	if err != nil {
		return err
	}
	if len(content) > 1024*1024 {
		return fmt.Errorf("icon exceeds 1 MiB")
	}
	return logic.GetLocalClient().UploadByContent(formula.GetIconRelativePath(), string(content))
}
