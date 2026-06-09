<template>
	<div style="min-height:99%;">
		<router-view :userInfo="userInfo"></router-view>
	</div>
</template>

<script>
import myAxios from "./utils";
export default {
	data() {
		return { userInfo: {} }
	},
	async created() {
		window.$wujie?.bus.$on("routeChange", (path) => {
			this.$router.push(path)
		});

		await this.getUserInfo()
	},
	methods: {
		getUserInfo() {
			myAxios.get("/v2/api/user/cur-user/info").then(res => {
				this.userInfo = res.data?.data?.user || {}
			});
		}
	}
}
</script>

<style>
:root {
	--color-text-1: #1d2129;
	--color-text-2: #4e5969;
	--color-text-3: #86909c;
	--color-text-4: #c9cdd4;
}

.create-item-menupopover {
	padding: 0 !important;
}

.cursor {
	cursor: pointer;
}

.content {
	padding: 20px;
}

.zpk-page-toolbar {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 16px;
}

.zpk-page-header {
	display: flex;
	align-items: center;
	min-height: 56px;
	padding: 0 24px;
	box-sizing: border-box;
	background: #ffffff;
}

.zpk-page-header-actions {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 10px;
	margin-left: auto;
}

.zpk-toolbar-left,
.zpk-toolbar-right {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	gap: 10px;
	min-width: 0;
}

.zpk-toolbar-right {
	justify-content: flex-end;
}

.zpk-page-toolbar .arco-btn+.arco-btn,
.zpk-toolbar-left .arco-btn+.arco-btn,
.zpk-toolbar-right .arco-btn+.arco-btn {
	margin-left: 0;
}

.arco-btn+.arco-btn {
	margin-left: 8px;
}

.arco-btn-group .arco-btn+.arco-btn {
	margin-left: 0;
}

.arco-btn.arco-btn-secondary.arco-btn-status-normal:not(.arco-btn-disabled) {
	color: var(--color-text-2) !important;
	background: #ffffff !important;
	border-color: #dcdfe6 !important;
}

.arco-btn.arco-btn-secondary.arco-btn-status-normal:not(.arco-btn-disabled):hover {
	color: #165dff !important;
	background: #ffffff !important;
	border-color: #165dff !important;
}

.arco-btn.arco-btn-secondary.arco-btn-status-normal:not(.arco-btn-disabled):active {
	color: #0e42d2 !important;
	background: #ffffff !important;
	border-color: #0e42d2 !important;
}

.arco-pagination-options {
	order: -1;
	margin-right: 8px;
	margin-left: 0 !important;
}

.arco-table {
	color: #1d2129;
}

.arco-table .arco-table-container,
.arco-table-border .arco-table-container,
.arco-table-border:not(.arco-table-border-cell) .arco-table-container {
	border: 0 !important;
}

.arco-table .arco-table-th,
.table-header .arco-table-th {
	color: #1d2129 !important;
	font-weight: 400 !important;
	background: #f2f3f5 !important;
	border-bottom: 1px solid #f2f3f5 !important;
}

.arco-table .arco-table-td {
	color: #1d2129;
	background: transparent;
	border-bottom: 1px solid #f2f3f5 !important;
}

.arco-table .arco-table-cell {
	min-height: 22px;
	padding: 10px 16px !important;
	line-height: 22px;
	font-size: 14px;
}

.arco-table-hover:not(.arco-table-dragging) .arco-table-tr:not(.arco-table-tr-empty):not(.arco-table-tr-summary):hover .arco-table-td:not(.arco-table-col-fixed-left):not(.arco-table-col-fixed-right),
.arco-table-hover .arco-table-tr-drag .arco-table-td:not(.arco-table-col-fixed-left):not(.arco-table-col-fixed-right) {
	background: #f7f8fa;
}

.arco-table .arco-table-td .arco-btn-text {
	height: auto;
	padding: 0 !important;
	color: #165dff !important;
	line-height: 22px;
	background: transparent !important;
	border: 0 !important;
}

.arco-table .arco-table-td .arco-btn-text:hover,
.arco-table .arco-table-td .arco-btn-text:focus {
	color: #165dff !important;
	background: transparent !important;
}

.arco-table .arco-table-td .arco-btn-text+.arco-btn-text {
	margin-left: 16px;
}

.arco-alert.zpk-primary-alert {
	color: var(--color-text-2) !important;
	background: #f2f7ff !important;
	padding: 20px 15px;
	border: 0 !important;
	align-items: start;
}

.zpk-primary-alert .arco-alert-body {
	align-items: flex-start;
}

.zpk-primary-alert.no-title .arco-alert-icon {
	margin-top: 0;
}

.zpk-primary-alert .arco-alert-title {
	color: var(--color-text-1) !important;
	font-size: 16px !important;
	font-weight: 500;
	line-height: 24px;
}

.zpk-primary-alert .arco-alert-content {
	color: var(--color-text-2) !important;
	font-size: 14px !important;
	line-height: 22px;
}

.zpk-primary-alert a,
.zpk-primary-alert .c-blue,
.zpk-primary-alert .credential-link {
	color: #165dff;
}

