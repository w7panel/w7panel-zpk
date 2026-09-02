package logic

import (
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
)

type Tag struct {
}

func (l Tag) ResetTags() error {
	defaultTags := []string{"云原生运维", "开发工具", "运行环境", "数据库/中间件", "业务应用", "游戏", "AI", "网关插件", "系统镜像"}
	count, err := dao.Q.Tag.Where(dao.Q.Tag.Name.In(defaultTags...)).Count()
	if err != nil {
		return err
	}
	if int(count) == len(defaultTags) {
		return nil
	}

	err = dao.Q.Transaction(func(tx *dao.Query) error {
		_, err = tx.Tag.Where(tx.Tag.ID.Gt(0)).Delete()
		if err != nil {
			return err
		}

		_, err = tx.TagFormula.Where(tx.TagFormula.ID.Gt(0)).Delete()
		if err != nil {
			return err
		}

		tagBatch := make([]*entity.Tag, 0)
		for _, item := range defaultTags {
			tagBatch = append(tagBatch, &entity.Tag{
				Name: item,
			})
		}
		return tx.Tag.CreateInBatches(tagBatch, 20)
	})

	return err
}
