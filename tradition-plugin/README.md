# 传统应用插件安装工具

传统应用插件安装及文件优先级管理工具。项目结构与 `../cli` 一致，使用 Rangine、Cobra 和独立 Go module，可单独构建。

OCI Layout 只有一个逻辑制品和一个 `current` tag：config 只保存每个插件的文件路径列表，manifest 的 layers 固定按“原应用文件 → 低优先级插件 → 高优先级插件”排列。Layer 名称、Blob descriptor 和顺序都直接从 manifest 获取，不在 config 中重复保存。

优先级只在生成 manifest 时用于排列插件层。安装、卸载和恢复不再计算优先级，只从下到上调用 containerd 应用 `manifest.layers`；tar 解包和摘要校验由 containerd/ORAS 处理。

## 构建

```bash
make build
```

## 命令

参数说明：

| 参数 | 使用命令 | 作用 |
| --- | --- | --- |
| `--data-dir` | 全部命令 | 保存 OCI Layout 和插件安装状态的持久化目录；同一个传统应用应始终使用同一目录。 |
| `--site-dir` | 全部命令 | 传统应用实际运行的站点根目录，插件文件最终会应用到这里。 |
| `--package-dir` | `app update` | 已解压的新传统应用包目录，用于更新插件涉及的原应用文件。 |
| `--package-dir` | `plugin install` | 已解压的插件包目录，该目录内容会打成当前插件的 tar layer。 |
| `--policy-file` | `app update`、`plugin install`、`plugin uninstall` | 传统应用配置 JSON 文件，工具从 `platform.tradition.plugins` 读取受管插件及优先级。 |
| `--plugin` | `plugin install`、`plugin uninstall` | 插件唯一标识，必须与配置中的 `identifie` 完全一致，不会自动转换格式。 |

```bash
# 原有流程完成传统应用安装或升级后，刷新受管原文件并重新应用插件
w7-tradition-plugin app update \
  --data-dir "$data_dir" \
  --site-dir /www/wwwroot \
  --package-dir "$app_package_dir" \
  --policy-file "$app_json"

# 安装插件
w7-tradition-plugin plugin install \
  --data-dir "$data_dir" \
  --site-dir /www/wwwroot \
  --package-dir "$plugin_package_dir" \
  --policy-file "$app_json" \
  --plugin "$plugin_id"

# 卸载插件
w7-tradition-plugin plugin uninstall \
  --data-dir "$data_dir" \
  --site-dir /www/wwwroot \
  --policy-file "$app_json" \
  --plugin "$plugin_id"

# 安装或卸载中断后，重新生成已管理的文件
w7-tradition-plugin restore --data-dir "$data_dir" --site-dir /www/wwwroot
```

`app update` 和 `plugin install/uninstall` 会自动读取 `--policy-file` 中的最新插件优先级，不需要额外执行配置更新或检查命令。未配置优先级的插件会返回 `managed: false`，并由原有安装或卸载流程继续处理。

本工具不会将完整传统应用打包到 OCI。首次安装受管插件时，只从站点中保存该插件覆盖到的原文件；应用升级时，只从 `--package-dir` 刷新这些已跟踪路径。插件层也只包含插件包中实际存在的文件。

每个插件的文件路径列表保存在 config，当前安装的插件、Blob 和层顺序由 manifest layers 表示；这些信息都在唯一的 `current` OCI 制品中，不会另外生成状态 JSON 文件。每次更新后会清理旧制品不再引用的内容。

工具不会备份 OCI 文件。OCI Layout 被人为删除或损坏后，需要重新安装传统应用及相应插件。