.zpk-modal-content {
	padding-top: 4px;
}

.import-zpk-content {
	display: flex;
	align-items: center;
}

.import-zpk-content .arco-auto-complete,
.import-zpk-content .arco-input-wrapper,
.import-zpk-content .arco-select-view {
	width: 100% !important;
}

.dialog-footer {
	display: flex;
	justify-content: flex-start;
	align-items: center;
	gap: 16px;
	margin: 12px 0 0;
	padding: 0 0 0 var(--zpk-modal-footer-offset, 0px);
	border-top: 0;
}

.dialog-footer .arco-btn {
	min-width: 88px;
	height: 36px;
	margin-left: 0 !important;
	font-size: 14px;
	order: 2;
}

.dialog-footer .arco-btn-primary {
	order: 1;
}

.arco-modal {
	max-width: calc(100vw - 32px);
}

.registry-detail-form+.dialog-footer,
.manifest-dialog-form+.dialog-footer {
	--zpk-modal-footer-offset: 80px;
}

.user-form+.dialog-footer {
	--zpk-modal-footer-offset: 90px;
}

.namespace-form+.dialog-footer {
	--zpk-modal-footer-offset: 92px;
}

.registry-form+.dialog-footer {
	--zpk-modal-footer-offset: 96px;
}

.respo-add-form+.dialog-footer {
	--zpk-modal-footer-offset: 52px;
}

.respo-import-form+.dialog-footer {
	--zpk-modal-footer-offset: 100px;
}

.file-name-dialog-footer {
	--zpk-modal-footer-offset: 90px;
}

.registry-build-form+.dialog-footer,
.version-create-footer {
	--zpk-modal-footer-offset: 120px;
}

.arco-modal .arco-modal-header {
	position: relative;
	padding: 30px 30px 16px;
	border-bottom: 0;
}

.arco-modal .arco-modal-title {
	color: var(--color-text-1);
	font-size: 22px;
	font-weight: 400;
	line-height: 32px;
}

.arco-modal .arco-modal-title-align-center {
	justify-content: flex-start !important;
}

.arco-modal .arco-modal-body {
	padding: 28px 30px 28px;
}

.arco-modal .arco-modal-close-btn {
	position: absolute !important;
	top: 16px !important;
	right: 16px !important;
	width: 28px;
	height: 28px;
	font-size: 18px;
	line-height: 28px;
}

.arco-breadcrumb {
	color: var(--color-text-3);
	font-size: 16px;
	line-height: 24px;
}

.arco-breadcrumb .arco-breadcrumb-item,
.arco-breadcrumb .arco-breadcrumb-item a {
	color: var(--color-text-3) !important;
	font-size: 16px;
	font-weight: 400;
}

.arco-breadcrumb .arco-breadcrumb-item {
	padding: 0 2px;
}

.arco-breadcrumb .arco-breadcrumb-item:last-child,
.arco-breadcrumb .arco-breadcrumb-item:last-child span {
	color: var(--color-text-1) !important;
	font-size: 16px;
}

.arco-breadcrumb .arco-breadcrumb-item-separator {
	display: inline-flex;
	width: auto;
	min-width: 0;
	padding: 0;
	margin: 0;
	color: var(--color-text-4);
	font-size: 16px;
	font-weight: 900;
}

.com-back {
	height: 56px;
	padding: 0 24px;
	background: #ffffff;
}

.backbtn {
	cursor: pointer;
}

.backicon,
.com-back .backicon,
.zpk-page-header .backicon {
	color: #0052D9;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	width: 20px;
	height: 20px;
	margin-right: 4px;
	font-size: 20px;
	line-height: 1;
}

.com-back .backbtn,
.zpk-page-header .backbtn {
	color: var(--color-text-1);
	font-size: 16px;
}

.cluster-box {
	padding: 10px 20px;
	height: 60px;
}

.refresh {
	margin: 0 16px;
	cursor: pointer;
}

.com-table-header {
	background-color: #F2F4F5;
}

@-webkit-keyframes rotation {
	from {
		-webkit-transform: rotate(0deg);
		transform: rotate(0deg);
	}

	to {
		-webkit-transform: rotate(360deg);
		transform: rotate(360deg);
	}
}

@keyframes rotation {
	from {
		-webkit-transform: rotate(0deg);
		transform: rotate(0deg);
	}

	to {
		-webkit-transform: rotate(360deg);
		transform: rotate(360deg);
	}
}

.my-rotate {
	animation: rotation 2s linear infinite;
	-moz-animation: rotation 2s linear infinite;
	-webkit-animation: rotation 2s linear infinite;
	-o-animation: rotation 2s linear infinite;
}

.registry-alert-item {
  position: relative;
  padding-left: 14px;
}

.registry-alert-item::before {
  position: absolute;
  top: 10px;
  left: 0;
  width: 4px;
  height: 4px;
  content: "";
  background: var(--color-text-2);
  border-radius: 50%;
}

.registry-alert-item .txt-line {
  color: #165dff;
  text-decoration-color: #165dff;
}

</style>
