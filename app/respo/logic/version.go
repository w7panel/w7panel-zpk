package logic

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/hashicorp/go-version"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"
)

type Version struct {
}

func (l Version) FindNextUpgrade(formula *Formula, currentVerStr string, maxVersion string) (*entity.Version, error) {
	curVersion, _ := dao.Q.Version.Where(dao.Q.Version.FormulaID.Eq(formula.ID)).Where(dao.Q.Version.Name.Eq(currentVerStr)).First()
	if curVersion == nil {
		return nil, errors.New("当前版本不存在")
	}

	var allVersions []*entity.Version
	if maxVersion == "" {
		allVersions, _ = dao.Q.Version.
			Where(dao.Q.Version.FormulaID.Eq(formula.ID)).
			Where(dao.Q.Version.ID.Lte(formula.LatestVersionId)).
			Where(dao.Q.Version.PublishStatus.In(FormulaPublishStatusSuccess, 0)).
			Find()
	} else {
		maxVersionModel, _ := dao.Q.Version.Where(dao.Q.Version.FormulaID.Eq(formula.ID)).Where(dao.Q.Version.Name.Eq(maxVersion)).First()
		if maxVersionModel == nil {
			return nil, errors.New("指定的最大可升级版本不存在")
		}
		allVersions, _ = dao.Q.Version.
			Where(dao.Q.Version.FormulaID.Eq(formula.ID)).
			Where(dao.Q.Version.ID.Lte(maxVersionModel.ID)).
			Where(dao.Q.Version.PublishStatus.In(FormulaPublishStatusSuccess, 0)).
			Find()
	}
	if allVersions == nil || len(allVersions) == 0 {
		return nil, nil
	}

	versionMap := map[string]*entity.Version{}
	versions := make([]*version.Version, 0, len(allVersions))
	for _, cversion := range allVersions {
		v, err := version.NewVersion(cversion.Name)
		if err != nil {
			continue
		}

		versions = append(versions, v)
		versionMap[cversion.Name] = cversion
	}
	versionCollection := version.Collection(versions)
	sort.Sort(versionCollection)

	sortVersionNames := make([]string, 0, len(versionCollection))
	for i := range versionCollection {
		sortVersionNames = append(sortVersionNames, versionCollection[i].Original())
	}

	curVersionIndex := slices.Index(sortVersionNames, currentVerStr)
	if curVersionIndex == -1 {
		return nil, nil
	}

	// 2. 解析当前版本
	currentVersion, err := version.NewVersion(currentVerStr)
	if err != nil {
		return nil, fmt.Errorf("当前版本格式无效: %v", err)
	}
	currentSegments := currentVersion.Segments()
	if len(currentSegments) < 2 {
		return nil, nil
	}

	versionCollectionLen := len(versionCollection)
	nextVersionIndex := curVersionIndex + 1
	existsSameMinor := false
	for {
		if nextVersionIndex >= versionCollectionLen {
			nextVersionIndex = nextVersionIndex - 1
			break
		}

		versionSegments := versionCollection[nextVersionIndex].Segments()
		if len(versionSegments) < 2 {
			nextVersionIndex = nextVersionIndex + 1
			continue
		}
		if currentSegments[0] == versionSegments[0] && currentSegments[1] == versionSegments[1] {
			existsSameMinor = true
			nextVersionIndex = nextVersionIndex + 1
			continue
		}

		if currentSegments[0] != versionSegments[0] || currentSegments[1] != versionSegments[1] {
			if existsSameMinor {
				nextVersionIndex = nextVersionIndex - 1
			}
			break
		}
	}
	if nextVersionIndex == curVersionIndex {
		return nil, nil
	}

	return versionMap[sortVersionNames[nextVersionIndex]], nil
}

func (c Version) PublishFormula(consoleUid int32, formula *Formula) error {
	// 如果没有zip包，只有镜像时，不需要打包发布
	// 将文件打包到 Storage 目录，需要同步的再进行同步
	depot, _ := NewDepot()
	err := depot.Pack(formula, false)
	if err != nil {
		return err
	}

	if formula.Setting != nil && formula.Setting.SupportAutoPublishToZpkMarket {
		if consoleUid <= 0 {
			return errors.New("请先在面板绑定微擎云端账号")
		}
		if formula.ConsoleUid > 0 && formula.ConsoleUid != consoleUid {
			return errors.New("当前面板绑定的微擎云端账号与该制品所属账号不一致，请切换账号后重试")
		}
		err = FormulaGoods{}.PublishGoods(formula, devcenter.PublishGoodsReq{
			ConsoleUid: int(consoleUid),
		})
		if err != nil {
			return err
		}
	}

	return AddFormulaPublishTask(formula.Name, formula.Version, formula.VersionId)
}
