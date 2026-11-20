package logic

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	documentsdk "github.com/w7panel/w7panel-zpk/common/service/document"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

// FormulaDocument is the artifact-side integration. All document operations go through its SDK.
type FormulaDocument struct{}

func documentClient() *documentsdk.Client {
	config := facade.GetConfig()
	return documentsdk.New(config.GetString("setting.document.base_url"))
}

func IsFormulaDocumentPath(filename string) bool {
	return filename == "docs/.order" || strings.HasPrefix(filename, "docs/") && strings.HasSuffix(strings.ToLower(filename), ".md")
}

// Files preserves the shared-file response consumed by the existing artifact editor.
// Legacy documents are removed before adding the files readable in the document service.
func (l FormulaDocument) Files(ctx context.Context, token, reference string, assets map[string]string) (map[string]string, error) {
	return documentFiles(ctx, documentClient(), token, reference, assets)
}

func documentFiles(ctx context.Context, client *documentsdk.Client, token, reference string, assets map[string]string) (map[string]string, error) {
	tree, err := client.Tree(ctx, token, reference)
	if err != nil {
		return nil, err
	}
	files := make(map[string]string, len(assets)+len(tree.Files))
	for filename, content := range assets {
		if !IsFormulaDocumentPath(filename) {
			files[filename] = content
		}
	}
	order := make([]string, 0, len(tree.Files))
	for _, file := range tree.Files {
		if file.Path == "docs/.order" || !IsFormulaDocumentPath(file.Path) {
			continue
		}
		content, err := client.Read(ctx, token, reference, file.Path)
		if err != nil {
			return nil, err
		}
		files[file.Path] = content.Content
		order = append(order, file.Path)
	}
	files["docs/.order"] = strings.Join(order, ",")
	return files, nil
}

func (l FormulaDocument) Save(ctx context.Context, token, reference, filename, content string) error {
	return saveDocument(ctx, documentClient(), token, reference, filename, content)
}

func saveDocument(ctx context.Context, client *documentsdk.Client, token, reference, filename, content string) error {
	if !IsFormulaDocumentPath(filename) {
		return errors.New("不支持的制品文档路径")
	}
	tree, err := client.Tree(ctx, token, reference)
	if err != nil {
		return err
	}
	if filename == "docs/.order" {
		return orderDocuments(ctx, client, token, reference, content, tree)
	}
	var revision int32
	for _, file := range tree.Files {
		if file.Path == filename {
			revision = file.Revision
			break
		}
	}
	// The original shared-file protocol uses empty content to request deletion.
	if content == "" {
		if revision == 0 {
			return errors.New("文档不存在或无读取权限")
		}
		return client.Delete(ctx, token, reference, filename, revision)
	}
	title := strings.TrimSuffix(strings.TrimPrefix(filename, "docs/"), path.Ext(filename))
	return client.Save(ctx, token, reference, filename, title, content, revision)
}

func orderDocuments(ctx context.Context, client *documentsdk.Client, token, reference, content string, tree documentsdk.FileTree) error {
	existing := make(map[string]bool, len(tree.Files))
	order := make([]string, 0, len(tree.Files))
	for _, file := range tree.Files {
		existing[file.Path] = true
		if !IsFormulaDocumentPath(file.Path) {
			order = append(order, file.Path)
		}
	}
	seen := make(map[string]bool, len(tree.Files))
	if content != "" {
		for _, filename := range strings.Split(content, ",") {
			filename = strings.TrimSpace(filename)
			if filename == "docs/.order" || !IsFormulaDocumentPath(filename) || !existing[filename] || seen[filename] {
				return errors.New("文档排序包含无效、重复或不可读的路径")
			}
			order = append(order, filename)
			seen[filename] = true
		}
	}
	for _, file := range tree.Files {
		if IsFormulaDocumentPath(file.Path) && !seen[file.Path] {
			order = append(order, file.Path)
		}
	}
	return client.Order(ctx, token, reference, order, tree.Revision)
}

func (l FormulaDocument) Ensure(ctx context.Context, token, identifie string, user *entity.RegistryUser) (string, error) {
	if user == nil {
		return "", errors.New("未登录")
	}
	query := dao.Q.Formula
	formula, err := query.WithContext(ctx).Where(query.Name.Eq(strings.ReplaceAll(identifie, "_", "-"))).First()
	if err != nil {
		return "", err
	}
	if formula.UserID != user.ID && !(userlogic.User{}).IsAdminUser(user) {
		return "", errors.New("无制品操作权限")
	}
	if formula.Setting != nil && formula.Setting.DocumentRef != "" {
		return formula.Setting.DocumentRef, nil
	}
	client := documentClient()
	title := formula.Title
	if title == "" {
		title = formula.Name
	}
	reference, err := client.Initialize(ctx, token, formula.Name, title, formula.UserID)
	if err != nil {
		return "", err
	}
	// Read the latest settings before adding the document reference.
	formula, err = query.WithContext(ctx).Where(query.ID.Eq(formula.ID)).First()
	if err != nil {
		return "", err
	}
	setting := accessor.FormulaSettingOption{}
	if formula.Setting != nil {
		setting = *formula.Setting
	}
	setting.DocumentRef = reference
	if _, err = query.WithContext(ctx).Where(query.ID.Eq(formula.ID)).Select(query.Setting).Updates(entity.Formula{Setting: &setting}); err != nil {
		return "", err
	}
	return reference, nil
}
