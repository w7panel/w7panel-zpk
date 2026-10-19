package formula

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

// AssetsOCITag 是保存制品附属资源的保留 OCI tag。
// 这些资源归属于制品，不随版本隔离，也不包含版本专属 manifest。
// tag 值沿用 shared，以兼容已有附件数据。
const AssetsOCITag = "shared"

// SaveAssetFileAndSyncOCI 先在本地保存一份制品附属资源，再同步全部制品附属资源到 OCI。
// 如只需本地保存，使用 SaveAssetFile。
func (d *Depot) SaveAssetFileAndSyncOCI(formula *Formula, filename, content string) error {
	if err := d.SaveAssetFile(formula, filename, content); err != nil {
		return err
	}
	if err := d.syncAssetsToOCI(formula); err != nil {
		return fmt.Errorf("保存共享文件到 OCI 失败: %w", err)
	}
	return nil
}

// syncAssetsToOCI 将本地的全部制品附属资源上传到保留的 OCI tag。
func (d *Depot) syncAssetsToOCI(formula *Formula) error {
	files, err := d.GetAssetFiles(formula)
	if err != nil {
		return err
	}
	remoteOci, err := commonlogic.GetDefaultRemoteOci(commonlogic.GetFormulaOciName(formula.Name))
	if err != nil {
		return err
	}
	descriptors, err := commonlogic.PackFileListToOci(files)
	if err != nil {
		return err
	}
	return commonlogic.PushOciToRemote(remoteOci, AssetsOCITag, descriptors, nil)
}

// restoreAssetsFromOCI 从 OCI 恢复制品附属资源到本地目录。
func (d *Depot) restoreAssetsFromOCI(formula *Formula) error {
	remoteOci, err := commonlogic.GetDefaultRemoteOci(commonlogic.GetFormulaOciName(formula.Name))
	if err != nil {
		return err
	}
	manifest, err := commonlogic.GetOciManifest(remoteOci, AssetsOCITag)
	if errors.Is(err, commonlogic.OciManifestNotFoundErr) {
		return nil
	}
	if err != nil {
		return err
	}

	return commonlogic.UnPackOciToLocal(remoteOci, manifest, []string{commonlogic.MediaTypeFilesJson}, func(_ string, _ string, reader io.Reader) error {
		files := map[string]string{}
		if err = json.NewDecoder(reader).Decode(&files); err != nil {
			return err
		}

		targetDir := d.getAssetsDir(formula.Name)
		for name, content := range files {
			targetPath, resolveErr := resolveFormulaFilePath(targetDir, name)
			if resolveErr != nil {
				return resolveErr
			}
			if err = os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
				return err
			}
			if err = os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
				return err
			}
		}

		return nil
	})
}
