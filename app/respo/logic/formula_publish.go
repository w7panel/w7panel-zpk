package logic

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	"github.com/w7panel/w7panel-zpk/app/respo/logic/helm"
	goodslogic "github.com/w7panel/w7panel-zpk/app/respo/logic/zpkmarket/goods"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gorm"
)

type FormulaPublish struct{}

func (p FormulaPublish) OnRepositoryPushed(payload registry.RegistryRepositoryWebHookPayLoad) {
	slog.Info("OnRepositoryPushed", "payload", payload)
	repositoryName, namespace := commonlogic.ParseRepositoryNameAndNamespace(payload.Event.Target.Repository)
	tag := payload.Event.Target.Tag
	if namespace != facade.GetConfig().GetString("setting.depot.oci_namespace") || tag == "" || tag == formulalogic.AssetsOCITag || tag == "latest" {
		return
	}
	var err error
	defer func() {
		if err != nil {
			slog.Error("OnRepositoryPushed", "repository", payload.Event.Target.Repository, "tag", tag, "err", err)
		}
	}()
	depot, _ := formulalogic.NewDepot()
	formula := formulalogic.GetFormulaByName(repositoryName)
	if formula == nil {
		if payload.Event.Actor.Name == "" {
			err = errors.New("推送用户不能为空")
			return
		}
		user, userErr := (commonlogic.User{}).GetByUsername(payload.Event.Actor.Name)
		if userErr != nil {
			err = userErr
			return
		}
		if err = depot.AddFormula(repositoryName, tag, user); err != nil {
			return
		}
		formula = formulalogic.GetFormulaByName(repositoryName)
	}
	if formula == nil {
		err = errors.New("应用不存在")
		return
	}
	version, err := dao.Q.Version.Where(dao.Q.Version.FormulaID.Eq(formula.ID), dao.Q.Version.Name.Eq(tag)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		version = &entity.Version{FormulaID: formula.ID, Name: tag, Description: "新版本", PublishStatus: -1}
		err = dao.Q.Version.Create(version)
	}
	if err != nil {
		return
	}
	filesDir := filepath.Join(depot.GetBasePath(), formulalogic.GetFormulaRelativeDir(formula.Name, version.ID), "files")
	if err = os.RemoveAll(filesDir); err != nil {
		return
	}

	// CLI 已完成 OCI 打包，只在显式请求时触发后续发布。
	remoteOci, err := commonlogic.GetDefaultRemoteOci(payload.Event.Target.Repository)
	if err != nil {
		return
	}
	descriptor, err := remoteOci.Resolve(context.Background(), tag)
	if err != nil {
		return
	}
	if payload.Event.Target.Digest != "" && payload.Event.Target.Digest != descriptor.Digest.String() {
		return
	}
	manifest, err := commonlogic.GetOciManifest(context.Background(), remoteOci, descriptor.Digest.String())
	if err != nil || manifest.Annotations["w7.cc/auto_publish"] != "true" {
		return
	}
	user, err := (commonlogic.User{}).GetByUsername(payload.Event.Actor.Name)
	if err != nil {
		return
	}
	if formula.UserID != user.ID && !(commonlogic.User{}.IsAdminUser(user)) {
		err = errors.New("仅应用拥有者或管理员可以发布版本")
		return
	}
	app, err := depot.GetFormula(formula.Name, tag, user)
	if err != nil {
		return
	}
	err = p.PublishFormula(app.ConsoleUid, app, false)
}

func (p FormulaPublish) PublishFormula(consoleUid int32, formula *formulalogic.Formula, packhelm bool) (err error) {
	defer func() {
		if err != nil {
			_, stateErr := dao.Q.Version.Where(dao.Q.Version.ID.Eq(formula.VersionId)).
				Select(dao.Q.Version.PublishStatus, dao.Q.Version.PublishFailReason).Updates(entity.Version{
				PublishStatus: formulalogic.FormulaPublishStatusFail, PublishFailReason: err.Error(),
			})
			slog.Error("publish application version", "name", formula.Name, "version", formula.Version, "err", err, "state_err", stateErr)
		}
	}()
	if helm.IsFormulaHelmPackPending(formula.Name, formula.Version) {
		return nil
	}
	if packhelm {
		depot, depotErr := formulalogic.NewDepot()
		if depotErr != nil {
			return depotErr
		}
		if err = depot.Pack(formula, false); err != nil {
			return err
		}
	}
	if formula.Setting != nil && formula.Setting.SupportAutoPublishToZpkMarket {
		if consoleUid <= 0 {
			return errors.New("请先在面板绑定微擎云端账号")
		}
		if formula.ConsoleUid > 0 && formula.ConsoleUid != consoleUid {
			return errors.New("当前面板绑定的微擎云端账号与该制品所属账号不一致，请切换账号后重试")
		}
		err := goodslogic.PublishGoods(formula, devcenter.PublishGoodsReq{
			ConsoleUid: int(consoleUid),
		})
		if err != nil {
			return err
		}
	}

	return helm.AddFormulaHelmPackTask(formula.Name, formula.Version, formula.VersionId)
}
