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
                                                       `latest_trigger_at` datetime DEFAULT NULL
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
                                                                 `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
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
                                               `latest_push_at` datetime DEFAULT NULL
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
