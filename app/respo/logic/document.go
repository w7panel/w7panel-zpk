package logic

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"

	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	"github.com/w7panel/w7panel-zpk/common/accessor"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	userlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	documentsdk "github.com/w7panel/w7panel-zpk/common/service/w7/document"
)

// FormulaDocument is the artifact-side integration. All document operations go through its SDK.
type FormulaDocument struct{}

func (l FormulaDocument) Initialize(ctx context.Context, formula *entity.Formula) (string, error) {
	title := formula.Title
	if title == "" {
		title = formula.Name
	}
	reference := "app/" + formula.Name
	depot, err := formulalogic.NewDepot()
	if err != nil {
		return "", err
	}
	item, err := depot.GetFormula(formula.Name, "", nil)
	if err != nil {
		return "", err
	}
	homepage := buildDocumentHomepage(item)
	file := documentsdk.FileContent{File: documentsdk.File{Path: "index.md", Title: "首页", Protected: 1}, Content: homepage}
	owner, err := (userlogic.User{}).GetById(int(formula.UserID))
	if err != nil {
		return "", err
	}
	err = w7.DocumentSdk.Create(ctx, formula.UserID, documentsdk.CreateLibrary{
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
	if err := w7.DocumentSdk.SaveFile(ctx, formula.UserID, reference, file); err != nil {
		return "", err
	}
	return reference, nil
}

func isDocumentMarkdownPath(filename string) bool {
	return filename != "index.md" && strings.HasSuffix(strings.ToLower(filename), ".md")
}

// Files returns library-relative paths without mixing in artifact assets.
func (l FormulaDocument) Files(ctx context.Context, userID int32, reference string) (map[string]string, error) {
	tree, err := w7.DocumentSdk.Tree(ctx, userID, reference)
	if err != nil {
		return nil, err
	}
	files := make(map[string]string, len(tree.Files)+1)
	order := make([]string, 0, len(tree.Files))
	for _, file := range tree.Files {
		if !isDocumentMarkdownPath(file.Path) {
			continue
		}
		content, err := w7.DocumentSdk.Read(ctx, userID, reference, file.Path)
		if err != nil {
			return nil, err
		}
		files[file.Path] = content.Content
		order = append(order, file.Path)
	}
	files[".order"] = strings.Join(order, ",")
	return files, nil
}

func (l FormulaDocument) Save(ctx context.Context, userID int32, reference, filename, content string) error {
	if filename != ".order" && !isDocumentMarkdownPath(filename) {
		return errors.New("不支持的制品文档路径")
	}
	if filename == ".order" {
		tree, err := w7.DocumentSdk.Tree(ctx, userID, reference)
		if err != nil {
			return err
		}
		return l.orderDocuments(ctx, userID, reference, content, tree)
	}
	// The artifact editor uses empty content to request deletion.
	if content == "" {
		return w7.DocumentSdk.Delete(ctx, userID, reference, filename)
	}
	title := strings.TrimSuffix(filename, path.Ext(filename))
	return w7.DocumentSdk.Save(ctx, userID, reference, filename, title, content)
}

func (l FormulaDocument) orderDocuments(ctx context.Context, userID int32, reference, content string, tree documentsdk.FileTree) error {
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
	return w7.DocumentSdk.Order(ctx, userID, reference, order)
}

func (l FormulaDocument) Ensure(ctx context.Context, identifie string, user *entity.RegistryUser) (string, error) {
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
	reference, err := l.Initialize(ctx, formula)
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

// SyncHomepage updates only an already linked library, leaving user documents untouched.
func (FormulaDocument) SyncHomepage(ctx context.Context, item *formulalogic.Formula) error {
	if item.Setting == nil || item.Setting.DocumentRef == "" {
		return nil
	}
	file := documentsdk.FileContent{
		File: documentsdk.File{Path: "index.md", Title: "首页", Protected: 1}, Content: buildDocumentHomepage(item),
	}
	return w7.DocumentSdk.SaveFile(ctx, item.UserId, item.Setting.DocumentRef, file)
}
