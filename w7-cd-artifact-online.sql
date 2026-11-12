-- phpMyAdmin SQL Dump
-- version 5.2.0
-- https://www.phpmyadmin.net/
--
-- 主机： w7-mysql-zhqppgqs.default.svc.cluster.local
-- 生成日期： 2026-04-23 04:03:44
-- 服务器版本： 8.0.29
-- PHP 版本： 8.0.25

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
START TRANSACTION;
SET time_zone = "+08:00";


/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;

--
-- 数据库： `w7-cd-artifact`
--

-- --------------------------------------------------------

--
-- 表的结构 `ims_formula`
--

CREATE TABLE `ims_formula` (
                               `id` int NOT NULL,
                               `user_id` int NOT NULL DEFAULT '0',
                               `remote_uid` int NOT NULL DEFAULT '0',
                               `name` varchar(64) NOT NULL,
                               `title` varchar(255) NOT NULL,
                               `remote_formula_info_url` varchar(255) DEFAULT NULL,
                               `version_latest_id` int NOT NULL DEFAULT '0',
                               `install_total` int NOT NULL DEFAULT '0',
                               `install_service_fee` decimal(10,2) NOT NULL DEFAULT '0.00',
                               `is_free_upgrade` tinyint NOT NULL DEFAULT '0',
                               `status` int NOT NULL DEFAULT '2',
                               `goods_id` int NOT NULL DEFAULT '0',
                               `goods_product_id` int NOT NULL DEFAULT '0',
                               `product_type` tinyint NOT NULL DEFAULT '0',
                               `service_packages` text,
                               `version_prices` text,
                               `cross_upgrade_formulas` text,
                               `setting` text,
                               `audit_status` tinyint DEFAULT '3',
                               `audit_remark` varchar(500) DEFAULT NULL,
                               `publish_official_store_status` tinyint DEFAULT '0',
                               `created_at` int NOT NULL DEFAULT '0',
                               `updated_at` int NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ims_registry_namespace`
--

CREATE TABLE `ims_registry_namespace` (
                                          `id` int NOT NULL,
                                          `user_id` int DEFAULT '0',
                                          `name` varchar(255) NOT NULL,
                                          `visible_type` tinyint NOT NULL DEFAULT '1',
                                          `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                          `updated_at` datetime DEFAULT NULL,
                                          `deleted_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ims_registry_repository`
--

CREATE TABLE `ims_registry_repository` (
                                           `id` int NOT NULL,
                                           `user_id` int DEFAULT '0',
                                           `name` varchar(255) NOT NULL,
                                           `registry` varchar(255) NOT NULL,
                                           `namespace` varchar(128) NOT NULL,
                                           `visible_type` tinyint NOT NULL DEFAULT '1',
                                           `pull_num` int NOT NULL DEFAULT '0',
                                           `desc` varchar(500) NOT NULL,
                                           `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                           `updated_at` datetime DEFAULT NULL,
                                           `deleted_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ims_registry_repository_deploy_rule`
--

CREATE TABLE `ims_registry_repository_deploy_rule` (
                                                       `id` int NOT NULL,
                                                       `repository_id` int NOT NULL DEFAULT '0',
                                                       `deploy_type` int NOT NULL DEFAULT '1',
                                                       `match_type` int NOT NULL DEFAULT '0',
                                                       `tag_name` varchar(255) NOT NULL,
                                                       `k8s_config` text,
                                                       `k8s_namespace` varchar(128) NOT NULL,
                                                       `k8s_controller_type` varchar(128) NOT NULL,
                                                       `k8s_app_name` varchar(255) NOT NULL,
                                                       `k8s_container_name` varchar(255) NOT NULL,
                                                       `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                                       `latest_trigger_at` datetime DEFAULT NULL,
 `workflow_task_key` varchar(40) DEFAULT NULL COMMENT '工作流任务稳定标识，供镜像来源和执行记录引用',
 `execution_order` int DEFAULT 1 COMMENT '执行分组顺序，数字小的先执行，同值并行',
 `task_action` varchar(20) DEFAULT 'deploy' COMMENT '任务动作：deploy 部署应用，build 镜像重构',
 `image_source` varchar(20) DEFAULT 'original' COMMENT '镜像来源：original 本轮触发镜像，task 指定构建任务产物',
 `source_build_task_key` varchar(40) DEFAULT NULL COMMENT '来源构建任务的稳定标识，仅 image_source=task 时使用',
 `dockerfile_instructions` text COMMENT '镜像重构追加的 Dockerfile 指令，不包含系统生成的 FROM',
 `output_tag_suffix` varchar(100) DEFAULT NULL COMMENT '输出 tag 相对本轮触发 tag 的后缀，空值覆盖原 tag',
 `task_name` varchar(120) DEFAULT NULL COMMENT '任务展示名称，改名不改变任务稳定标识'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ims_registry_repository_deploy_rule_match_log`
--

CREATE TABLE `ims_registry_repository_deploy_rule_match_log` (
                                                                 `id` int NOT NULL,
                                                                 `rule_id` int NOT NULL DEFAULT '0',
                                                                 `image_name` varchar(255) NOT NULL,
                                                                 `k8s_log` text,
                                                                 `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `workflow_run_id` varchar(40) DEFAULT NULL COMMENT '本轮流程唯一标识，关联汇总记录及全部任务记录，旧日志为 NULL',
 `workflow_task_key` varchar(40) DEFAULT '' COMMENT '对应规则的工作流任务标识，空值表示整轮流程汇总记录',
 `repository_id` int DEFAULT NULL,
 `trigger_repository` varchar(255) DEFAULT NULL COMMENT '本轮触发镜像仓库名快照，不含 registry 域名',
 `trigger_tag` varchar(128) DEFAULT NULL COMMENT '本轮触发事件的原始镜像 tag',
 `trigger_digest` varchar(80) DEFAULT NULL COMMENT '本轮触发原镜像的 sha256 digest，不是构建输出 digest',
 `execution_status` varchar(20) DEFAULT NULL COMMENT '执行状态：pending 排队，running 执行中，succeeded 成功，failed 失败，skipped 跳过',
 `failure_reason` text COMMENT '执行失败或中断原因，不保存完整任务日志',
 `encrypted_config_snapshot` longtext COMMENT '本轮加密任务配置快照，仅汇总记录保存，执行不受后续编辑影响',
 `updated_at` datetime DEFAULT NULL,
 `build_output_tag` varchar(128) DEFAULT NULL COMMENT '构建任务实际推送的镜像 tag',
 `build_output_digest` varchar(80) DEFAULT NULL COMMENT '构建任务推送产物的 sha256 digest',
 UNIQUE KEY `workflow_run_task` (`workflow_run_id`, `workflow_task_key`),
 KEY `idx_workflow_repository` (`repository_id`),
 KEY `idx_workflow_status` (`execution_status`),
 KEY `idx_workflow_output_tag` (`build_output_tag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ims_registry_repository_tag`
--

CREATE TABLE `ims_registry_repository_tag` (
                                               `id` int NOT NULL,
                                               `repository_id` int NOT NULL DEFAULT '0',
                                               `name` varchar(255) NOT NULL,
                                               `creared_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                               `latest_push_at` datetime DEFAULT NULL,
 `is_rebuilt` tinyint(1) NOT NULL DEFAULT 0 COMMENT '当前 tag 是否为工作流重构产物，外部上传后清除'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ims_registry_user`
--

CREATE TABLE `ims_registry_user` (
                                     `id` int NOT NULL,
                                     `username` varchar(255) NOT NULL,
                                     `password` varchar(255) NOT NULL,
                                     `desc` varchar(255) NOT NULL,
                                     `type` tinyint NOT NULL DEFAULT '0',
                                     `expire_days` int NOT NULL DEFAULT '-1',
                                     `setting` text,
                                     `role` varchar(32) DEFAULT NULL,
                                     `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                     `updated_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ims_registry_user_permission`
--

CREATE TABLE `ims_registry_user_permission` (
                                                `id` int NOT NULL,
                                                `user_id` int NOT NULL DEFAULT '0',
                                                `resource_value` varchar(255) NOT NULL,
                                                `resource_type` varchar(32) NOT NULL,
                                                `action` varchar(32) NOT NULL,
                                                `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ims_tag`
--

CREATE TABLE `ims_tag` (
                           `id` int NOT NULL,
                           `name` varchar(32) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- --------------------------------------------------------

--
-- 表的结构 `ims_tag_formula`
--

CREATE TABLE `ims_tag_formula` (
                                   `id` int NOT NULL,
                                   `tag_id` int NOT NULL DEFAULT '0',
                                   `formula_id` int NOT NULL DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `ims_version`
--

CREATE TABLE `ims_version` (
                               `id` int NOT NULL,
                               `formula_id` int NOT NULL DEFAULT '0',
                               `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
                               `description` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
                               `publish_status` tinyint DEFAULT '0',
                               `publish_fail_reason` varchar(500) DEFAULT '',
                               `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `ims_w7panel_user`
--

CREATE TABLE `ims_w7panel_user` (
                                    `id` int NOT NULL,
                                    `w7panel_uid` varchar(128) NOT NULL,
                                    `w7panel_username` varchar(255) NOT NULL,
                                    `user_uid` int NOT NULL DEFAULT '0',
                                    `created_at` datetime NOT NULL,
                                    `updated_at` datetime DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- --------------------------------------------------------

--
-- 表的结构 `sessions`
--

CREATE TABLE `sessions` (
                            `id` varchar(191) NOT NULL,
                            `data` longtext,
                            `created_at` datetime(3) DEFAULT NULL,
                            `updated_at` datetime(3) DEFAULT NULL,
                            `expires_at` datetime(3) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

--
-- 转储表的索引
--

--
-- 表的索引 `ims_formula`
--
ALTER TABLE `ims_formula`
    ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `name` (`name`);

--
-- 表的索引 `ims_registry_namespace`
--
ALTER TABLE `ims_registry_namespace`
    ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `name` (`name`,`deleted_at`);

--
-- 表的索引 `ims_registry_repository`
--
ALTER TABLE `ims_registry_repository`
    ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `name` (`name`,`namespace`,`deleted_at`);

--
-- 表的索引 `ims_registry_repository_deploy_rule`
--
ALTER TABLE `ims_registry_repository_deploy_rule`
    ADD PRIMARY KEY (`id`);

--
-- 表的索引 `ims_registry_repository_deploy_rule_match_log`
--
ALTER TABLE `ims_registry_repository_deploy_rule_match_log`
    ADD PRIMARY KEY (`id`);

--
-- 表的索引 `ims_registry_repository_tag`
--
ALTER TABLE `ims_registry_repository_tag`
    ADD PRIMARY KEY (`id`);

--
-- 表的索引 `ims_registry_user`
--
ALTER TABLE `ims_registry_user`
    ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `username` (`username`);

--
-- 表的索引 `ims_registry_user_permission`
--
ALTER TABLE `ims_registry_user_permission`
    ADD PRIMARY KEY (`id`);

--
-- 表的索引 `ims_tag`
--
ALTER TABLE `ims_tag`
    ADD PRIMARY KEY (`id`);

--
-- 表的索引 `ims_tag_formula`
--
ALTER TABLE `ims_tag_formula`
    ADD PRIMARY KEY (`id`);

--
-- 表的索引 `ims_version`
--
ALTER TABLE `ims_version`
    ADD PRIMARY KEY (`id`);

--
-- 表的索引 `ims_w7panel_user`
--
ALTER TABLE `ims_w7panel_user`
    ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `w7panel_username` (`w7panel_username`);

--
-- 表的索引 `sessions`
--
ALTER TABLE `sessions`
    ADD PRIMARY KEY (`id`);

--
-- 在导出的表使用AUTO_INCREMENT
--

--
-- 使用表AUTO_INCREMENT `ims_formula`
--
ALTER TABLE `ims_formula`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_registry_namespace`
--
ALTER TABLE `ims_registry_namespace`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_registry_repository`
--
ALTER TABLE `ims_registry_repository`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_registry_repository_deploy_rule`
--
ALTER TABLE `ims_registry_repository_deploy_rule`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_registry_repository_deploy_rule_match_log`
--
ALTER TABLE `ims_registry_repository_deploy_rule_match_log`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_registry_repository_tag`
--
ALTER TABLE `ims_registry_repository_tag`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_registry_user`
--
ALTER TABLE `ims_registry_user`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_registry_user_permission`
--
ALTER TABLE `ims_registry_user_permission`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_tag`
--
ALTER TABLE `ims_tag`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_tag_formula`
--
ALTER TABLE `ims_tag_formula`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_version`
--
ALTER TABLE `ims_version`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;

--
-- 使用表AUTO_INCREMENT `ims_w7panel_user`
--
ALTER TABLE `ims_w7panel_user`
    MODIFY `id` int NOT NULL AUTO_INCREMENT;
COMMIT;

/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
