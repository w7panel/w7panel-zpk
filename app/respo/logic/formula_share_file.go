package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

// FormulaSharedOciTag stores formula-level files which are shared by all
// versions. It is reserved and must never be treated as a formula version.
const FormulaSharedOciTag = "shared"

// PersistSharedFile keeps the original local-file editing flow and persists
// the complete shared working directory to OCI after the mutation succeeds.
func (d *Depot) PersistSharedFile(formula *Formula, filename, content string) error {
	if err := d.SaveSharedFile(formula, filename, content); err != nil {
		return err
	}
	if err := d.packSharedFilesToOci(formula); err != nil {
		return fmt.Errorf("保存共享文件到 OCI 失败: %w", err)
	}
	return nil
}

func (d *Depot) packSharedFilesToOci(formula *Formula) error {
	files, err := d.GetSharedFileList(formula)
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
	return commonlogic.PushOciToRemote(remoteOci, FormulaSharedOciTag, descriptors, nil)
}

// unPackSharedFilesFromOci restores the local shared working directory.
func (d *Depot) unPackSharedFilesFromOci(formula *Formula) error {
	remoteOci, err := commonlogic.GetDefaultRemoteOci(commonlogic.GetFormulaOciName(formula.Name))
	if err != nil {
		return err
	}
	manifest, err := commonlogic.GetOciManifest(remoteOci, FormulaSharedOciTag)
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

		targetDir := d.getSharedFilesDir(formula.Name)
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
