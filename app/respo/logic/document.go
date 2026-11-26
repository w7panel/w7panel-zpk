package logic

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	documentsdk "github.com/w7panel/w7panel-zpk/common/service/w7/document"
)

// FormulaDocument is the artifact-side integration. All document operations go through its SDK.
type FormulaDocument struct{}

func (l FormulaDocument) Initialize(ctx context.Context, token string, formula *entity.Formula) (string, error) {
	if err := w7.DocumentSdk.CreateCategory(ctx, token, documentsdk.CreateCategory{Identifie: "app", Name: "制品文档"}); err != nil {
		var responseError *documentsdk.ResponseError
		if !errors.As(err, &responseError) || responseError.StatusCode != http.StatusConflict {
			return "", err
		}
	}
	title := formula.Title
	if title == "" {
		title = formula.Name
	}
	reference := "app/" + formula.Name
	homepage := "# " + title + "\n\n应用标识：" + formula.Name + "\n"
	file := documentsdk.FileContent{File: documentsdk.File{Path: "index.md", Title: "首页", Protected: 1}, Content: homepage}
	owner, err := (userlogic.User{}).GetById(int(formula.UserID))
	if err != nil {
		return "", err
	}
	err = w7.DocumentSdk.Create(ctx, token, documentsdk.CreateLibrary{
		Ref: reference, Name: title, OwnerID: formula.UserID, OwnerName: owner.Username, Protected: 1,
		Files: []documentsdk.FileContent{file},
	})
	if err == nil {
		return reference, nil
	}
	var responseError *documentsdk.ResponseError
	if !errors.As(err, &responseError) || responseError.StatusCode != http.StatusConflict {
		return "", err
	}
	if err := w7.DocumentSdk.SaveFile(ctx, token, reference, file); err != nil {
		return "", err
	}
	return reference, nil
}

func isDocumentMarkdownPath(filename string) bool {
	return filename != "index.md" && strings.HasSuffix(strings.ToLower(filename), ".md")
}

// Files returns library-relative paths without mixing in artifact assets.
func (l FormulaDocument) Files(ctx context.Context, token, reference string) (map[string]string, error) {
	tree, err := w7.DocumentSdk.Tree(ctx, token, reference)
	if err != nil {
		return nil, err
	}
	files := make(map[string]string, len(tree.Files)+1)
	order := make([]string, 0, len(tree.Files))
	for _, file := range tree.Files {
		if !isDocumentMarkdownPath(file.Path) {
			continue
		}
		content, err := w7.DocumentSdk.Read(ctx, token, reference, file.Path)
		if err != nil {
			return nil, err
		}
		files[file.Path] = content.Content
		order = append(order, file.Path)
	}
	files[".order"] = strings.Join(order, ",")
	return files, nil
}

func (l FormulaDocument) Save(ctx context.Context, token, reference, filename, content string) error {
	if filename != ".order" && !isDocumentMarkdownPath(filename) {
		return errors.New("不支持的制品文档路径")
	}
	if filename == ".order" {
		tree, err := w7.DocumentSdk.Tree(ctx, token, reference)
		if err != nil {
			return err
		}
		return l.orderDocuments(ctx, token, reference, content, tree)
	}
	// The artifact editor uses empty content to request deletion.
	if content == "" {
		return w7.DocumentSdk.Delete(ctx, token, reference, filename)
	}
	title := strings.TrimSuffix(filename, path.Ext(filename))
	return w7.DocumentSdk.Save(ctx, token, reference, filename, title, content)
}

func (l FormulaDocument) orderDocuments(ctx context.Context, token, reference, content string, tree documentsdk.FileTree) error {
	existing := make(map[string]bool, len(tree.Files))
	order := make([]string, 0, len(tree.Files))
	for _, file := range tree.Files {
		if isDocumentMarkdownPath(file.Path) {
			existing[file.Path] = true
		} else {
			order = append(order, file.Path)
		}
	}
	seen := make(map[string]bool, len(tree.Files))
	if content != "" {
		for _, filename := range strings.Split(content, ",") {
			filename = strings.TrimSpace(filename)
			if !existing[filename] || seen[filename] {
				return errors.New("文档排序包含无效、重复或不可读的路径")
			}
			order = append(order, filename)
			seen[filename] = true
		}
	}
	for _, file := range tree.Files {
		if isDocumentMarkdownPath(file.Path) && !seen[file.Path] {
			order = append(order, file.Path)
		}
	}
	return w7.DocumentSdk.Order(ctx, token, reference, order)
}

func (l FormulaDocument) Ensure(ctx context.Context, token, identifie string, user *entity.RegistryUser) (string, error) {
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
	reference, err := l.Initialize(ctx, token, formula)
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
